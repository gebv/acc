package accounts

import (
	"context"

	"github.com/gebv/acca/api"
	"github.com/gebv/acca/engine"
	"google.golang.org/grpc/codes"
	"gopkg.in/reform.v1"

	"github.com/pkg/errors"
)

func NewServer(db *reform.DB) *Server {
	return &Server{db: db}
}

type Server struct {
	db *reform.DB
}

func (s *Server) CreateAccount(ctx context.Context, req *api.CreateAccountRequest) (*api.CreateAccountResponse, error) {
	m := engine.NewAccountManager(s.db)

	accID, err := m.CreateAccount(req.GetCurrencyId(), req.GetKey(), req.GetMeta())

	if err != nil {
		return nil, errors.Wrapf(err, "Failed create new account %q.", req.GetKey())
	}
	return &api.CreateAccountResponse{
		AccId: accID,
	}, nil
}

func (s *Server) GetCurrency(ctx context.Context, req *api.GetCurrencyRequest) (*api.GetCurrencyResponse, error) {
	curr, err := engine.NewAccountManager(s.db).GetCurrency(req.GetKey())
	if err != nil {
		if err == engine.ErrCurrencyNotExists {
			return nil, api.MakeError(codes.NotFound, "currency not found")
		}
		return nil, errors.Wrapf(err, "Failed get currency %q.", req.GetKey())
	}
	return &api.GetCurrencyResponse{
		Currency: &api.Currency{
			CurrId: curr.CurrencyID,
			Key:    curr.Key,
			Meta:   curr.Meta,
		},
	}, nil
}

func (s *Server) CreateCurrency(ctx context.Context, req *api.CreateCurrencyRequest) (*api.CreateCurrencyResponse, error) {
	currID, err := engine.NewAccountManager(s.db).UpsertCurrency(req.GetKey(), req.GetMeta())
	if err != nil {
		return nil, errors.Wrapf(err, "Failed create new currency %q.", req.GetKey())
	}
	return &api.CreateCurrencyResponse{
		CurrencyId: currID,
	}, nil
}

func (s *Server) GetAccountByKey(ctx context.Context, req *api.GetAccountByKeyRequest) (*api.GetAccountByKeyResponse, error) {
	m := engine.NewAccountManager(s.db)

	acc, err := m.FindAccountByKey(req.GetCurrId(), req.GetKey())
	if err != nil {
		if err == engine.ErrAccountNotExists {
			return nil, api.MakeError(codes.NotFound, "account not found")
		}
		return nil, errors.Wrap(err, "Failed find accounts - scan row.")
	}
	return &api.GetAccountByKeyResponse{
		Account: &api.Account{
			AccId:           acc.AccountID,
			CurrId:          acc.CurrencyID,
			Key:             acc.Key,
			Balance:         acc.Balance,
			Meta:            acc.Meta,
			BalanceAccepted: acc.BalanceAccepted,
		},
	}, nil
}

var _ api.AccountsServer = (*Server)(nil)
