package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type accountInfo struct {
	AccID   string
	Balance uint64
}

func cmdApply(t *testing.T, cmd command, txIDs *[]int64) {
	newTxIDs := CmdTransfersExecutor(t, cmd.Transfers)
	(*txIDs) = append((*txIDs), newTxIDs...)

	CmdInitAccountsExecutor(t, cmd.InitAccounts)
	CmdApproveExecutor(t, cmd.Approve, (*txIDs))
	CmdRejectExecutor(t, cmd.Reject, (*txIDs))
	CmdRollbackExecutor(t, cmd.Rollback, (*txIDs))
	CmdExecuteExecutor(t, cmd.Execute)
	CmdCheckBalancesExecutor(t, cmd.CheckBalances)
	CmdCheckStatusesExecutor(t, cmd.CheckStatuses, (*txIDs))
	CmdCustomFnExecutor(t, cmd.CustomFn, (*txIDs))
}

type cmdBatch struct {
	Name     string
	Commands []command
}

type command struct {
	InitAccounts  *cmdInitAccounts
	Transfers     *cmdTransfers
	Approve       *cmdApprove
	Reject        *cmdReject
	Rollback      *cmdRollback
	Execute       *cmdExecute
	CheckBalances *cmdCheckBalances
	CheckStatuses *cmdCheckStatuses
	CustomFn      *cmdCustomFn
}

func CmdInitAccounts(currName string, acc []accountInfo) command {
	return command{
		InitAccounts: &cmdInitAccounts{
			CurrName: currName,
			Accounts: acc,
		},
	}
}

