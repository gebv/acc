package engine

import (
	"github.com/pkg/errors"
)

func newLowLevelMoneyTransferStrategy() *lowLevelMoneyTransferStrategy {
	m := &lowLevelMoneyTransferStrategy{
		accountBalances:         make(listBalances),
		accountAcceptedBalances: make(listBalances),
		allowedSetNextStatus:    true,
		execMap:                 make(map[lowLevelMoneyTransferStrategy__execKey]func(nextTxStatus TransactionStatus, oper *Operation)),
	}
	m.execMap[lowLevelMoneyTransferStrategy__execKey{SIMPLE_OPS, AUTH_TX}] = m.simpleTransfer_auth
	m.execMap[lowLevelMoneyTransferStrategy__execKey{SIMPLE_OPS, ACCEPTED_TX}] = m.simpleTransfer_accepted
	m.execMap[lowLevelMoneyTransferStrategy__execKey{SIMPLE_OPS, REJECTED_TX}] = m.simpleTransfer_rejected

	m.execMap[lowLevelMoneyTransferStrategy__execKey{RECHARGE_OPS, AUTH_TX}] = m.recharge_auth
	m.execMap[lowLevelMoneyTransferStrategy__execKey{RECHARGE_OPS, ACCEPTED_TX}] = m.recharge_accepted
	m.execMap[lowLevelMoneyTransferStrategy__execKey{RECHARGE_OPS, REJECTED_TX}] = m.simpleTransfer_rejected

	m.execMap[lowLevelMoneyTransferStrategy__execKey{WITHDRAW_OPS, AUTH_TX}] = m.withdraw_auth
	m.execMap[lowLevelMoneyTransferStrategy__execKey{WITHDRAW_OPS, ACCEPTED_TX}] = m.withdraw_accepted
	m.execMap[lowLevelMoneyTransferStrategy__execKey{WITHDRAW_OPS, REJECTED_TX}] = m.withdraw_rejected
	return m
}

func newLowLevelMoneyTransferStrategy_withoutChangeOperStatus() *lowLevelMoneyTransferStrategy {
	m := newLowLevelMoneyTransferStrategy()
	m.allowedSetNextStatus = false
	return m
}

type lowLevelMoneyTransferStrategy struct {
	accountBalances         listBalances
	accountAcceptedBalances listBalances
	allowedSetNextStatus    bool
	execMap                 map[lowLevelMoneyTransferStrategy__execKey]func(nextTxStatus TransactionStatus, oper *Operation)
}

type lowLevelMoneyTransferStrategy__execKey struct {
	strategy OperationStrategy
	status   TransactionStatus
}

func (t *lowLevelMoneyTransferStrategy) Process(nextTxStatus TransactionStatus, oper *Operation) error {
	if !allowedOperationStrategies[oper.Strategy] {
		return errors.New("not allowed operation strategy")
	}

	switch nextTxStatus {
	case AUTH_TX:
		if oper.Status != DRAFT_OP {
			return errors.New("not allowed - operation is not in the allowed status")
		}
	case ACCEPTED_TX, REJECTED_TX:
		if oper.Status != HOLD_OP {
			return errors.New("not allowed - operation is not in the allowed status")
		}
	default:
		return errors.New("not allowed - operation is not in the allowed status")
	}

	exec, exists := t.execMap[lowLevelMoneyTransferStrategy__execKey{oper.Strategy, nextTxStatus}]
	if !exists {
		return errors.New("not supported pairs strategy and next tx status")
	}

	exec(nextTxStatus, oper)

	if t.allowedSetNextStatus {
		switch nextTxStatus {
		case AUTH_TX:
			if oper.Hold {
				oper.Status = HOLD_OP
			} else {
				oper.Status = ACCEPTED_OP
			}
		case ACCEPTED_TX:
			oper.Status = ACCEPTED_OP
		case REJECTED_TX:
			oper.Status = REJECTED_OP
		}
	}

	return nil
}

//////////////////////////////////////
// auth
//////////////////////////////////////

func (t *lowLevelMoneyTransferStrategy) simpleTransfer_auth(nextTxStatus TransactionStatus, oper *Operation) {
	if oper.Hold {
		t.accountBalances.Dec(oper.SrcAccID, oper.Amount)
		if oper.HoldAccID != nil {
			t.accountBalances.Inc(*oper.HoldAccID, oper.Amount)
		}
	} else {
		t.accountBalances.Dec(oper.SrcAccID, oper.Amount)
		t.accountAcceptedBalances.Dec(oper.SrcAccID, oper.Amount)

		t.accountBalances.Inc(oper.DstAccID, oper.Amount)
		t.accountAcceptedBalances.Inc(oper.DstAccID, oper.Amount)
	}
}

