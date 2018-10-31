package accounts

import (
	"context"
	"database/sql"

	"github.com/lib/pq"

	"github.com/gebv/acca/api/acca"
	"github.com/pkg/errors"
)

func NewServer(db *sql.DB) *Server {
	return &Server{db: db}
}

type Server struct {
	db *sql.DB
}

func (s *Server) CreateAccount(ctx context.Context, req *acca.CreateAccountRequest) (*acca.CreateAccountResponse, error) {
	res := &acca.CreateAccountResponse{}
	err := s.db.QueryRow(`INSERT INTO acca.accounts(curr_id, key, meta) VALUES ($1, $2, $3) RETURNING acc_id`, req.GetCurrencyId(), req.GetKey(), MetaFrom(req.GetMeta())).Scan(&res.AccId)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed create new account %q.", req.GetKey())
	}
	return res, nil
}

func (s *Server) GetCurrencies(ctx context.Context, req *acca.GetCurrenciesRequest) (*acca.GetCurrenciesResponse, error) {
	rows, err := s.db.Query(`SELECT curr_id, key, meta FROM acca.currencies WHERE $1 @> key`, req.GetKey())
	if err != nil {
		return nil, errors.Wrapf(err, "Failed find currencies by key %v.", req.GetKey())
	}
	res := &acca.GetCurrenciesResponse{}
	defer rows.Close()
	for rows.Next() {
		row := acca.Currency{}
		m := new(Meta)
		err := rows.Scan(
			&row.CurrId,
			&row.Key,
			m,
		)
		if err != nil {
			return nil, errors.Wrap(err, "Failed find currencies - scan row.")
		}
		row.Meta = *m
		res.Currencies = append(res.Currencies, &row)
	}
	return res, nil
}

func (s *Server) CreateCurrency(ctx context.Context, req *acca.CreateCurrencyRequest) (*acca.CreateCurrencyResponse, error) {
	res := &acca.CreateCurrencyResponse{}
	err := s.db.QueryRow(`INSERT INTO acca.currencies(key, meta) VALUES ($1, $2) RETURNING curr_id`, req.GetKey(), MetaFrom(req.GetMeta())).Scan(&res.CurrencyId)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed create new currency %q.", req.GetKey())
	}
	return res, nil
}

func (s *Server) GetAccountsByIDs(ctx context.Context, req *acca.GetAccountsByIDsRequest) (*acca.GetAccountsByIDsResponse, error) {
	rows, err := s.db.Query(`SELECT acc_id, curr_id, key, balance, meta FROM acca.accounts WHERE acc_id = ANY($1)`, pq.Int64Array(req.GetAccIds()))
	if err != nil {
		return nil, errors.Wrapf(err, "Failed find accounts by ids %v.", req.GetAccIds())
	}
	res := &acca.GetAccountsByIDsResponse{}
	defer rows.Close()
	for rows.Next() {
		row := acca.Account{}
		m := new(Meta)
		err := rows.Scan(
			&row.AccId,
			&row.CurrId,
			&row.Key,
			&row.Balance,
			m,
		)
		if err != nil {
			return nil, errors.Wrap(err, "Failed find accounts - scan row.")
		}
		row.Meta = *m
		res.Accounts = append(res.Accounts, &row)
	}
	return res, nil
}

func (s *Server) GetAccountsByKey(ctx context.Context, req *acca.GetAccountsByKeyRequest) (*acca.GetAccountsByKeyResponse, error) {
	rows, err := s.db.Query(`SELECT acc_id, curr_id, key, balance, meta FROM acca.accounts WHERE $1 @> key`, req.GetKey())
	if err != nil {
		return nil, errors.Wrapf(err, "Failed find accounts by key %q.", req.GetKey())
	}
	res := &acca.GetAccountsByKeyResponse{}
	defer rows.Close()
	for rows.Next() {
		row := acca.Account{}
		err := rows.Scan(
			&row.AccId,
			&row.CurrId,
			&row.Key,
			&row.Balance,
			&row.Meta,
		)
		if err != nil {
			return nil, errors.Wrap(err, "Failed find accounts - scan row.")
		}
		res.Accounts = append(res.Accounts, &row)
	}
	return res, nil
}

func (s *Server) GetAccountsByUserID(ctx context.Context, req *acca.GetAccountsByUserIDRequest) (*acca.GetAccountsByUserIDResponse, error) {
	res := &acca.GetAccountsByUserIDResponse{
		UserAccounts: &acca.UserAccounts{},
	}
	// balances short info
	bsi := BalancesShortInfo{}

	err := s.db.QueryRow(`SELECT ma_balances FROM acca.ma_accounts WHERE user_id = $1`, req.GetUserId()).Scan(&bsi)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed find accounts by user ID %q.", req.GetUserId())
	}

	res.UserAccounts.UserId = req.UserId
	res.UserAccounts.Balances = convertBsiToAccaBsi(bsi)

	return res, nil
}

func convertBsiToAccaBsi(bsi BalancesShortInfo) []*acca.BalanceShortInfo {
	res := make([]*acca.BalanceShortInfo, len(bsi), len(bsi))
	for i, b := range bsi {
		res[i] = &acca.BalanceShortInfo{
			Type:    b.Type,
			Balance: b.Balance,
			AccId:   b.AccID,
		}
	}
	return res
}

var _ acca.AccountsServer = (*Server)(nil)
