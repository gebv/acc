package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test01Basic_Simple(t *testing.T) {
	tests := []testCase{
		{
			"InternalTransferWithoutHold",
			[]string{"cur1"},
			[]accountInfo{
				{
					"cur1",
					"acc1.1",
					10,
				},
				{
					"cur1",
					"acc1.2",
					20,
				},
				{
					"cur1",
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
			"InternalTransferWithHold",
			[]string{"cur2"},
			[]accountInfo{
				{
					"cur2",
					"acc2.1",
					10,
				},
				{
					"cur2",
					"acc2.2",
					20,
				},
				{
					"cur2",
					"acc2.3",
					30,
				},
				{
					"cur2",
					"acc2.hold1",
					0,
				},
			},
			transfers{
				{
					SrcAccID: "acc2.1",
					DstAccID: "acc2.2",
					Type:     Internal,
					Amount:   9,
					Reason:   "fortesting",
					Meta: MetaData{
						"foo": "bar",
					},
					Hold:      true,
					HoldAccID: "acc2.hold1",
				},
				{
					SrcAccID: "acc2.2",
					DstAccID: "acc2.3",
					Type:     Internal,
					Amount:   19,
					Reason:   "fortesting",
					Meta: MetaData{
						"foo": "bar",
					},
					Hold:      true,
					HoldAccID: "acc2.hold1",
				},
			},
			MetaData{"foo": "bar"},
			"testing",
			0,
			map[string]uint64{
				"acc2.1":     10 - 9,
				"acc2.2":     20 - 19,
				"acc2.3":     30,
				"acc2.hold1": 0 + 9 + 19,
			},
			"draft",
			"auth",
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.CaseName, func(t *testing.T) {
			t.Run("SetupCurrencies", func(t *testing.T) {
				for _, curr := range tt.Currencies {
					_, err := db.Exec(`INSERT INTO acca.currencies(curr) VALUES ($1)`, curr)
					assert.NoErrorf(t, err, "Failed insert currency: %v", curr)
				}
			})

			t.Run("SetupAccounts", func(t *testing.T) {
				for _, acc := range tt.Accounts {
					_, err := db.Exec(`INSERT INTO acca.accounts(acc_id, curr, balance) VALUES ($1, $2, $3)`, acc.AccID, acc.Curr, acc.Balance)
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
					assert.Equal(t, int64(after[accID]), int64(balance), "Not equal balances for acc = %v", accID)
				}
			})
		})
	}
}
