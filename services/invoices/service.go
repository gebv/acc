package invoices

import (
	"context"
	"strings"

	"github.com/gebv/acca/api"
	"github.com/gebv/acca/engine"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

func NewServer(db *reform.DB, nc *nats.Conn) *Server {
	return &Server{
		db: db,
		nc: nc,
	}
}

type Server struct {
	db *reform.DB
	nc *nats.Conn
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
	return &api.GetInvoiceByIDResponse{
		Invoice: &api.Invoice{
			InvoiceId:  inv.InvoiceID,
			Key:        inv.Key,
			Status:     mapInvStatusToApiInvStatus[inv.Status],
			NextStatus: mapInvStatusToApiInvStatus[inv.NextStatus],
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

	tr := &engine.Transaction{
		InvoiceID: req.GetInvoiceId(),
		Strategy:  strategy,
		Meta:      req.GetMeta(),
		Status:    engine.DRAFT_TX,
		// TODO заполнить провайдер, по идее из стратегии
	}
	if req.GetKey() != nil {
		key := strings.TrimSpace(strings.ToLower(*req.GetKey()))
		tr.Key = &key
	}
	if err := s.db.Insert(tr); err != nil {
		return nil, errors.Wrap(err, "failed insert new transaction")
	}
	return &api.AddTransactionToInvoiceResponse{TxId: tr.TransactionID}, nil
}

func (s Server) GetTransactionByID(ctx context.Context, req *api.GetTransactionByIDRequest) (*api.GetTransactionByIDResponse, error) {
	tr := engine.Transaction{TransactionID: req.GetTxId()}
	if err := s.db.Reload(&tr); err != nil {
		return nil, errors.Wrap(err, "Failed get transaction by ID.")
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
			NextStatus:         mapTrStatusToApiTrStatus[tr.NextStatus],
			CreatedAt:          &tr.CreatedAt,
			UpdatedAt:          &tr.UpdatedAt,
		},
	}, nil
}

func (s Server) AddOperationToTx(ctx context.Context, req *api.AddOperationToTxRequest) (*api.AddOperationToTxResponse, error) {
	panic("implement me")
}

func (s Server) GetOperationByID(ctx context.Context, req *api.GetOperationByIDRequest) (*api.GetOperationByIDResponse, error) {
	panic("implement me")
}

func (s Server) AuthInvoice(ctx context.Context, req *api.AuthInvoiceRequest) (*api.AuthInvoiceResponse, error) {
	panic("implement me")
}

func (s Server) AcceptInvoice(ctx context.Context, req *api.AcceptInvoiceRequest) (*api.AcceptInvoiceResponse, error) {
	panic("implement me")
}

func (s Server) RejectInvoice(ctx context.Context, req *api.RejectInvoiceRequest) (*api.RejectInvoiceResponse, error) {
	panic("implement me")
}
