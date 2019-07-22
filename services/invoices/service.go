package invoices

import (
	"context"
	"strings"
	"time"

	"github.com/gebv/acca/api"
	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/engine/strategies"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

func NewServer(db *reform.DB, nc *nats.EncodedConn) *Server {
	return &Server{
		db: db,
		nc: nc,
	}
}

type Server struct {
	db *reform.DB
	nc *nats.EncodedConn
}

func (s Server) NewInvoice(ctx context.Context, req *api.NewInvoiceRequest) (*api.NewInvoiceResponse, error) {
	key := strings.TrimSpace(strings.ToLower(req.GetKey()))
	strategy := strings.TrimSpace(strings.ToLower(req.GetStrategy()))

	inv := &engine.Invoice{
		Key:      key,
		Strategy: strategy,
		Status:   engine.DRAFT_I,
		Meta:     req.GetMeta(),
		Payload:  nil,
	}
	if err := s.db.Insert(inv); err != nil {
		return nil, errors.Wrap(err, "failed insert new invoice")
	}
	return &api.NewInvoiceResponse{InvoiceId: inv.InvoiceID}, nil
}

func (s Server) GetInvoiceByID(ctx context.Context, req *api.GetInvoiceByIDRequest) (*api.GetInvoiceByIDResponse, error) {
	inv := engine.Invoice{InvoiceID: req.GetInvoiceId()}
	if err := s.db.Reload(&inv); err != nil {
		return nil, errors.Wrap(err, "Failed get invoice by ID.")
	}
	var nextStatus api.InvoiceStatus
	if inv.NextStatus != nil {
		nextStatus = mapInvStatusToApiInvStatus[*inv.NextStatus]
	}
	return &api.GetInvoiceByIDResponse{
		Invoice: &api.Invoice{
			InvoiceId:  inv.InvoiceID,
			Key:        inv.Key,
			Status:     mapInvStatusToApiInvStatus[inv.Status],
			NextStatus: nextStatus,
			Strategy:   inv.Strategy,
			Meta:       inv.Meta,
			CreatedAt:  &inv.CreatedAt,
			UpdatedAt:  &inv.UpdatedAt,
		},
	}, nil
}

func (s Server) AddTransactionToInvoice(ctx context.Context, req *api.AddTransactionToInvoiceRequest) (*api.AddTransactionToInvoiceResponse, error) {
	invoice := &engine.Invoice{InvoiceID: req.GetInvoiceId()}
	if err := s.db.Reload(invoice); err != nil {
		return nil, errors.Wrap(err, "failed find invocie by ID")
	}
	if !invoice.Status.Match(engine.DRAFT_I) {
		return nil, errors.New("not allowed add transaction (invoice is not draft)")
	}

	strategy := strings.TrimSpace(strings.ToLower(req.GetStrategy()))
	var txID int64
	err := s.db.InTransaction(func(tx *reform.TX) error {
		tr := &engine.Transaction{
			InvoiceID: req.GetInvoiceId(),
			Strategy:  strategy,
			Key:       req.GetKey(),
			Meta:      req.GetMeta(),
			Status:    engine.DRAFT_TX,
			// TODO заполнить провайдер, по идее из стратегии
			Provider: engine.INTERNAL,
		}
		if req.GetKey() != nil {
			key := strings.TrimSpace(strings.ToLower(*req.GetKey()))
			tr.Key = &key
		}
		if err := tx.Insert(tr); err != nil {
			return errors.Wrap(err, "failed insert new transaction")
		}
		txID = tr.TransactionID
		for _, op := range req.GetOperations() {
			o := engine.Operation{
				TransactionID: tr.TransactionID,
				InvoiceID:     tr.InvoiceID,
				SrcAccID:      op.GetSrcAccId(),
				DstAccID:      op.GetDstAccId(),
				Hold:          op.GetHold(),
				HoldAccID:     op.GetHoldAccId(),
				Strategy:      mapStrategyOperFromApiStrategyOper[op.GetStrategy()],
				Amount:        op.GetAmount(),
				Key:           op.GetKey(),
				Meta:          op.GetMeta(),
				Status:        engine.DRAFT_OP,
				UpdatedAt:     time.Now(),
				CreatedAt:     time.Now(),
			}
			if err := tx.Insert(&o); err != nil {
				return errors.Wrap(err, "Failed insert new operation")
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &api.AddTransactionToInvoiceResponse{TxId: txID}, nil
}

func (s Server) GetTransactionByID(ctx context.Context, req *api.GetTransactionByIDRequest) (*api.GetTransactionByIDResponse, error) {
	tr := engine.Transaction{TransactionID: req.GetTxId()}
	if err := s.db.Reload(&tr); err != nil {
		return nil, errors.Wrap(err, "Failed get transaction by ID.")
	}
	var nextStatus api.TxStatus
	if tr.NextStatus != nil {
		nextStatus = mapTrStatusToApiTrStatus[*tr.NextStatus]
	}
	list, err := s.db.SelectAllFrom(engine.OperationTable, "WHERE tx_id = $1", tr.TransactionID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed get operation by transaction ID.")
	}
	opers := make([]*api.Oper, 0, len(list))
	for _, v := range list {
		op := v.(*engine.Operation)
		opers = append(opers, &api.Oper{
			OperId:    op.OperationID,
			InvoiceId: op.InvoiceID,
			TxId:      op.TransactionID,
			SrcAccId:  op.SrcAccID,
			Hold:      op.Hold,
			HoldAccId: op.HoldAccID,
			DstAccId:  op.DstAccID,
			Strategy:  mapOperStrategyToApiTrStrategy[op.Strategy],
			Amount:    op.Amount,
			Key:       op.Key,
			Meta:      op.Meta,
			Status:    mapOperStatusToApiTrStatus[op.Status],
			CreatedAt: &op.CreatedAt,
			UpdatedAt: &op.UpdatedAt,
		})
	}
	return &api.GetTransactionByIDResponse{
		Tx: &api.Tx{
			TxId:               tr.TransactionID,
			InvoiceId:          tr.InvoiceID,
			Key:                tr.Key,
			Strategy:           tr.Strategy,
			Provider:           mapTrProviderToApiTrProvider[tr.Provider],
			ProviderOperId:     tr.ProviderOperID,
			ProviderOperStatus: tr.ProviderOperStatus,
			Meta:               tr.Meta,
			Status:             mapTrStatusToApiTrStatus[tr.Status],
			NextStatus:         nextStatus,
			CreatedAt:          &tr.CreatedAt,
			UpdatedAt:          &tr.UpdatedAt,
			Operations:         opers,
		},
	}, nil
}

func (s Server) AuthInvoice(ctx context.Context, req *api.AuthInvoiceRequest) (*api.AuthInvoiceResponse, error) {
	invoice := &engine.Invoice{InvoiceID: req.GetInvoiceId()}
	if err := s.db.Reload(invoice); err != nil {
		return nil, errors.Wrap(err, "failed find invocie by ID")
	}
	if !invoice.Status.Match(engine.DRAFT_I) {
		return nil, errors.New("not transition to auth (invoice is not draft)")
	}
	err := s.nc.Publish(strategies.UPDATE_INVOICE_SUBJECT, strategies.MessageUpdateInvoice{
		InvoiceID: invoice.InvoiceID,
		Strategy:  invoice.Strategy,
		Status:    engine.AUTH_I,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed publish update status invoice")
	}
	return &api.AuthInvoiceResponse{}, nil
}

func (s Server) AcceptInvoice(ctx context.Context, req *api.AcceptInvoiceRequest) (*api.AcceptInvoiceResponse, error) {
	invoice := &engine.Invoice{InvoiceID: req.GetInvoiceId()}
	if err := s.db.Reload(invoice); err != nil {
		return nil, errors.Wrap(err, "failed find invocie by ID")
	}
	err := s.nc.Publish(strategies.UPDATE_INVOICE_SUBJECT, strategies.MessageUpdateInvoice{
		InvoiceID: invoice.InvoiceID,
		Strategy:  invoice.Strategy,
		Status:    engine.MACCEPTED_I,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed publish update status invoice")
	}
	return &api.AcceptInvoiceResponse{}, nil
}

func (s Server) RejectInvoice(ctx context.Context, req *api.RejectInvoiceRequest) (*api.RejectInvoiceResponse, error) {
	invoice := &engine.Invoice{InvoiceID: req.GetInvoiceId()}
	if err := s.db.Reload(invoice); err != nil {
		return nil, errors.Wrap(err, "failed find invocie by ID")
	}
	err := s.nc.Publish(strategies.UPDATE_INVOICE_SUBJECT, strategies.MessageUpdateInvoice{
		InvoiceID: invoice.InvoiceID,
		Strategy:  invoice.Strategy,
		Status:    engine.MREJECTED_I,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed publish update status invoice")
	}
	return &api.RejectInvoiceResponse{}, nil
}

func (s Server) AuthTx(ctx context.Context, req *api.AuthTxRequest) (*api.AuthTxResponse, error) {
	tx := &engine.Transaction{TransactionID: req.GetTxId()}
	if err := s.db.Reload(tx); err != nil {
		return nil, errors.Wrap(err, "failed find transaction by ID")
	}
	if !tx.Status.Match(engine.DRAFT_TX) {
		return nil, errors.New("not transition to auth (transaction is not draft)")
	}
	err := s.nc.Publish(strategies.UPDATE_TRANSACTION_SUBJECT, strategies.MessageUpdateTransaction{
		TransactionID: tx.TransactionID,
		Strategy:      tx.Strategy,
		Status:        engine.AUTH_TX,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed publish update status invoice")
	}
	return &api.AuthTxResponse{}, nil
}

func (s Server) AcceptTx(ctx context.Context, req *api.AcceptTxRequest) (*api.AcceptTxResponse, error) {
	tx := &engine.Transaction{TransactionID: req.GetTxId()}
	if err := s.db.Reload(tx); err != nil {
		return nil, errors.Wrap(err, "failed find transaction by ID")
	}
	err := s.nc.Publish(strategies.UPDATE_TRANSACTION_SUBJECT, strategies.MessageUpdateTransaction{
		TransactionID: tx.TransactionID,
		Strategy:      tx.Strategy,
		Status:        engine.ACCEPTED_TX,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed publish update status invoice")
	}
	return &api.AcceptTxResponse{}, nil
}

func (s Server) RejectTx(ctx context.Context, req *api.RejectTxRequest) (*api.RejectTxResponse, error) {
	tx := &engine.Transaction{TransactionID: req.GetTxId()}
	if err := s.db.Reload(tx); err != nil {
		return nil, errors.Wrap(err, "failed find transaction by ID")
	}
	err := s.nc.Publish(strategies.UPDATE_TRANSACTION_SUBJECT, strategies.MessageUpdateTransaction{
		TransactionID: tx.TransactionID,
		Strategy:      tx.Strategy,
		Status:        engine.REJECTED_TX,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed publish update status invoice")
	}
	return &api.RejectTxResponse{}, nil
}

var mapInvStatusToApiInvStatus = map[engine.InvoiceStatus]api.InvoiceStatus{
	engine.DRAFT_I:     api.InvoiceStatus_DRAFT_I,
	engine.AUTH_I:      api.InvoiceStatus_AUTH_I,
	engine.WAIT_I:      api.InvoiceStatus_WAIT_I,
	engine.ACCEPTED_I:  api.InvoiceStatus_ACCEPTED_I,
	engine.MACCEPTED_I: api.InvoiceStatus_MACCEPTED_I,
	engine.REJECTED_I:  api.InvoiceStatus_REJECTED_I,
	engine.MREJECTED_I: api.InvoiceStatus_MREJECTED_I,
}

var mapStrategyOperFromApiStrategyOper = map[api.OperStrategy]engine.OperationStrategy{
	api.OperStrategy_SIMPLE_OPS:   engine.SIMPLE_OPS,
	api.OperStrategy_RECHARGE_OPS: engine.RECHARGE_OPS,
	api.OperStrategy_WITHDRAW_OPS: engine.WITHDRAW_OPS,
}

var mapOperStrategyToApiTrStrategy = map[engine.OperationStrategy]api.OperStrategy{
	engine.SIMPLE_OPS:   api.OperStrategy_SIMPLE_OPS,
	engine.RECHARGE_OPS: api.OperStrategy_RECHARGE_OPS,
	engine.WITHDRAW_OPS: api.OperStrategy_WITHDRAW_OPS,
}

var mapTrProviderToApiTrProvider = map[engine.Provider]api.Provider{
	engine.INTERNAL: api.Provider_INTERNAL_PROVIDER,
	engine.SBERBANK: api.Provider_SBERBANK_PROVIDER,
}

var mapTrStatusToApiTrStatus = map[engine.TransactionStatus]api.TxStatus{
	engine.DRAFT_TX:     api.TxStatus_DRAFT_TX,
	engine.AUTH_TX:      api.TxStatus_AUTH_TX,
	engine.WAUTH_TX:     api.TxStatus_AUTH_TX,
	engine.HOLD_TX:      api.TxStatus_HOLD_TX,
	engine.WHOLD_TX:     api.TxStatus_WHOLD_TX,
	engine.ACCEPTED_TX:  api.TxStatus_ACCEPTED_TX,
	engine.WACCEPTED_TX: api.TxStatus_WACCEPTED_TX,
	engine.REJECTED_TX:  api.TxStatus_REJECTED_TX,
	engine.WREJECTED_TX: api.TxStatus_WREJECTED_TX,
	engine.FAILED_TX:    api.TxStatus_FAILED_TX,
}

var mapOperStatusToApiTrStatus = map[engine.OperationStatus]api.OperStatus{
	engine.DRAFT_OP:    api.OperStatus_DRAFT_OP,
	engine.HOLD_OP:     api.OperStatus_HOLD_OP,
	engine.ACCEPTED_OP: api.OperStatus_ACCEPTED_OP,
	engine.REJECTED_OP: api.OperStatus_REJECTED_OP,
}
