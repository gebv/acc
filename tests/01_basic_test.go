package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type accountInfo1 struct {
	AccID   string
	Balance uint64
}

type testCase1 struct {
	CaseName string

	InitAccounts []accountInfo1

	Transfers transfers
	MetaData  MetaData
	Reason    string

	TxID             int64
	ExpectedBalances map[string]uint64

	TxStatusBefore string
	TxStatusAfter  string

	NumProcess int
}

// Follow workflow
// - Create accounts with initial balances
// - Add transfers
// - Check statuses before execute requests
// - Process from queue
// - Check statuses after execute requests
// - Check balances
func Test01Basic_01SimpleTransferWithoutHold(t *testing.T) {
	curr := "curr1"
	t.Run("SetupCurrencies", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO acca.currencies(curr) VALUES ($1)`, curr)
		assert.NoErrorf(t, err, "Failed insert currency: %v", curr)
	})

	tests := []testCase1{
		{
			"InternalTransfer",
			[]accountInfo1{
				{
					"acc1.1",
					10,
				},
				{
					"acc1.2",
					20,
				},
				{
					"acc1.3",
					30,
				},
			},
			transfers{
				{
					SrcAccID: "acc1.1",
					DstAccID: "acc1.2",
					Type:     Internal,
					Amount:   9,
					Reason:   "fortesting",
					Meta: MetaData{
						"foo": "bar",
					},
					Hold: false,
				},
				{
					SrcAccID: "acc1.2",
					DstAccID: "acc1.3",
					Type:     Internal,
					Amount:   29,
					Reason:   "fortesting",
					Meta: MetaData{
						"foo": "bar",
					},
					Hold: false,
				},
			},
			MetaData{"foo": "bar"},
			"testing",
			0,
			map[string]uint64{
				"acc1.1": 10 - 9,
				"acc1.2": 20 + 9 - 29,
				"acc1.3": 30 + 29,
			},
			"draft",
			"accepted",
			1,
		},
		{
			"InternalTransfer_Error_NotEnoughMoney",
			[]accountInfo1{
				{
					"acc2.1",
					10,
				},
				{
					"acc2.2",
					20,
				},
				{
					"acc2.3",
					30,
				},
			},
			transfers{
				{
					SrcAccID: "acc2.1",
					DstAccID: "acc2.2",
					Type:     Internal,
					Amount:   1000,
					Reason:   "fortesting",
					Meta: MetaData{
						"foo": "bar",
					},
					Hold: false,
				},
				{
					SrcAccID: "acc2.2",
					DstAccID: "acc2.3",
					Type:     Internal,
					Amount:   1,
					Reason:   "fortesting",
					Meta: MetaData{
						"foo": "bar",
					},
					Hold: false,
				},
			},
			MetaData{"foo": "bar"},
			"testing",
			0,
			map[string]uint64{
				"acc2.1": 10,
				"acc2.2": 20,
				"acc2.3": 30,
			},
			"draft",
			"failed",
			1,
		},
		{
			"InternalTransfer_EmptyListTransfers",
			[]accountInfo1{
				{
					"acc4.1",
					10,
				},
				{
					"acc4.2",
					20,
				},
				{
					"acc4.3",
					30,
				},
			},
			transfers{},
			MetaData{"foo": "bar"},
			"testing",
			0,
			map[string]uint64{
				"acc4.1": 10,
				"acc4.2": 20,
				"acc4.3": 30,
			},
			"draft",
			"accepted",
			1,
		},
		{
			"InternalTransfer_Recharge",
			[]accountInfo1{
				{
					"acc5.payment_gateway.paypal",
					0,
				},
				{
					"acc5.client",
					0,
				},
			},
			transfers{
				{
					SrcAccID: "acc5.payment_gateway.paypal",
					DstAccID: "acc5.client",
					Type:     Recharge,
					Amount:   10,
					Reason:   "fortesting",
					Meta: MetaData{
						"foo": "bar",
					},
					Hold: false,
				},
			},
			MetaData{"foo": "bar"},
			"testing",
			0,
			map[string]uint64{
				"acc5.payment_gateway.paypal": 10,
				"acc5.client":                 10,
			},
			"draft",
			"accepted",
			1,
		},
		{
			"InternalTransfer_Withdraw",
			[]accountInfo1{
				{
					"acc6.payment_gateway.paypal",
					100,
				},
				{
					"acc6.client",
					10,
				},
			},
			transfers{
				{
					SrcAccID: "acc6.client",
					DstAccID: "acc6.payment_gateway.paypal",
					Type:     Withdraw,
					Amount:   10,
					Reason:   "fortesting",
					Meta: MetaData{
						"foo": "bar",
					},
					Hold: false,
				},
			},
			MetaData{"foo": "bar"},
			"testing",
			0,
			map[string]uint64{
				"acc6.payment_gateway.paypal": 90,
				"acc6.client":                 0,
			},
			"draft",
			"accepted",
			1,
		},
		{
			"InternalTransfer_WithdrawError1_NotEnoughMoneyFromSysAccount",
			[]accountInfo1{
				{
					"acc7.payment_gateway.paypal",
					0,
				},
				{
					"acc7.client",
					10,
				},
			},
			transfers{
				{
					SrcAccID: "acc7.client",
					DstAccID: "acc7.payment_gateway.paypal",
					Type:     Withdraw,
					Amount:   10,
					Reason:   "fortesting",
					Meta: MetaData{
						"foo": "bar",
					},
					Hold: false,
				},
			},
			MetaData{"foo": "bar"},
			"testing",
			0,
			map[string]uint64{
				"acc7.payment_gateway.paypal": 0,
				"acc7.client":                 10,
			},
			"draft",
			"failed",
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.CaseName, func(t *testing.T) {
			t.Run("SetupAccounts", func(t *testing.T) {
				for _, acc := range tt.InitAccounts {
					_, err := db.Exec(`INSERT INTO acca.accounts(acc_id, curr, balance) VALUES ($1, $2, $3)`, acc.AccID, curr, acc.Balance)
					assert.NoErrorf(t, err, "Failed insert account: %+v", acc)
				}
			})

			t.Run("AddNewTransfer", func(t *testing.T) {
				err := db.QueryRow(`SELECT acca.new_transfer($1, $2, $3);`, tt.Transfers, tt.Reason, tt.MetaData).Scan(&tt.TxID)
				assert.NoError(t, err, "Add new transfers")
				assert.NotEmpty(t, tt.TxID)
			})

			t.Run("StatusTransactionBeforeProcess", func(t *testing.T) {
				var status string
				err := db.QueryRow(`SELECT status FROM acca.transactions WHERE tx_id = $1`, tt.TxID).Scan(&status)
				if assert.NoError(t, err) {
					assert.Equal(t, tt.TxStatusBefore, status)
				}
			})

			t.Run("ProcessFromQueue", func(t *testing.T) {
				processFromQueue(t, tt.NumProcess)
			})

			t.Run("StatusTransactionAfterProcess", func(t *testing.T) {
				var status string
				err := db.QueryRow(`SELECT status FROM acca.transactions WHERE tx_id = $1`, tt.TxID).Scan(&status)
				if assert.NoError(t, err) {
					assert.Equal(t, tt.TxStatusAfter, status)
				}
			})

			t.Run("CheckBalances", func(t *testing.T) {
				after := loadBalances(t)
				for accID, balance := range tt.ExpectedBalances {
					assert.Equal(t, int64(balance), int64(after[accID]), "Not equal balances for acc = %v", accID)
				}
			})
		})
	}
}

type testCase2 struct {
	CaseName string

	InitAccounts []accountInfo1

	Commands []transfers

	// approve tx from result of execute command by index
	ApproveCommandIdx []int

	// reject tx from result of execute command by index
	RejectCommandIdx []int

	// filled in as a result of executing commands
	GotTxIDs []int64

	NumProcess1Phase int
	NumProcess2Phase int

	ExpectedTxStatusesBefore1Phase []string
	ExpectedTxStatusesAfter1Phase  []string
	ExpectedTxStatusesAfter2Phase  []string

	ExpectedBalancesAfter1Phase map[string]uint64
	ExpectedBalancesAfter2Phase map[string]uint64
}

// Follow workflow
// - Create accounts with initial balances
// - Execute commands
// - 1 Phase: Execute requests from queue
// - Save recived tx IDs from executed commands
// - Checking statuses for recived tx IDs and interesting balances
// - Approve and reject interested tx (if need)
// - 2 Phase: Execute requests from queue
// - Checking statuses
// - Checking balances
func Test01Basic_01SimpleTransferWithHold(t *testing.T) {
	curr := "cur2"
	t.Run("SetupCurrencies", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO acca.currencies(curr) VALUES ($1)`, curr)
		assert.NoErrorf(t, err, "Failed insert currency: %v", curr)
	})

	tests := []testCase2{}

	for _, tt := range tests {
		t.Run(tt.CaseName, func(t *testing.T) {
			t.Run("SetupAccounts", func(t *testing.T) {
				for _, acc := range tt.InitAccounts {
					_, err := db.Exec(`INSERT INTO acca.accounts(acc_id, curr, balance) VALUES ($1, $2, $3)`, acc.AccID, curr, acc.Balance)
					assert.NoErrorf(t, err, "Failed insert account: %+v", acc)
				}
			})

			t.Run("ExecuteCommands", func(t *testing.T) {
				tt.GotTxIDs = make([]int64, len(tt.Commands), len(tt.Commands))

				for index, cmd := range tt.Commands {
					t.Run("ExecuteCommandWithIndex#"+fmt.Sprint(index), func(t *testing.T) {
						var txID int64
						err := db.QueryRow(`SELECT acca.new_transfer($1, $2, $3);`, cmd, "testing", MetaData{"foo": "bar"}).Scan(&txID)
						assert.NoErrorf(t, err, "Add new transfers for command with index '%d'", index)
						if assert.NotEmpty(t, txID) {
							tt.GotTxIDs[index] = txID
							t.Logf("Recived txID for a command with index '%d': %d", index, txID)
						}
					})
				}

			})

			t.Run("StatusTransactionBefore1Phase", func(t *testing.T) {
				for index, txID := range tt.GotTxIDs {
					if txID == 0 {
						continue
					}

					t.Run("CommandWithIndex#"+fmt.Sprint(index), func(t *testing.T) {
						var got string
						expected := tt.ExpectedTxStatusesBefore1Phase[index]
						err := db.QueryRow(`SELECT status FROM acca.transactions WHERE tx_id = $1`, txID).Scan(&got)
						if assert.NoErrorf(t, err, "Failed get status for txID=%d", txID) {
							assert.Equal(t, expected, got)
						}
					})
				}
			})

			t.Run("1Phase", func(t *testing.T) {
				t.Run("ProcessFromQueue", func(t *testing.T) {
					processFromQueue(t, tt.NumProcess1Phase)
				})

				t.Run("StatusTxsAfter", func(t *testing.T) {
					for index, txID := range tt.GotTxIDs {
						if txID == 0 {
							continue
						}

						t.Run("CommandWithIndex#"+fmt.Sprint(index), func(t *testing.T) {
							var got string
							expected := tt.ExpectedTxStatusesAfter1Phase[index]
							err := db.QueryRow(`SELECT status FROM acca.transactions WHERE tx_id = $1`, txID).Scan(&got)
							if assert.NoErrorf(t, err, "Failed get status for txID=%d", txID) {
								assert.Equal(t, expected, got)
							}
						})
					}
				})

				t.Run("BalancesAfter", func(t *testing.T) {
					after := loadBalances(t)
					for accID, balance := range tt.ExpectedBalancesAfter1Phase {
						assert.Equal(t, int64(balance), int64(after[accID]), "Not equal balances for acc = %v", accID)
					}
				})

				t.Run("ApproveIfNeed", func(t *testing.T) {
					for _, cmdIndex := range tt.ApproveCommandIdx {
						txID := tt.GotTxIDs[cmdIndex]
						if txID <= 0 {
							// should not be
							t.Errorf("Expected a positive txID, got %d", txID)
							continue
						}
						t.Run("ForCommandIndex#"+fmt.Sprint(cmdIndex), func(t *testing.T) {
							_, err := db.Exec(`SELECT acca.accept_tx($1)`, txID)
							assert.NoError(t, err)
						})
					}
				})

				t.Run("RejectIfNeed", func(t *testing.T) {
					for _, cmdIndex := range tt.RejectCommandIdx {
						txID := tt.GotTxIDs[cmdIndex]
						if txID <= 0 {
							// should not be
							t.Errorf("Expected a positive txID, got %d", txID)
							continue
						}
						t.Run("ForCommandIndex#"+fmt.Sprint(cmdIndex), func(t *testing.T) {
							_, err := db.Exec(`SELECT acca.reject_tx($1)`, txID)
							assert.NoError(t, err)
						})
					}
				})

				// end 1phase
			})

			t.Run("2Phase", func(t *testing.T) {
				t.Run("ProcessFromQueue", func(t *testing.T) {
					processFromQueue(t, tt.NumProcess2Phase)
				})

				t.Run("StatusTxsAfter", func(t *testing.T) {
					for index, txID := range tt.GotTxIDs {
						if txID == 0 {
							continue
						}

						t.Run("CommandWithIndex#"+fmt.Sprint(index), func(t *testing.T) {
							var got string
							expected := tt.ExpectedTxStatusesAfter2Phase[index]
							err := db.QueryRow(`SELECT status FROM acca.transactions WHERE tx_id = $1`, txID).Scan(&got)
							if assert.NoErrorf(t, err, "Failed get status for txID=%d", txID) {
								assert.Equal(t, expected, got)
							}
						})
					}
				})

				t.Run("BalancesAfter", func(t *testing.T) {
					after := loadBalances(t)
					for accID, balance := range tt.ExpectedBalancesAfter2Phase {
						assert.Equal(t, int64(balance), int64(after[accID]), "Not equal balances for acc = %v", accID)
					}
				})

				// end 2phase
			})
		})
	}
}
