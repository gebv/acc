package acc

import "time"

//go:generate reform

//reform:invoices
type Invoice struct {
	InvoiceID     int64  `reform:"invoice_id,pk"`
	OrderID       string `reform:"order_id" `      // ref to ext layer
	DestinationID int64  `reform:"destination_id"` // ref to account ID
	SourceID      int64  `reform:"source_id"`

	Paid bool `reform:"paid"`

	Amount    int64     `reform:"amount"`
	CreatedAt time.Time `reform:"created_at"`
}
