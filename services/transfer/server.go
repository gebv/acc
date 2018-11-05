package transfer

import (
	"context"
	"database/sql"

	"github.com/gebv/acca/api/acca"
	"github.com/pkg/errors"
)

func NewServer(db *sql.DB) *Server {
	return &Server{db: db}
}

type Server struct {
	db *sql.DB
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
	panic("not implemented")
}

func (s *Server) GetTxByID(ctx context.Context, req *acca.GetTxByIDRequest) (*acca.GetTxByIDResponse, error) {
	panic("not implemented")
}

func (s *Server) RecentActivity(ctx context.Context, req *acca.RecentActivityRequest) (*acca.RecentActivityResponse, error) {
	panic("not implemented")
}

func (s *Server) MARecentActivity(ctx context.Context, req *acca.MARecentActivityRequest) (*acca.MARecentActivityResponse, error) {
	panic("not implemented")
}

var _ acca.TransferServer = (*Server)(nil)
