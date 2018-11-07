package transfer

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/lib/pq"

	"github.com/gebv/acca/api/acca"
	"github.com/gebv/acca/services/accounts"
	"github.com/pkg/errors"
)

func NewServer(db *sql.DB, dbl *pq.Listener) *Server {
	h := &hub{dbl: dbl}
	go h.run(context.Background())
	return &Server{db: db, hub: h}
}

type Server struct {
	db  *sql.DB
	hub *hub
}

func (s *Server) NewTransfer(ctx context.Context, req *acca.NewTransferRequest) (*acca.NewTransferResponse, error) {
	res := &acca.NewTransferResponse{}
	opers := pgOpers(req.Opers)
	meta := MetaFrom(req.Meta)
	err := s.db.QueryRow(`SELECT acca.new_transfer($1, $2, $3)`, opers, req.GetReason(), meta).Scan(&res.TxId)
	if err != nil {
		return nil, errors.Wrap(err, "Failed created new transfer.")
	}
	return res, nil
}

func (s *Server) AcceptTx(ctx context.Context, req *acca.AcceptTxRequest) (*acca.AcceptTxResponse, error) {
	_, err := s.db.Exec(`SELECT acca.accept_tx($1)`, req.TxId)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed reject transaction %d.", req.TxId)
	}
	return &acca.AcceptTxResponse{}, nil
}

func (s *Server) RejectTx(ctx context.Context, req *acca.RejectTxRequest) (*acca.RejectTxResponse, error) {
	_, err := s.db.Exec(`SELECT acca.reject_tx($1)`, req.TxId)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed reject transaction %d.", req.TxId)
	}
	return &acca.RejectTxResponse{}, nil
}

func (s *Server) RollbackTx(ctx context.Context, req *acca.RollbackTxRequest) (*acca.RollbackTxResponse, error) {
	_, err := s.db.Exec(`SELECT acca.rollback_tx($1)`, req.TxId)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed rollback transaction %d.", req.TxId)
	}
	return &acca.RollbackTxResponse{}, nil
}

func (s *Server) HandleRequests(ctx context.Context, req *acca.HandleRequestsRequest) (*acca.HandleRequestsResponse, error) {
	res := &acca.HandleRequestsResponse{}
	err := s.db.QueryRow(`SELECT t.ok, t.err FROM acca.handle_requests($1) t;`, req.Limit).Scan(&res.NumOk, &res.NumErr)
	if err != nil {
		return nil, errors.Wrap(err, "Failed handler requests from queue.")
	}
	return res, nil
}

func (s *Server) GetUpdates(req *acca.GetUpdatesRequest, stream acca.Transfer_GetUpdatesServer) error {
	subID, ch := s.hub.subscribe()
	log.Println("Successful subscribe", subID)
	defer func() {
		s.hub.unsubscribe(subID)
		log.Println("Successful unsubscribe", subID)
	}()

	for u := range ch {
		if err := stream.Send(u); err != nil {
			return errors.Wrap(err, "Failed to send update.")
		}
	}
	return nil
}

func (s *Server) GetTxByID(ctx context.Context, req *acca.GetTxByIDRequest) (*acca.GetTxByIDResponse, error) {
	qFindTx := `SELECT
		tx_id,
		reason,
		meta,
		status,
		errm,
		created_at,
		updated_at
	FROM acca.transactions WHERE tx_id = $1`

	tx := &acca.Tx{}
	{
		m := new(Meta)
		var status string
		var errm *string
		err := s.db.QueryRow(qFindTx, req.TxId).Scan(
			&tx.TxId,
			&tx.Reason,
			m,
			&status,
			&errm,
			&tx.CreatedAt,
			&tx.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed scan tx by ID=%d.", req.TxId)
		}
		tx.Meta = *m
		tx.Status = acca.TxStatus(acca.TxStatus_value[strings.ToUpper(status)])
		if errm != nil {
			tx.Errm = *errm
		}
	}

	opers := []*acca.Oper{}
	if req.WithOpers {
		qFindOpers := `SELECT
			oper_id,
			tx_id,
			src_acc_id,
			dst_acc_id,
			type,
			amount,
			reason,
			meta,
			hold,
			hold_acc_id,
			status,
			created_at,
			updated_at
		FROM acca.operations WHERE tx_id = $1`

		rows, err := s.db.Query(qFindOpers, req.TxId)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed find opers by tx ID=%d.", req.TxId)
		}
		for rows.Next() {
			row := &acca.Oper{}
			m := new(Meta)
			var status string
			var holdAccID *int64
			var operType string
			err := rows.Scan(
				&row.OperId,
				&row.TxId,
				&row.SrcAccId,
				&row.DstAccId,
				&operType,
				&row.Amount,
				&row.Reason,
				m,
				&row.Hold,
				&holdAccID,
				&status,
				&row.CreatedAt,
				&row.UpdatedAt,
			)
			if err != nil {
				panic(errors.Wrapf(err, "Failed scan opers by tx ID=%d.", req.TxId)) // TODO: err log
				break
			}
			row.Status = acca.OperStatus(acca.OperStatus_value[strings.ToUpper(status)])
			row.Type = acca.OperType(acca.OperType_value[strings.ToUpper(operType)])
			row.Meta = *m
			if holdAccID != nil {
				row.HoldAccId = *holdAccID
			}

			opers = append(opers, row)
		}
		if rows.Err() != nil {
			return nil, errors.Wrapf(rows.Err(), "Failed scan opers by tx ID=%d.", req.TxId)
		}
	}

	return &acca.GetTxByIDResponse{
		Tx:    tx,
		Opers: opers,
	}, nil
}

func (s *Server) RecentActivity(ctx context.Context, req *acca.RecentActivityRequest) (*acca.RecentActivityResponse, error) {
	query := `SELECT
		id,
		oper_id,
		acc_id,
		amount,
		balance,
		ma_balances,
		tx_id,
		src_acc_id,
		dst_acc_id,
		reason,
		tx_reason,
		acc_key,
		acc_curr_id,
		acc_curr_key

	FROM acca.recent_activity`
	args := []interface{}{}
	if req.LastId > 0 {
		args = append(args, req.LastId)
		query += fmt.Sprintf(` WHERE id < $%d`, len(args))
	}
	query += ` ORDER BY id DESC`
	if req.Limit > 0 {
		if req.Limit > 50 {
			req.Limit = 50
		}
	} else {
		req.Limit = 50
	}
	args = append(args, req.Limit)
	query += fmt.Sprintf(` LIMIT $%d`, len(args))

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "Failed get recent activity.")
	}
	res := &acca.RecentActivityResponse{}
	defer rows.Close()
	for rows.Next() {
		row := acca.RecentActivity{}
		var maBalances accounts.BalancesShortInfo
		err := rows.Scan(
			&row.Id,
			&row.OperId,
			&row.AccId,
			&row.Amount,
			&row.Balance,
			&maBalances,
			&row.TxId,
			&row.SrcAccId,
			&row.DstAccId,
			&row.Reason,
			&row.TxReason,
			&row.AccKey,
			&row.AccCurrId,
			&row.AccCurrKey,
		)
		if err != nil {
			return nil, errors.Wrap(err, "Failed scan row.")
		}
		if len(maBalances) > 0 {
			row.MaBalances = maBalances
		}

		res.List = append(res.List, &row)
	}

	if rows.Err() != nil {
		return nil, errors.Wrap(err, "Failed scan row.")
	}

	return res, nil
}

var _ acca.TransferServer = (*Server)(nil)
