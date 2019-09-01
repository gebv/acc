package accounts

import (
	"context"
	"fmt"

	"github.com/gebv/acca/api"
	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/services"
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
	clientID := services.GetClient(ctx).ClientID

	m := engine.NewAccountManager(s.db)

	curr, err := m.GetCurrency(clientID, req.GetCurrencyKey())
	if err != nil {
		if err == engine.ErrCurrencyNotExists {
			return nil, api.MakeError(codes.NotFound, "currency not found")
		}
		if err == engine.ErrInvalidCurrencyKey {
			return nil, api.MakeError(codes.InvalidArgument, "Invalid key.")
		}
		return nil, errors.Wrapf(err, "Failed get currency %q.", req.GetKey())
	}

	accID, err := m.CreateAccount(clientID, curr.CurrencyID, req.GetKey(), req.GetMeta())

	if err != nil {
		if err == engine.ErrAccountExists {
			return nil, api.MakeError(codes.AlreadyExists, "Already exists.")
		}
		return nil, errors.Wrapf(err, "Failed create new account %q.", req.GetKey())
	}
	return &api.CreateAccountResponse{
		AccId: accID,
	}, nil
}

func (s *Server) GetCurrency(ctx context.Context, req *api.GetCurrencyRequest) (*api.GetCurrencyResponse, error) {
	clientID := services.GetClient(ctx).ClientID
	curr, err := engine.NewAccountManager(s.db).GetCurrency(clientID, req.GetKey())
	if err != nil {
		if err == engine.ErrCurrencyNotExists {
			return nil, api.MakeError(codes.NotFound, "currency not found")
		}
		if err == engine.ErrInvalidCurrencyKey {
			return nil, api.MakeError(codes.InvalidArgument, "Invalid key.")
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
	clientID := services.GetClient(ctx).ClientID
	err := engine.NewAccountManager(s.db).UpsertCurrency(clientID, req.GetKey(), req.GetMeta())
	if err != nil {
		if err == engine.ErrInvalidCurrencyKey {
			return nil, api.MakeError(codes.InvalidArgument, "Invalid key.")
		}
		return nil, errors.Wrapf(err, "Failed create new currency %q.", req.GetKey())
	}
	return &api.CreateCurrencyResponse{}, nil
}

func (s *Server) GetAccountByKey(ctx context.Context, req *api.GetAccountByKeyRequest) (*api.GetAccountByKeyResponse, error) {
	clientID := services.GetClient(ctx).ClientID

	m := engine.NewAccountManager(s.db)

	curr, err := m.GetCurrency(clientID, req.GetCurrKey())
	if err != nil {
		if err == engine.ErrCurrencyNotExists {
			return nil, api.MakeError(codes.NotFound, "currency not found")
		}
		if err == engine.ErrInvalidCurrencyKey {
			return nil, api.MakeError(codes.InvalidArgument, "Invalid key.")
		}
		return nil, errors.Wrapf(err, "Failed get currency %q.", req.GetKey())
	}

	acc, err := m.FindAccountByKey(clientID, curr.CurrencyID, req.GetKey())
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
	clientID := services.GetClient(ctx).ClientID

	var tail string
	args := make([]interface{}, 0, 4)
	if req.GetAccId() != nil {
		tail += "WHERE client_id = $1 AND acc_id = $2"
		args = append(args, clientID, req.GetAccId())
	}
	tail += fmt.Sprintf(" OFFSET $%d LIMIT $%d", len(args)+1, len(args)+2)
	args = append(args, req.GetOffset(), req.GetLimit())

	list, err := s.db.SelectAllFrom(engine.ViewBalanceChangesView, tail, args...)
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
					Amount:    op.Amount,
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
			CurrId:          vbc.CurrID,
			Amount:          vbc.Amount,
			Balance:         vbc.Balance,
			BalanceAccepted: vbc.BalanceAccepted,
			Invoice: &api.BalanceChanges_Invoice{
				InvoiceId: vbc.Invoice.InvoiceID,
				Key:       vbc.Invoice.Key,
				Meta:      vbc.Invoice.Meta,
				Strategy:  vbc.Invoice.Strategy,
				Status:    invoices.MapInvStatusToApiInvStatus[vbc.Invoice.Status],
			},
			Transaction: &api.BalanceChanges_Transaction{
				TxId:               vbc.Transaction.TxID,
				Key:                vbc.Transaction.Key,
				Meta:               vbc.Transaction.Meta,
				Strategy:           vbc.Transaction.Strategy,
				Status:             invoices.MapTrStatusToApiTrStatus[vbc.Transaction.Status],
				Provider:           invoices.MapTrProviderToApiTrProvider[vbc.Transaction.Provider],
				ProviderOperId:     vbc.Transaction.ProviderOperID,
				ProviderOperStatus: vbc.Transaction.ProviderOperStatus,
				ProviderOperUrl:    vbc.Transaction.ProviderOperUrl,
			},
			Operations: operations,
			ActualAccount: &api.BalanceChanges_Account{
				AccId:           vbc.Account.AccID,
				Key:             vbc.Account.Key,
				Balance:         vbc.Account.Balance,
				BalanceAccepted: vbc.Account.BalanceAccepted,
			},
			ActualTransaction: &api.BalanceChanges_Transaction{
				TxId:               vbc.ActualTransaction.TxID,
				Key:                vbc.ActualTransaction.Key,
				Meta:               vbc.ActualTransaction.Meta,
				Strategy:           vbc.ActualTransaction.Strategy,
				Status:             invoices.MapTrStatusToApiTrStatus[vbc.Transaction.Status],
				Provider:           invoices.MapTrProviderToApiTrProvider[vbc.Transaction.Provider],
				ProviderOperId:     vbc.ActualTransaction.ProviderOperID,
				ProviderOperStatus: vbc.ActualTransaction.ProviderOperStatus,
				ProviderOperUrl:    vbc.ActualTransaction.ProviderOperUrl,
			},
		})
	}
	return &api.BalanceChangesResponse{
		BalanceChanges: balanceChanges,
	}, nil
}

var _ api.AccountsServer = (*Server)(nil)