func (t *lowLevelMoneyTransferStrategy) recharge_auth(nextTxStatus TransactionStatus, oper *Operation) {
	if oper.Hold {
		if oper.HoldAccID != nil {
			t.accountBalances.Inc(*oper.HoldAccID, oper.Amount)
		}
	} else {
		t.accountBalances.Inc(oper.SrcAccID, oper.Amount)
		t.accountAcceptedBalances.Inc(oper.SrcAccID, oper.Amount)

		t.accountBalances.Inc(oper.DstAccID, oper.Amount)
		t.accountAcceptedBalances.Inc(oper.DstAccID, oper.Amount)
	}
}

func (t *lowLevelMoneyTransferStrategy) withdraw_auth(nextTxStatus TransactionStatus, oper *Operation) {
	if oper.Hold {
		t.accountBalances.Dec(oper.SrcAccID, oper.Amount)
		if oper.HoldAccID != nil {
			t.accountBalances.Inc(*oper.HoldAccID, oper.Amount)
		}
	} else {
		t.accountBalances.Dec(oper.SrcAccID, oper.Amount)
		t.accountAcceptedBalances.Dec(oper.SrcAccID, oper.Amount)

		t.accountBalances.Dec(oper.DstAccID, oper.Amount)
		t.accountAcceptedBalances.Dec(oper.DstAccID, oper.Amount)
	}
}

//////////////////////////////////////
// accepted
//////////////////////////////////////

func (t *lowLevelMoneyTransferStrategy) simpleTransfer_accepted(nextTxStatus TransactionStatus, oper *Operation) {
	if oper.Hold {
		t.accountAcceptedBalances.Dec(oper.SrcAccID, oper.Amount)

		t.accountBalances.Inc(oper.DstAccID, oper.Amount)
		t.accountAcceptedBalances.Inc(oper.DstAccID, oper.Amount)
		if oper.HoldAccID != nil {
			t.accountBalances.Dec(*oper.HoldAccID, oper.Amount)
		}
	}
}

func (t *lowLevelMoneyTransferStrategy) recharge_accepted(nextTxStatus TransactionStatus, oper *Operation) {
	if oper.Hold {
		t.accountBalances.Inc(oper.SrcAccID, oper.Amount)
		t.accountAcceptedBalances.Inc(oper.SrcAccID, oper.Amount)

		t.accountBalances.Inc(oper.DstAccID, oper.Amount)
		t.accountAcceptedBalances.Inc(oper.DstAccID, oper.Amount)

		if oper.HoldAccID != nil {
			t.accountBalances.Dec(*oper.HoldAccID, oper.Amount)
		}
	}
}

func (t *lowLevelMoneyTransferStrategy) withdraw_accepted(nextTxStatus TransactionStatus, oper *Operation) {
	if oper.Hold {
		t.accountAcceptedBalances.Dec(oper.SrcAccID, oper.Amount)

		t.accountBalances.Dec(oper.DstAccID, oper.Amount)
		t.accountAcceptedBalances.Dec(oper.DstAccID, oper.Amount)
		if oper.HoldAccID != nil {
			t.accountBalances.Dec(*oper.HoldAccID, oper.Amount)
		}
	}
}

//////////////////////////////////////
// rejected
//////////////////////////////////////

func (t *lowLevelMoneyTransferStrategy) simpleTransfer_rejected(nextTxStatus TransactionStatus, oper *Operation) {
	if oper.Hold {
		t.accountBalances.Inc(oper.SrcAccID, oper.Amount)
		if oper.HoldAccID != nil {
			t.accountBalances.Dec(*oper.HoldAccID, oper.Amount)
		}
	} else {
		t.accountBalances.Inc(oper.SrcAccID, oper.Amount)
		t.accountAcceptedBalances.Inc(oper.SrcAccID, oper.Amount)

		t.accountBalances.Dec(oper.DstAccID, oper.Amount)
		t.accountAcceptedBalances.Dec(oper.DstAccID, oper.Amount)
	}
}

func (t *lowLevelMoneyTransferStrategy) recharge_rejected(nextTxStatus TransactionStatus, oper *Operation) {
	if oper.Hold {
		if oper.HoldAccID != nil {
			t.accountBalances.Dec(*oper.HoldAccID, oper.Amount)
		}
	} else {
		t.accountBalances.Dec(oper.SrcAccID, oper.Amount)
		t.accountAcceptedBalances.Dec(oper.SrcAccID, oper.Amount)

		t.accountBalances.Dec(oper.DstAccID, oper.Amount)
		t.accountAcceptedBalances.Dec(oper.DstAccID, oper.Amount)
	}
}

func (t *lowLevelMoneyTransferStrategy) withdraw_rejected(nextTxStatus TransactionStatus, oper *Operation) {
	if oper.Hold {
		t.accountBalances.Inc(oper.SrcAccID, oper.Amount)
		if oper.HoldAccID != nil {
			t.accountBalances.Dec(*oper.HoldAccID, oper.Amount)
		}
	} else {
		t.accountBalances.Inc(oper.SrcAccID, oper.Amount)
		t.accountAcceptedBalances.Inc(oper.SrcAccID, oper.Amount)

		t.accountBalances.Inc(oper.DstAccID, oper.Amount)
		t.accountAcceptedBalances.Inc(oper.DstAccID, oper.Amount)
	}
}
