package engine

import (
	"fmt"
	"testing"
)

func Test_lowLevelMoneyTransferStrategy_Process(t *testing.T) {
	var srcAccID, dstAccID, holdAccID, amount int64 = 1, 2, 3, 10

	tests := []struct {
		opStrategy           OperationStrategy
		opInitStatus         OperationStatus
		opNextStatus         OperationStatus
		txNextStatus         TransactionStatus
		hold                 bool
		wantBalances         []int64
		wantAcceptedBalances []int64
		wantErr              bool
	}{
		// SIMPLE - without hold
		{SIMPLE_OPS, DRAFT_OP, DRAFT_OP, DRAFT_TX, false, []int64{0, 0}, []int64{0, 0}, true},
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, DRAFT_TX, false, []int64{0, 0}, []int64{0, 0}, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, DRAFT_TX, false, []int64{0, 0}, []int64{0, 0}, true},

		{SIMPLE_OPS, DRAFT_OP, ACCEPTED_OP, AUTH_TX, false, []int64{-10, 10}, []int64{-10, 10}, false}, // simple
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, AUTH_TX, false, []int64{0, 0}, []int64{0, 0}, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, AUTH_TX, false, []int64{0, 0}, []int64{0, 0}, true},

		{SIMPLE_OPS, DRAFT_OP, DRAFT_OP, ACCEPTED_TX, false, []int64{0, 0}, []int64{0, 0}, true},
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, ACCEPTED_TX, false, []int64{0, 0}, []int64{0, 0}, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, ACCEPTED_TX, false, []int64{0, 0}, []int64{0, 0}, true},

		{SIMPLE_OPS, DRAFT_OP, DRAFT_OP, REJECTED_TX, false, []int64{0, 0}, []int64{0, 0}, true},
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, REJECTED_TX, false, []int64{0, 0}, []int64{0, 0}, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, REJECTED_TX, false, []int64{0, 0}, []int64{0, 0}, true},

		{SIMPLE_OPS, DRAFT_OP, DRAFT_OP, FAILED_TX, false, []int64{0, 0}, []int64{0, 0}, true},
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, FAILED_TX, false, []int64{0, 0}, []int64{0, 0}, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, FAILED_TX, false, []int64{0, 0}, []int64{0, 0}, true},

		// SIMPLE - with hold
		{SIMPLE_OPS, DRAFT_OP, DRAFT_OP, DRAFT_TX, true, []int64{0, 0, 0}, []int64{0, 0, 0}, true},
		{SIMPLE_OPS, HOLD_OP, HOLD_OP, DRAFT_TX, true, []int64{0, 0, 0}, []int64{0, 0, 0}, true},
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, DRAFT_TX, true, []int64{0, 0, 0}, []int64{0, 0, 0}, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, DRAFT_TX, true, []int64{0, 0, 0}, []int64{0, 0, 0}, true},

		{SIMPLE_OPS, DRAFT_OP, HOLD_OP, AUTH_TX, true, []int64{-10, 0, 10}, []int64{0, 0, 0}, false}, // simple
		{SIMPLE_OPS, HOLD_OP, HOLD_OP, AUTH_TX, true, []int64{0, 0, 0}, []int64{0, 0, 0}, true},
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, AUTH_TX, true, []int64{0, 0, 0}, []int64{0, 0, 0}, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, AUTH_TX, true, []int64{0, 0, 0}, []int64{0, 0, 0}, true},

		{SIMPLE_OPS, DRAFT_OP, DRAFT_OP, ACCEPTED_TX, true, []int64{0, 0, 0}, []int64{0, 0, 0}, true},
		{SIMPLE_OPS, HOLD_OP, ACCEPTED_OP, ACCEPTED_TX, true, []int64{0, 10, -10}, []int64{-10, 10, 0}, false}, // accepted hold
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, ACCEPTED_TX, true, []int64{0, 0, 0}, []int64{0, 0, 0}, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, ACCEPTED_TX, true, []int64{0, 0, 0}, []int64{0, 0, 0}, true},

		{SIMPLE_OPS, DRAFT_OP, DRAFT_OP, REJECTED_TX, true, []int64{0, 0, 0}, []int64{0, 0, 0}, true},
		{SIMPLE_OPS, HOLD_OP, REJECTED_OP, REJECTED_TX, true, []int64{10, 0, -10}, []int64{0, 0, 0}, false}, // rejected hold
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, REJECTED_TX, true, []int64{0, 0, 0}, []int64{0, 0, 0}, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, REJECTED_TX, true, []int64{0, 0, 0}, []int64{0, 0, 0}, true},

		{SIMPLE_OPS, DRAFT_OP, DRAFT_OP, FAILED_TX, true, []int64{0, 0, 0}, []int64{0, 0, 0}, true},
		{SIMPLE_OPS, HOLD_OP, HOLD_OP, FAILED_TX, true, []int64{0, 0, 0}, []int64{0, 0, 0}, true},
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, FAILED_TX, true, []int64{0, 0, 0}, []int64{0, 0, 0}, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, FAILED_TX, true, []int64{0, 0, 0}, []int64{0, 0, 0}, true},
	}
	for _, tt := range tests {
		tname := fmt.Sprintf("%s: op:%s->op:%s (tx:%s) err=%v balances=%v",
			tt.opStrategy,
			tt.opInitStatus,
			tt.opNextStatus,
			tt.txNextStatus,
			tt.wantErr,
			tt.wantBalances,
		)

		t.Run(tname, func(t *testing.T) {
			// asserts

			if tt.hold && len(tt.wantBalances) != 3 {
				t.Errorf("not exptected number list balances")
			}
			if tt.hold && len(tt.wantAcceptedBalances) != 3 {
				t.Errorf("not exptected number list balances")
			}

			ll := newLowLevelMoneyTransferStrategy()
			oper := &Operation{
				SrcAccID: srcAccID,
				DstAccID: dstAccID,
				Amount:   amount,
				Strategy: tt.opStrategy,
				Status:   tt.opInitStatus,
				Hold:     tt.hold,
			}
			if oper.Hold {
				oper.HoldAccID = &holdAccID
			}
			if err := ll.Process(tt.txNextStatus, oper); (err != nil) != tt.wantErr {
				t.Errorf("lowLevelMoneyTransferStrategy.Process() error = %v, wantErr %v", err, tt.wantErr)
			}

			for i, balance := range tt.wantBalances {
				accountID := int64(i + 1)
				gotBalance := ll.accountBalances.Get(accountID)
				if gotBalance != balance {
					t.Errorf("accountBalances[id=%d] = %d, want %d", accountID, gotBalance, balance)
				}
			}

			for i, balance := range tt.wantAcceptedBalances {
				accountID := int64(i + 1)
				gotBalance := ll.accountAcceptedBalances.Get(accountID)
				if gotBalance != balance {
					t.Errorf("accountAcceptedBalances[id=%d] = %d, want %d", accountID, gotBalance, balance)
				}
			}

			if oper.Status != tt.opNextStatus {
				t.Errorf("not exptected next status of operation got=%v, want=%v", oper.Status, tt.opNextStatus)
			}
		})
	}
}
