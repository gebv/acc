package shop

import (
	"github.com/gebv/acca"
)

var _ acca.Transfer = (*SimpleTransfer)(nil)

type SimpleTransfer struct {
	tr acca.Transfer
}

func (t *SimpleTransfer) Hold(sourceID, invoiceID int64) (txID int64, err error) {
	txID, err = t.tr.Hold(sourceID, invoiceID)
	if err != nil {
		return
	}
	return txID, t.Accept(txID)
}

// Accept second phase of payment - payment confimration.
func (t *SimpleTransfer) Accept(txID int64) (err error) {
	return acca.ErrNotSupported
}

// Reject second phase of payment - payment not rejected.
func (t *SimpleTransfer) Reject(txID int64) (err error) {
	return acca.ErrNotSupported
}
