package engine

var operationStatusTransitionChart = OperationStatusTransitionChart{
	DRAFT_OP: {HOLD_OP, ACCEPTED_OP, REJECTED_OP},
	HOLD_OP:  {ACCEPTED_OP, REJECTED_OP},
}

var transactionStatusTransitionChart = TransactionStatusTransitionChart{
	DRAFT_TX: {AUTH_TX},
	AUTH_TX:  {ACCEPTED_TX, REJECTED_TX, FAILED_TX},
}

var transitionsStatusesOfInvoice = InvoicesStatusTransitionChart{
	AUTH_I: {WAIT_I, ACCEPTED_I, REJECTED_I},
	WAIT_I: {ACCEPTED_I, REJECTED_I},
}

type OperationStatusTransitionChart map[OperationStatus][]OperationStatus

func (s OperationStatusTransitionChart) Allowed(from, to OperationStatus) bool {
	list, exists := s[from]
	if !exists {
		return false
	}
	for _, status := range list {
		if status.Match(to) {
			return true
		}
	}
	return false
}

type TransactionStatusTransitionChart map[TransactionStatus][]TransactionStatus

func (s TransactionStatusTransitionChart) Allowed(from, to TransactionStatus) bool {
	list, exists := s[from]
	if !exists {
		return false
	}
	for _, status := range list {
		if status.Match(to) {
			return true
		}
	}
	return false
}

type InvoicesStatusTransitionChart map[InvoiceStatus][]InvoiceStatus

func (s InvoicesStatusTransitionChart) Allowed(from, to InvoiceStatus) bool {
	list, exists := s[from]
	if !exists {
		return false
	}
	for _, status := range list {
		if status.Match(to) {
			return true
		}
	}
	return false
}
