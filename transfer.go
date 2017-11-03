package acca

type Transfer interface {
	// Hold first phase of payment - hold amount of invoice (optional to specifed source account).
	Hold(invoiceID int64, sourceID ...int64) (txID int64, err error)

	// Accept second phase of payment - payment confimration.
	Accept(txID int64) (err error)

	// Reject second phase of payment - payment not rejected.
	Reject(txID int64) (err error)
}

type TransferInspector interface {
	CanTransfer(sourceID, destinationID int64, amount int64) error
}