func CmdInitAccountsExecutor(t *testing.T, cmd *cmdInitAccounts) {
	if cmd == nil {
		return
	}
	t.Run("InitAccounts", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO acca.currencies(curr) VALUES ($1)`, cmd.CurrName)
		if err != nil {
			assert.True(t, strings.HasPrefix(err.Error(), "pq: duplicate key"))
		}

		for _, acc := range cmd.Accounts {
			_, err := db.Exec(`INSERT INTO acca.accounts(acc_id, curr, balance) VALUES ($1, $2, $3)`, acc.AccID, cmd.CurrName, acc.Balance)
			assert.NoErrorf(t, err, "Failed insert account: %+v", acc)
		}
	})
}

type cmdInitAccounts struct {
	CurrName string
	Accounts []accountInfo
}

func CmdTransfers(trs []transfers) command {
	return command{
		Transfers: &cmdTransfers{
			Transfers: trs,
		},
	}
}

func CmdTransfersExecutor(t *testing.T, cmd *cmdTransfers) []int64 {
	if cmd == nil {
		return []int64{}
	}

	txIDs := make([]int64, len(cmd.Transfers), len(cmd.Transfers))

	t.Run("Transfer", func(t *testing.T) {
		for index, cmd := range cmd.Transfers {
			t.Run("Batch#"+fmt.Sprint(index), func(t *testing.T) {
				var txID int64
				err := db.QueryRow(`SELECT acca.new_transfer($1, $2, $3);`, cmd, "testing", MetaData{"foo": "bar"}).Scan(&txID)
				assert.NoErrorf(t, err, "Add new transfers for batch with index '%d'", index)
				if assert.NotEmpty(t, txID) {
					txIDs[index] = txID
					t.Logf("Recived txID for a batch with index '%d': %d", index, txID)
				}
			})
		}
	})

	return txIDs
}

type cmdTransfers struct {
	Transfers []transfers
}

func CmdApprove(ids ...int) command {
	return command{
		Approve: &cmdApprove{
			IDs: ids,
		},
	}
}

func CmdApproveExecutor(t *testing.T, cmd *cmdApprove, txIDs []int64) {
	if cmd == nil {
		return
	}
	t.Run("Approve", func(t *testing.T) {
		for _, cmdIndex := range cmd.IDs {
			txID := txIDs[cmdIndex]
			if txID <= 0 {
				// should not be
				t.Errorf("Expected a positive txID, got %d", txID)
				continue
			}
			t.Run("ForBatchIndex#"+fmt.Sprint(cmdIndex), func(t *testing.T) {
				_, err := db.Exec(`SELECT acca.accept_tx($1)`, txID)
				assert.NoError(t, err)
			})
		}
	})
}

type cmdApprove struct {
	IDs []int
}

func CmdReject(ids ...int) command {
	return command{
		Reject: &cmdReject{
			IDs: ids,
		},
	}
}

func CmdRejectExecutor(t *testing.T, cmd *cmdReject, txIDs []int64) {
	if cmd == nil {
		return
	}
	t.Run("Reject", func(t *testing.T) {
		for _, cmdIndex := range cmd.IDs {
			txID := txIDs[cmdIndex]
			if txID <= 0 {
				// should not be
				t.Errorf("Expected a positive txID, got %d", txID)
				continue
			}
			t.Run("ForBatchIndex#"+fmt.Sprint(cmdIndex), func(t *testing.T) {
				_, err := db.Exec(`SELECT acca.reject_tx($1)`, txID)
				assert.NoError(t, err)
			})
		}
	})
}

type cmdReject struct {
	IDs []int
}

func CmdRollback(ids ...int) command {
	return command{
		Rollback: &cmdRollback{
			IDs: ids,
		},
	}
}

func CmdRollbackExecutor(t *testing.T, cmd *cmdRollback, txIDs []int64) {
	if cmd == nil {
		return
	}
	t.Run("Rollback", func(t *testing.T) {
		for _, cmdIndex := range cmd.IDs {
			txID := txIDs[cmdIndex]
			if txID <= 0 {
				// should not be
				t.Errorf("Expected a positive txID, got %d", txID)
				continue
			}
			t.Run("ForBatchIndex#"+fmt.Sprint(cmdIndex), func(t *testing.T) {
				_, err := db.Exec(`SELECT acca.rollback_tx($1)`, txID)
				assert.NoError(t, err)
			})
		}
	})
}

type cmdRollback struct {
	IDs []int
}

func CmdExecute(limit int) command {
	return command{
		Execute: &cmdExecute{
			Limit: limit,
		},
	}
}

func CmdExecuteExecutor(t *testing.T, cmd *cmdExecute) {
	if cmd == nil {
		return
	}
	t.Run("Execute", func(t *testing.T) {
		processFromQueue(t, cmd.Limit)
	})
}

type cmdExecute struct {
	Limit int
}

func CmdCheckBalances(expected map[string]uint64) command {
	return command{
		CheckBalances: &cmdCheckBalances{
			Expected: expected,
		},
	}
}

func CmdCheckBalancesExecutor(t *testing.T, cmd *cmdCheckBalances) {
	if cmd == nil {
		return
	}
	t.Run("CheckBalances", func(t *testing.T) {
		after := loadBalances(t)
		for accID, balance := range cmd.Expected {
			assert.Equal(t, int64(balance), int64(after[accID]), "Not equal balances for acc = %v", accID)
		}
	})
}

type cmdCheckBalances struct {
	Expected map[string]uint64
}

func CmdCheckStatuses(expected ...string) command {
	return command{
		CheckStatuses: &cmdCheckStatuses{
			Expected: expected,
		},
	}
}

func CmdCheckStatusesExecutor(t *testing.T, cmd *cmdCheckStatuses, txIDs []int64) {
	if cmd == nil {
		return
	}
	t.Run("CheckStatuses", func(t *testing.T) {
		for index, txID := range txIDs {
			if txID == 0 {
				continue
			}

			t.Run("BatchIndex#"+fmt.Sprint(index), func(t *testing.T) {
				var got string
				expected := cmd.Expected[index]
				err := db.QueryRow(`SELECT status FROM acca.transactions WHERE tx_id = $1`, txID).Scan(&got)
				if assert.NoErrorf(t, err, "Failed get status for txID=%d", txID) {
					assert.Equal(t, expected, got, "Not equal statuses for tx = %v, batch index = %v", txID, index)
				}
			})
		}
	})
}

type cmdCheckStatuses struct {
	Expected []string
}

func CmdCustomFn(fn func(t *testing.T, txIDs []int64)) command {
	return command{
		CustomFn: &cmdCustomFn{
			Fn: fn,
		},
	}
}

func CmdCustomFnExecutor(t *testing.T, cmd *cmdCustomFn, txIDs []int64) {
	if cmd == nil || cmd.Fn == nil {
		return
	}
	t.Run("CustomFn", func(t *testing.T) {
		cmd.Fn(t, txIDs)
	})
}

type cmdCustomFn struct {
	Fn func(t *testing.T, txIDs []int64)
}
