package engine

import (
	"time"

	"github.com/pkg/errors"
)

//go:generate reform

//reform:acca.transactions
type Transaction struct {
	// TransactionID внутренний идентификатор транзакции.
	TransactionID int64 `reform:"tx_id,pk"`

	// InvoiceID связанный с транзакцией инвойс.
	InvoiceID int64 `reform:"invoice_id"`

	// Key Уникальный внешний идентификатор транзакции (опционально).
	Key *string `reform:"key"`

	Amount int64 `reform:"amount"`

	// Strategy стратегия работы с инвойсом.
	Strategy string `reform:"strategy"`

	// Provider Тип провайдера обслуживающий транзакцию.
	Provider Provider `reform:"provider"`

	// ProviderOperID Идентификатор связанной с транзакцией операции во внешней системе.
	ProviderOperID *string `reform:"provider_oper_id"`

	// ProviderOperStatus Статус связанной с транзакцией операции во внешней системе.
	ProviderOperStatus *string `reform:"provider_oper_status"`

	// ProviderOperStatus Статус связанной с транзакцией операции во внешней системе.
	ProviderOperUrl *string `reform:"provider_oper_url"`

	// Meta мета информация связанная с транзакцией (учавствующая в логике).
	Meta *[]byte `reform:"meta"`

	// Status статус транзакции.
	Status TransactionStatus `reform:"status"`

	// Status статус транзакции куда происходит переход.
	NextStatus *TransactionStatus `reform:"next_status"`

	UpdatedAt time.Time `reform:"updated_at"`
	CreatedAt time.Time `reform:"created_at"`
}

func (t *Transaction) BeforeInsert() error {
	t.UpdatedAt = time.Now()
	t.CreatedAt = time.Now()
	t.Status = DRAFT_TX
	if t.Provider == UNKNOWN_PROVIDER {
		return errors.New("unknown provider")
	}
	return nil
}

func (t *Transaction) BeforeUpdate() error {
	t.UpdatedAt = time.Now()
	return nil
}

type Provider string

func (p Provider) Match(in Provider) bool {
	return p == in
}

const (
	UNKNOWN_PROVIDER Provider = ""
	INTERNAL         Provider = "internal"
	SBERBANK         Provider = "sberbank"
)

type TransactionStatus string

func (s TransactionStatus) Match(in TransactionStatus) bool {
	return s == in
}

const (
	DRAFT_TX     TransactionStatus = "draft"
	AUTH_TX      TransactionStatus = "auth"
	WAUTH_TX     TransactionStatus = "auth_wait"
	HOLD_TX      TransactionStatus = "hold"
	WHOLD_TX     TransactionStatus = "hold_wait"
	ACCEPTED_TX  TransactionStatus = "accepted"
	WACCEPTED_TX TransactionStatus = "accepted_wait"
	REJECTED_TX  TransactionStatus = "rejected"
	WREJECTED_TX TransactionStatus = "rejected_wait"
	FAILED_TX    TransactionStatus = "failed"
)
