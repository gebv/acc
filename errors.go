package acca

import "errors"

var (
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrInvoiceHasBeenPaid = errors.New("invoice has been paid")
	ErrOrderClosed        = errors.New("order closed")

	ErrNotFound     = errors.New("not found")
	ErrNotSupported = errors.New("not supported")
	ErrNotAllowed   = errors.New("not allowed")
)
