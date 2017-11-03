package shop

import (
	"github.com/gebv/acca"
)

type Manager interface {
	CreateOrder(orderID string, destinationID int64, _type OrderType, amount int64) (Order, error)
	Invoice(orderID string, amount int64) (*acca.Invoice, error)
}

type Cashier interface {
	Pay(invoiceID, sourceID int64) error
}

type PaymentInspector interface {
	CanPay(invoiceID, sourceID int64) error
}

type Shop interface {
	Manager
	Cashier
}
