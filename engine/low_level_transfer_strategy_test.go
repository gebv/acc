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
		wantBalances         []int64
		wantAcceptedBalances []int64
		hold                 bool
		wantErr              bool
	}{
		// SIMPLE - without hold
		{SIMPLE_OPS, DRAFT_OP, DRAFT_OP, DRAFT_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, DRAFT_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, DRAFT_TX, []int64{0, 0}, []int64{0, 0}, false, true},

		{SIMPLE_OPS, DRAFT_OP, ACCEPTED_OP, AUTH_TX, []int64{-10, 10}, []int64{-10, 10}, false, false}, // simple
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, AUTH_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, AUTH_TX, []int64{0, 0}, []int64{0, 0}, false, true},

		{SIMPLE_OPS, DRAFT_OP, DRAFT_OP, ACCEPTED_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, ACCEPTED_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, ACCEPTED_TX, []int64{0, 0}, []int64{0, 0}, false, true},

		{SIMPLE_OPS, DRAFT_OP, DRAFT_OP, REJECTED_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, REJECTED_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, REJECTED_TX, []int64{0, 0}, []int64{0, 0}, false, true},

		{SIMPLE_OPS, DRAFT_OP, DRAFT_OP, FAILED_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, FAILED_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, FAILED_TX, []int64{0, 0}, []int64{0, 0}, false, true},

		// SIMPLE - with hold
		{SIMPLE_OPS, DRAFT_OP, DRAFT_OP, DRAFT_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{SIMPLE_OPS, HOLD_OP, HOLD_OP, DRAFT_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, DRAFT_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, DRAFT_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},

		{SIMPLE_OPS, DRAFT_OP, HOLD_OP, AUTH_TX, []int64{-10, 0, 10}, []int64{0, 0, 0}, true, false}, // hold
		{SIMPLE_OPS, HOLD_OP, HOLD_OP, AUTH_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, AUTH_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, AUTH_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},

		{SIMPLE_OPS, DRAFT_OP, DRAFT_OP, ACCEPTED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{SIMPLE_OPS, HOLD_OP, ACCEPTED_OP, ACCEPTED_TX, []int64{0, 10, -10}, []int64{-10, 10, 0}, true, false}, // accepted hold
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, ACCEPTED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, ACCEPTED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},

		{SIMPLE_OPS, DRAFT_OP, DRAFT_OP, REJECTED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{SIMPLE_OPS, HOLD_OP, REJECTED_OP, REJECTED_TX, []int64{10, 0, -10}, []int64{0, 0, 0}, true, false}, // rejected hold
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, REJECTED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, REJECTED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},

		{SIMPLE_OPS, DRAFT_OP, DRAFT_OP, FAILED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{SIMPLE_OPS, HOLD_OP, HOLD_OP, FAILED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{SIMPLE_OPS, ACCEPTED_OP, ACCEPTED_OP, FAILED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{SIMPLE_OPS, REJECTED_OP, REJECTED_OP, FAILED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},

		// RECHARGE - without hold
		{RECHARGE_OPS, DRAFT_OP, DRAFT_OP, DRAFT_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{RECHARGE_OPS, ACCEPTED_OP, ACCEPTED_OP, DRAFT_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{RECHARGE_OPS, REJECTED_OP, REJECTED_OP, DRAFT_TX, []int64{0, 0}, []int64{0, 0}, false, true},

		{RECHARGE_OPS, DRAFT_OP, ACCEPTED_OP, AUTH_TX, []int64{10, 10}, []int64{10, 10}, false, false}, // recharge
		{RECHARGE_OPS, ACCEPTED_OP, ACCEPTED_OP, AUTH_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{RECHARGE_OPS, REJECTED_OP, REJECTED_OP, AUTH_TX, []int64{0, 0}, []int64{0, 0}, false, true},

		{RECHARGE_OPS, DRAFT_OP, DRAFT_OP, ACCEPTED_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{RECHARGE_OPS, ACCEPTED_OP, ACCEPTED_OP, ACCEPTED_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{RECHARGE_OPS, REJECTED_OP, REJECTED_OP, ACCEPTED_TX, []int64{0, 0}, []int64{0, 0}, false, true},

		{RECHARGE_OPS, DRAFT_OP, DRAFT_OP, REJECTED_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{RECHARGE_OPS, ACCEPTED_OP, ACCEPTED_OP, REJECTED_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{RECHARGE_OPS, REJECTED_OP, REJECTED_OP, REJECTED_TX, []int64{0, 0}, []int64{0, 0}, false, true},

		{RECHARGE_OPS, DRAFT_OP, DRAFT_OP, FAILED_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{RECHARGE_OPS, ACCEPTED_OP, ACCEPTED_OP, FAILED_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{RECHARGE_OPS, REJECTED_OP, REJECTED_OP, FAILED_TX, []int64{0, 0}, []int64{0, 0}, false, true},

		// RECHARGE - with hold
		{RECHARGE_OPS, DRAFT_OP, DRAFT_OP, DRAFT_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{RECHARGE_OPS, HOLD_OP, HOLD_OP, DRAFT_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{RECHARGE_OPS, ACCEPTED_OP, ACCEPTED_OP, DRAFT_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{RECHARGE_OPS, REJECTED_OP, REJECTED_OP, DRAFT_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},

		{RECHARGE_OPS, DRAFT_OP, HOLD_OP, AUTH_TX, []int64{0, 0, 10}, []int64{0, 0, 0}, true, false}, // hold
		{RECHARGE_OPS, HOLD_OP, HOLD_OP, AUTH_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{RECHARGE_OPS, ACCEPTED_OP, ACCEPTED_OP, AUTH_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{RECHARGE_OPS, REJECTED_OP, REJECTED_OP, AUTH_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},

		{RECHARGE_OPS, DRAFT_OP, DRAFT_OP, ACCEPTED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{RECHARGE_OPS, HOLD_OP, ACCEPTED_OP, ACCEPTED_TX, []int64{10, 10, -10}, []int64{10, 10, 0}, true, false}, // accepted hold
		{RECHARGE_OPS, ACCEPTED_OP, ACCEPTED_OP, ACCEPTED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{RECHARGE_OPS, REJECTED_OP, REJECTED_OP, ACCEPTED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},

		{RECHARGE_OPS, DRAFT_OP, DRAFT_OP, REJECTED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{RECHARGE_OPS, HOLD_OP, REJECTED_OP, REJECTED_TX, []int64{0, 0, -10}, []int64{0, 0, 0}, true, false}, // rejected hold
		{RECHARGE_OPS, ACCEPTED_OP, ACCEPTED_OP, REJECTED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{RECHARGE_OPS, REJECTED_OP, REJECTED_OP, REJECTED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},

		{RECHARGE_OPS, DRAFT_OP, DRAFT_OP, FAILED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{RECHARGE_OPS, HOLD_OP, HOLD_OP, FAILED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{RECHARGE_OPS, ACCEPTED_OP, ACCEPTED_OP, FAILED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{RECHARGE_OPS, REJECTED_OP, REJECTED_OP, FAILED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},

		// WITHDRAW - without hold
		{WITHDRAW_OPS, DRAFT_OP, DRAFT_OP, DRAFT_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{WITHDRAW_OPS, ACCEPTED_OP, ACCEPTED_OP, DRAFT_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{WITHDRAW_OPS, REJECTED_OP, REJECTED_OP, DRAFT_TX, []int64{0, 0}, []int64{0, 0}, false, true},

		{WITHDRAW_OPS, DRAFT_OP, ACCEPTED_OP, AUTH_TX, []int64{-10, -10}, []int64{-10, -10}, false, false}, // withdraw
		{WITHDRAW_OPS, ACCEPTED_OP, ACCEPTED_OP, AUTH_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{WITHDRAW_OPS, REJECTED_OP, REJECTED_OP, AUTH_TX, []int64{0, 0}, []int64{0, 0}, false, true},

		{WITHDRAW_OPS, DRAFT_OP, DRAFT_OP, ACCEPTED_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{WITHDRAW_OPS, ACCEPTED_OP, ACCEPTED_OP, ACCEPTED_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{WITHDRAW_OPS, REJECTED_OP, REJECTED_OP, ACCEPTED_TX, []int64{0, 0}, []int64{0, 0}, false, true},

		{WITHDRAW_OPS, DRAFT_OP, DRAFT_OP, REJECTED_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{WITHDRAW_OPS, ACCEPTED_OP, ACCEPTED_OP, REJECTED_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{WITHDRAW_OPS, REJECTED_OP, REJECTED_OP, REJECTED_TX, []int64{0, 0}, []int64{0, 0}, false, true},

		{WITHDRAW_OPS, DRAFT_OP, DRAFT_OP, FAILED_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{WITHDRAW_OPS, ACCEPTED_OP, ACCEPTED_OP, FAILED_TX, []int64{0, 0}, []int64{0, 0}, false, true},
		{WITHDRAW_OPS, REJECTED_OP, REJECTED_OP, FAILED_TX, []int64{0, 0}, []int64{0, 0}, false, true},

		// WITHDRAW - with hold
		{WITHDRAW_OPS, DRAFT_OP, DRAFT_OP, DRAFT_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{WITHDRAW_OPS, HOLD_OP, HOLD_OP, DRAFT_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{WITHDRAW_OPS, ACCEPTED_OP, ACCEPTED_OP, DRAFT_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{WITHDRAW_OPS, REJECTED_OP, REJECTED_OP, DRAFT_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},

		{WITHDRAW_OPS, DRAFT_OP, HOLD_OP, AUTH_TX, []int64{-10, -10, 10}, []int64{0, 0, 0}, true, false}, // hold
		{WITHDRAW_OPS, HOLD_OP, HOLD_OP, AUTH_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{WITHDRAW_OPS, ACCEPTED_OP, ACCEPTED_OP, AUTH_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{WITHDRAW_OPS, REJECTED_OP, REJECTED_OP, AUTH_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},

		{WITHDRAW_OPS, DRAFT_OP, DRAFT_OP, ACCEPTED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{WITHDRAW_OPS, HOLD_OP, ACCEPTED_OP, ACCEPTED_TX, []int64{0, 0, -10}, []int64{-10, -10, 0}, true, false}, // accepted hold
		{WITHDRAW_OPS, ACCEPTED_OP, ACCEPTED_OP, ACCEPTED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{WITHDRAW_OPS, REJECTED_OP, REJECTED_OP, ACCEPTED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},

		{WITHDRAW_OPS, DRAFT_OP, DRAFT_OP, REJECTED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{WITHDRAW_OPS, HOLD_OP, REJECTED_OP, REJECTED_TX, []int64{10, 10, -10}, []int64{0, 0, 0}, true, false}, // rejected hold
		{WITHDRAW_OPS, ACCEPTED_OP, ACCEPTED_OP, REJECTED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{WITHDRAW_OPS, REJECTED_OP, REJECTED_OP, REJECTED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},

		{WITHDRAW_OPS, DRAFT_OP, DRAFT_OP, FAILED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{WITHDRAW_OPS, HOLD_OP, HOLD_OP, FAILED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{WITHDRAW_OPS, ACCEPTED_OP, ACCEPTED_OP, FAILED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
		{WITHDRAW_OPS, REJECTED_OP, REJECTED_OP, FAILED_TX, []int64{0, 0, 0}, []int64{0, 0, 0}, true, true},
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
				gotBalance := ll.accountBalances.get(accountID)
				if gotBalance != balance {
					t.Errorf("accountBalances[id=%d] = %d, want %d", accountID, gotBalance, balance)
				}
			}

			for i, balance := range tt.wantAcceptedBalances {
				accountID := int64(i + 1)
				gotBalance := ll.accountAcceptedBalances.get(accountID)
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
