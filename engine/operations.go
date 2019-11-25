package engine

import (
	"time"

	"github.com/pkg/errors"
)

//go:generate reform

// OperationStrategy тип обработки операции.
type OperationStrategy string

const (
	// SIMPLE_OPS простой внутренний перевод с SRC->DST
	SIMPLE_OPS OperationStrategy = "simple_transfer"

	// RECHARGE_OPS ввод в систему средст из внешней среду.
	RECHARGE_OPS OperationStrategy = "recharge"

	// WITHDRAW_OPS вывод из системы средств во внешнюю среду.
	WITHDRAW_OPS OperationStrategy = "withdraw"
)

var allowedOperationStrategies = map[OperationStrategy]bool{
	SIMPLE_OPS:   true,
	RECHARGE_OPS: true,
	WITHDRAW_OPS: true,
}

// OperationStatus Статус операции.
type OperationStatus string

func (s OperationStatus) Match(in OperationStatus) bool {
	return s == in
}

const (
	DRAFT_OP    OperationStatus = "draft"
	HOLD_OP     OperationStatus = "hold"
	ACCEPTED_OP OperationStatus = "accepted"
	REJECTED_OP OperationStatus = "rejected"
)

//reform:acca.operations
type Operation struct {
	// OperationID внутренний идентификатор операции.
	OperationID int64 `reform:"oper_id,pk" json:"oper_id"`

	// TransactionID связь с транзакцией.
	TransactionID int64 `reform:"tx_id" json:"tx_id"`

	// InvoiceID связь с инвойсом (денормализация).
	InvoiceID int64 `reform:"invoice_id" json:"invoice_id"`

	SrcAccID int64 `reform:"src_acc_id" json:"src_acc_id"`
	DstAccID int64 `reform:"dst_acc_id" json:"dst_acc_id"`

	// Hold признак определяющий требуется ли холдировать средства (2-х факторная операция).
	Hold bool `reform:"hold" json:"hold"`

	// HoldAccID идентификатор счета в котором отражается сумма заходированных средств.
	HoldAccID *int64 `reform:"hold_acc_id" json:"hold_acc_id"`

	// Strategy стратегия обработки операции.
	Strategy OperationStrategy `reform:"strategy" json:"strategy"`

	// Amount сумма операции.
	Amount int64 `reform:"amount" json:"amount"`

	// Key Уникальный идентификатор операции (опционально).
	Key *string `reform:"key" json:"key"`

	// Meta Мета-информация связанная с операцией (учавствующая в логике).
	Meta *[]byte `reform:"meta" json:"meta"`

	// Status Статус операции.
	Status    OperationStatus `reform:"status" json:"status"`
	UpdatedAt time.Time       `reform:"updated_at" json:"updated_at"`
	CreatedAt time.Time       `reform:"created_at" json:"created_at"`
}

func (o *Operation) BeforeInsert() error {
	o.UpdatedAt = time.Now()
	o.CreatedAt = time.Now()
	o.Status = DRAFT_OP
	if o.Strategy == OperationStrategy("") {
		return errors.New("empty strategy of operation")
	}
	return nil
}

func (o *Operation) BeforeUpdate() error {
	o.UpdatedAt = time.Now()
	return nil
}
