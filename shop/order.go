package shop

import (
	"time"

	"github.com/gebv/acca"
)

//go:generate reform

type Order interface {
	OrderID() string // получаем из внешней системы

	DestinationID() int64

	Type() OrderType
	Total() int64

	Closed() bool
	CreatedAt() time.Time
}

type OrderManager interface {
	Find(orderID string) (Order, error)
	ListInvoices(orderID string) ([]*acca.Invoice, error)
}

func NewOrder(
	orderID string,
	desID int64,
	_type OrderType,
	amount int64,
) *order {
	return &order{
		OrderID:   orderID,
		Type:      _type,
		Total:     amount,
		CreatedAt: time.Now(),
	}
}

//reform:shop.orders
type order struct {
	OrderID string `reform:"order_id" json:"order_id,omitempty"`

	DestinationID int64     `reform:"destination_id" json:"destination_id,omitempty"` //ref to accounts
	Type          OrderType `reform:"order_type" json:"type,omitempty"`
	Total         int64     `reform:"total" json:"total,omitempty"`

	Closed bool `reform:"closed" json:"closed,omitempty"`

	CreatedAt time.Time `reform:"created_at" json:"created_at,omitempty"`
}

type orderContainer struct {
	dat *order
}

func (o orderContainer) OrderID() string {
	return o.dat.OrderID
}

func (o orderContainer) DestinationID() int64 {
	return o.dat.DestinationID
}

func (o orderContainer) Type() OrderType {
	return o.dat.Type
}
func (o orderContainer) Total() int64 {
	return o.dat.Total
}

func (o orderContainer) Closed() bool {
	return o.dat.Closed
}

func (o orderContainer) CreatedAt() time.Time {
	return o.dat.CreatedAt
}
