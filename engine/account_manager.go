package engine

import (
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/reform.v1"
)

var (
	ErrAccountExists = errors.New("account exists")
)

func NewAccountManager(db *reform.DB) *AccountManager {
	return &AccountManager{
		db:     db,
		logger: zap.L().Named("account_manager"),
	}
}

type AccountManager struct {
	db     *reform.DB
	logger *zap.Logger
}

// UpsertCurrency create or update currency by key.
func (m *AccountManager) UpsertCurrency(currencyName string, meta *[]byte) (currencyID int64, err error) {
	currencyName = formatCurrencyName(currencyName)

	newCurrency := &Currency{}
	if err := m.db.FindOneTo(newCurrency, "key", currencyName); err != nil {
		if err != reform.ErrNoRows {
			m.logger.Error("Failed find currency by key", zap.Error(err), zap.String("currency_key", currencyName))
			return 0, errors.Wrap(err, "failed find currency")
		}

		// not exists currency

		newCurrency = &Currency{
			Key: currencyName,
		}
	}

	// update meta
	newCurrency.Meta = meta

	if err := m.db.Save(newCurrency); err != nil {
		return 0, errors.Wrap(err, "failed update or create currency")
	}

	return newCurrency.CurrencyID, nil
}

// CreateAccount create new account.
func (m *AccountManager) CreateAccount(currencyID int64, accKey string, meta *[]byte) (accountID int64, err error) {
	newAccount := &Account{}
	err = m.db.SelectOneTo(newAccount, "WHERE curr_id = $1 AND key = $2", currencyID, accKey)
	if err == nil {
		return 0, ErrAccountExists
	}
	if err != reform.ErrNoRows {
		m.logger.Error("Failed find account by key", zap.Error(err), zap.String("account_key", accKey),
			zap.Int64("currency_id", currencyID),
		)
		return 0, errors.Wrap(err, "failed find account")
	}
	newAccount = &Account{
		CurrencyID: currencyID,
		Key:        accKey,
		Balance:    0,
		Meta:       meta,
	}

	if err := m.db.Insert(newAccount); err != nil {
		return 0, errors.Wrap(err, "failed create account")
	}
	return newAccount.AccountID, nil
}

func formatCurrencyName(currencyName string) string {
	return strings.TrimSpace(strings.ToLower(currencyName))
}
