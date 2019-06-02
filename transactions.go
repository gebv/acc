package acca

import (
	"encoding/json"
	"time"
)

//go:generate reform

type TransactionStatus string

const (
	DRAFT_TX    TransactionStatus = "draft"
	AUTH_TX     TransactionStatus = "auth"
	ACCEPTED_TX TransactionStatus = "accepted"
	REJECTED_TX TransactionStatus = "rejected"
	FAILED_TX   TransactionStatus = "FAILED"
)

//reform:acca.transactions
type Transaction struct {
	TransactionID      int64             `reform:"tx_id,pk"`
	InvoiceID          int64             `reform:"invoice_id"`
	Key                string            `reform:"key"`
	Provider           string            `reform:"provider"`
	ProviderOperID     *string           `reform:"provider_oper_id"`
	ProviderOperStatus *string           `reform:"provider_oper_status"`
	Meta               json.RawMessage   `reform:"meta"`
	Status             TransactionStatus `reform:"status"`
	UpdatedAt          time.Time         `reform:"updated_at"`
	CreatedAt          time.Time         `reform:"created_at"`
}

func (t *Transaction) BeforeInsert() error {
	t.UpdatedAt = time.Now()
	t.CreatedAt = time.Now()
	return nil
}

func (t *Transaction) BeforeUpdate() error {
	t.UpdatedAt = time.Now()
	return nil
}

type OperationStrategy string

const (
	SIMPLE_OPS   = "simple_transfer"
	RECHARGE_OPS = "recharge"
	WITHDRAW_OPS = "withdraw"
)

type OperationStatus string

const (
	DRAFT_OP    OperationStatus = "draft"
	HOLD_OP     OperationStatus = "hold"
	ACCEPTED_OP OperationStatus = "accepted"
	REJECTED_OP OperationStatus = "rejected"
)

//reform:acca.operations
type Operation struct {
	OperationID   int64             `reform:"oper_id,pk"`
	TransactionID int64             `reform:"tx_id"`
	SrcAccID      int64             `reform:"src_acc_id"`
	DstAccID      int64             `reform:"dst_acc_id"`
	HoldAccID     *int64            `reform:"hold_acc_id"`
	Strategy      OperationStrategy `reform:"strategy"`
	Amount        int64             `reform:"amount"`
	Key           string            `reform:"key"`
	Meta          json.RawMessage   `reform:"meta"`
	Status        OperationStatus   `reform:"status"`
	UpdatedAt     time.Time         `reform:"updated_at"`
	CreatedAt     time.Time         `reform:"created_at"`
}

func (o *Operation) BeforeInsert() error {
	o.UpdatedAt = time.Now()
	o.CreatedAt = time.Now()
	return nil
}

func (o *Operation) BeforeUpdate() error {
	o.UpdatedAt = time.Now()
	return nil
}
