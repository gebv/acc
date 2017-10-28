package acca

import "time"

//go:generate reform

type AccountType string

var (
	UnknownAccount AccountType = ""
	System         AccountType = "system"
	Partner        AccountType = "partner"
	Customer       AccountType = "customer"
)

//reform:accounts
type Account struct {
	AccountID  int64       `reform:"account_id,pk"`
	CustomerID string      `reform:"customer_id"`
	Type       AccountType `reform:"_type"`
	Balance    int64       `reform:"balance"`
	UpdatedAt  time.Time   `reform:"updated_at"`
}
