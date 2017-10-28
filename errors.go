package acca

import "errors"

var (
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrInvoiceHasBeenPaid = errors.New("invoice has been paid")
	ErrNotFound           = errors.New("not found")
)
