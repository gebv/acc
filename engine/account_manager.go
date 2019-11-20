package engine

// TODO: перенести в сервис

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/reform.v1"
)

var (
	ErrAccountExists      = errors.New("account exists")
	ErrAccountNotExists   = errors.New("account not exists")
	ErrCurrencyNotExists  = errors.New("currency not exists")
	ErrInvalidCurrencyKey = errors.New("currency not exists")
)

var IsKeyCurr = regexp.MustCompile(`^[a-z0-9]+$`).MatchString

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

// GetCurrency get currency by key.
func (m *AccountManager) GetCurrency(clientID int64, currencyName string) (*Currency, error) {
	currencyName = formatKey(currencyName)
	if !IsKeyCurr(currencyName) {
		return nil, ErrInvalidCurrencyKey
	}
	var curr Currency
	if err := m.db.SelectOneTo(&curr, "WHERE client_id = $1 AND key = $2", clientID, currencyName); err != nil {
		if err != reform.ErrNoRows {
			m.logger.Error(
				"Failed find currency by key",
				zap.Error(err),
				zap.Int64("client_id", clientID),
				zap.String("currency_key", currencyName),
			)
			return nil, ErrAccountNotExists
		}
		return nil, errors.Wrap(err, "Failed find currency by Key")
	}
	return &curr, nil
}

// UpsertCurrency create or update currency by key.
func (m *AccountManager) UpsertCurrency(clientID int64, currencyName string, meta *[]byte) error {
	currencyName = formatKey(currencyName)
	if !IsKeyCurr(currencyName) {
		return ErrInvalidCurrencyKey
	}
	newCurrency := &Currency{}
	if err := m.db.SelectOneTo(newCurrency, "WHERE client_id = $1 AND key = $2", clientID, currencyName); err != nil {
		if err != reform.ErrNoRows {
			m.logger.Error("Failed find currency by key",
				zap.Error(err),
				zap.Int64("client_id", clientID),
				zap.String("currency_key", currencyName),
			)
			return errors.Wrap(err, "failed find currency")
		}

		// not exists currency

		newCurrency = &Currency{
			ClientID: &clientID,
			Key:      currencyName,
		}
	}

	// update meta
	newCurrency.Meta = meta

	if err := m.db.Save(newCurrency); err != nil {
		return errors.Wrap(err, "failed update or create currency")
	}

	return nil
}

// CreateAccount create new account.
//
// Common errors:
// - ErrAccountExists - exists account
// - other errors
func (m *AccountManager) CreateAccount(clientID, currencyID int64, accKey string, meta *[]byte) (accountID int64, err error) {
	accKey = formatKey(accKey)

	newAccount := &Account{}
	err = m.db.SelectOneTo(newAccount, "WHERE client_id = $1 AND curr_id = $2 AND key = $3", clientID, currencyID, accKey)
	if err == nil {
		return 0, ErrAccountExists
	}
	if err != reform.ErrNoRows {
		m.logger.Error("Failed find account by key",
			zap.Error(err),
			zap.Int64("client_id", clientID),
			zap.String("account_key", accKey),
			zap.Int64("currency_id", currencyID),
		)
		return 0, errors.Wrap(err, "failed find account")
	}
	newAccount = &Account{
		ClientID:   &clientID,
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

// FindAccountByKey returns account by key.
//
// Common errors:
// - ErrAccountNotExists - not found account
// - other errors
func (m *AccountManager) FindAccountByKey(clientID, currencyID int64, accKey string) (*Account, error) {
	accKey = formatKey(accKey)

	foundAccount := &Account{}
	err := m.db.SelectOneTo(foundAccount, "WHERE client_id = $1 AND curr_id = $2 AND key = $3", clientID, currencyID, accKey)
	if err != nil {
		if err == reform.ErrNoRows {
			return nil, ErrAccountNotExists
		}
		return nil, errors.Wrap(err, "failed find account")
	}
	return foundAccount, nil
}

func formatKey(currencyName string) string {
	return strings.ReplaceAll(strings.TrimSpace(strings.ToLower(currencyName)), "-", "__")
}
