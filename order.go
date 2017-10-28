package acca

type OrderType string

type Order interface {
	OrderID() string // получаем из внешней системы

	DestinationID() int64

	Type() OrderType
	Total() int64
}

type Stock interface {
	FindOrderByID(orderID string) (Order, error)
}
