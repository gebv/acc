package acca

import (
	"database/sql"
	"time"
)

//go:generate reform

//reform:finances.invoices
type Invoice struct {
	InvoiceID     int64         `reform:"invoice_id,pk"`
	OrderID       string        `reform:"order_id" `      // ref to ext layer
	DestinationID int64         `reform:"destination_id"` // ref to account ID
	SourceID      sql.NullInt64 `reform:"source_id"`

	Paid bool `reform:"paid"`

	Amount    int64     `reform:"amount"`
	CreatedAt time.Time `reform:"created_at"`
}

func (i Invoice) SourceIDOrZero() int64 {
	return i.SourceID.Int64
}

func (i *Invoice) SetSourceID(v int64) {
	i.SourceID.Int64 = v
	i.SourceID.Valid = v > 0
}
