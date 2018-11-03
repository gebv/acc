package transfer

import (
	"context"
	"database/sql"
	"log"

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
	log.Println("DEBUG:")
	{
		dat, _ := opers.Value()
		log.Printf("%s\n", dat)
	}
	err := s.db.QueryRow(`SELECT acca.new_transfer($1, $2, $3)`, opers, req.GetReason(), meta).Scan(&res.TxId)
	if err != nil {
		return nil, errors.Wrap(err, "Failed created new transfer.")
	}
	return res, nil
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
