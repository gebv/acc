package shop

import (
	"errors"
	"time"

	"github.com/gebv/acca"
)

func NewInvoice(o Order, amount int64) (*acca.Invoice, error) {
	if amount > o.Total() {
		return nil, errors.New("invalid amount")
	}

	return &acca.Invoice{
		OrderID:       o.OrderID(),
		DestinationID: o.DestinationID(),
		Amount:        amount,
		CreatedAt:     time.Now(),
	}, nil
}
