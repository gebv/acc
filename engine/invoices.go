package engine

import (
	"time"
)

//go:generate reform

type InvoiceStatus string

func (s InvoiceStatus) Match(in InvoiceStatus) bool {
	return s == in
}

const (
	DRAFT_I    InvoiceStatus = "draft"
	AUTH_I     InvoiceStatus = "auth"
	WAIT_I     InvoiceStatus = "wait"
	ACCEPTED_I InvoiceStatus = "accepted"
	REJECTED_I InvoiceStatus = "rejected"
)

//reform:acca.invoices
type Invoice struct {
	InvoiceID   int64         `reform:"invoice_id,pk"`
	Key         string        `reform:"key"`
	Status      InvoiceStatus `reform:"status"`
	TotalAmount int64         `reform:"total_amount"`
	Strategy    string        `reform:"strategy"`
	Meta        *[]byte       `reform:"meta"`
	Payload     *[]byte       `reform:"payload"`
	UpdatedAt   time.Time     `reform:"updated_at"`
	CreatedAt   time.Time     `reform:"created_at"`
}

func (i *Invoice) BeforeInsert() error {
	i.UpdatedAt = time.Now()
	i.CreatedAt = time.Now()
	i.Status = DRAFT_I
	return nil
}

func (i *Invoice) BeforeUpdate() error {
	i.UpdatedAt = time.Now()
	return nil
}
