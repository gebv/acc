package acca

import "time"

//go:generate reform

type TransactionStatus string

var (
	Authorization TransactionStatus = "authorization" // транзакция авторизирована
	Accepted      TransactionStatus = "accepted"      // транзакция подтверждена
	Rejected      TransactionStatus = "rejected"      // транзакция отменена
)

//reform:finances.transactions
type Transaction struct {
	TransactionID int64 `reform:"transaction_id,pk"`
	InvoiceID     int64 `reform:"invoice_id"`

	Amount      int64 `reform:"amount"`
	Source      int64 `reform:"source"`      // ref to account ID
	Destination int64 `reform:"destination"` // ref to account ID

	Status TransactionStatus `reform:"status"`

	CreatedAt time.Time `reform:"created_at"`
	ClosedAt  time.Time `reform:"closed_at"`
}

type BalanceChangesType string

var (
	Hold     BalanceChangesType = "hold"     // средства заморозились
	Refund   BalanceChangesType = "refund"   // средства возвращаются
	Complete BalanceChangesType = "complete" // средства доставляются
)

//reform:finances.balance_changes
type BalanceChanges struct {
	ChangeID      int64 `reform:"change_id,pk"`
	AccountID     int64 `reform:"account_id"`
	TransactionID int64 `reform:"transaction_id"`

	Type BalanceChangesType `reform:"_type"`

	Amount  int64 `reform:"amount"`
	Balance int64 `reform:"balance"`

	CreatedAt time.Time `reform:"created_at"`
}
