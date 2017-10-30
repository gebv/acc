package acca

type OrderType string

type Order interface {
	OrderID() string // получаем из внешней системы

	DestinationID() int64

	Type() OrderType
	Total() int64

	Closed() bool
}

type OrderManager interface {
	Find(orderID string) (Order, error)
	ListInvoices(orderID string) ([]*Invoice, error)
}
