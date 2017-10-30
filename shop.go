package acca

type Seller interface {
	Invoice(orderID string, amount int64) (*Invoice, error)
}

type Payment interface {
	Pay(invoiceID, sourceID int64) error
}

type PaymentInspector interface {
	CanPay(invoiceID, sourceID int64) error
}
