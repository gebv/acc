package acca

type Cashier interface {
	// Hold first phase of payment - hold amount of invoice.
	Hold(sourceID, invoiceID int64) (txID int64, err error)

	// Accept second phase of payment - payment confimration.
	Accept(txID int64) (err error)

	// Reject second phase of payment - payment not rejected.
	Reject(txID int64) (err error)
}

// type Shop interface {
// 	Checkout(Order) error
// 	Invoice(orderID int64, destinationID int64, amount int64) (invoiceID int64, err error)
// 	FindOrder(orderID int64) (Order, error)
// 	FindInvoice(invoiceID int64) (*Invoice, error)
// }

// type Bank interface {
// 	// GetBalance возвращает баланс счета (доступный для трат).
// 	GetBalance(accountID int64) (amount int64, err error)

// 	// ListAccounts список счетов кустомера
// 	ListAccounts(customerID string) []Account
// 	FindAccount(accountID int64) (Account, error)
// }

// type Transfer interface {
// 	Transfer(
// 		sourceID, destinationID int64,
// 		amount int64,
// 	) (txID int64, err error)
// 	AcceptTx(txID int64) error
// 	RejectTx(txID int64) error
// }
