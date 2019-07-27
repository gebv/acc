package accounts

import (
	"context"

	"github.com/gebv/acca/api"
	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/services/invoices"
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

func (s *Server) BalanceChanges(ctx context.Context, req *api.BalanceChangesRequest) (*api.BalanceChangesResponse, error) {

	list, err := s.db.SelectAllFrom(engine.ViewBalanceChangesView, "OFFSET $1 LIMIT $2", req.GetOffset(), req.GetLimit())
	if err != nil {
		return nil, errors.Wrap(err, "Failed get balance_changes.")
	}
	balanceChanges := make([]*api.BalanceChanges, 0, len(list))
	for _, v := range list {
		vbc := v.(*engine.ViewBalanceChanges)
		var operations []*api.BalanceChanges_Operation
		if vbc.Operations != nil {
			operations = make([]*api.BalanceChanges_Operation, 0, len(*vbc.Operations))
			for _, op := range *vbc.Operations {
				operations = append(operations, &api.BalanceChanges_Operation{
					OperId:    op.OperID,
					SrcAccId:  op.SrcAccID,
					DstAccId:  op.DstAccID,
					Strategy:  invoices.MapOperStrategyToApiTrStrategy[op.Strategy],
					Key:       op.Key,
					Hold:      op.Hold,
					HoldAccId: op.HoldAccID,
					Status:    invoices.MapOperStatusToApiTrStatus[op.Status],
				})
			}
		}
		balanceChanges = append(balanceChanges, &api.BalanceChanges{
			ChId:            vbc.ChID,
			TxId:            vbc.TxID,
			AccId:           vbc.AccID,
			Amount:          vbc.Amount,
			Balance:         vbc.Balance,
			BalanceAccepted: vbc.BalanceAccepted,
			Account: &api.BalanceChanges_Account{
				AccId:           vbc.Account.AccID,
				Key:             vbc.Account.Key,
				Balance:         vbc.Account.Balance,
				BalanceAccepted: vbc.Account.BalanceAccepted,
			},
			Currency: &api.BalanceChanges_Currency{
				CurrId: vbc.Currency.CurrID,
				Key:    vbc.Currency.Key,
			},
			Invoice: &api.BalanceChanges_Invoice{
				InvoiceId: vbc.Invoice.InvoiceID,
				Key:       vbc.Invoice.Key,
				Strategy:  vbc.Invoice.Strategy,
				Status:    invoices.MapInvStatusToApiInvStatus[vbc.Invoice.Status],
			},
			Transaction: &api.BalanceChanges_Transaction{
				TxId:               vbc.Transaction.TxID,
				Key:                vbc.Transaction.Key,
				Strategy:           vbc.Transaction.Strategy,
				Status:             invoices.MapTrStatusToApiTrStatus[vbc.Transaction.Status],
				Provider:           invoices.MapTrProviderToApiTrProvider[vbc.Transaction.Provider],
				ProviderOperId:     vbc.Transaction.ProviderOperID,
				ProviderOperStatus: vbc.Transaction.ProviderOperStatus,
				ProviderOperUrl:    vbc.Transaction.ProviderOperUrl,
			},
			Operations: operations,
		})
	}
	return &api.BalanceChangesResponse{
		BalanceChanges: balanceChanges,
	}, nil
}

var _ api.AccountsServer = (*Server)(nil)
