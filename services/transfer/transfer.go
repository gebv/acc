package transfer

import (
	"context"
	"database/sql"

	"github.com/gebv/acca/api/acca"
)

type Server struct {
	db *sql.DB
}

func (s *Server) NewTransfer(ctx context.Context, req *acca.NewTransferRequest) (res *acca.NewTransferResponse, err error) {
	panic("not implemented")
}

func (s *Server) AcceptTx(ctx context.Context, req *acca.AcceptTxRequest) (res *acca.AcceptTxResponse, err error) {
	panic("not implemented")
}

func (s *Server) RejectTx(ctx context.Context, req *acca.RejectTxRequest) (res *acca.RejectTxResponse, err error) {
	panic("not implemented")
}

func (s *Server) RollbackTx(ctx context.Context, req *acca.RollbackTxRequest) (res *acca.RollbackTxResponse, err error) {
	panic("not implemented")
}

func (s *Server) HandleRequests(ctx context.Context, req *acca.HandleRequestsRequest) (res *acca.HandleRequestsResponse, err error) {
	panic("not implemented")
}

func (s *Server) GetUpdates(req *acca.GetUpdatesRequest, stream acca.Transfer_GetUpdatesServer) error {
	panic("not implemented")
}

var _ acca.TransferServer = (*Server)(nil)
