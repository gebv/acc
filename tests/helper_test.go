package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type accountInfo struct {
	AccKey  string
	Balance uint64
}

func cmdApply(t *testing.T, state *testCaseState, cmd command) {
	CmdTransfersExecutor(t, state, cmd.Transfers)

	CmdInitAccountsExecutor(t, state, cmd.InitAccounts)
	CmdApproveExecutor(t, state, cmd.Approve)
	CmdRejectExecutor(t, state, cmd.Reject)
	CmdRollbackExecutor(t, state, cmd.Rollback)
	CmdExecuteExecutor(t, cmd.Execute)
	CmdCheckBalancesExecutor(t, cmd.CheckBalances)
	CmdCheckStatusesExecutor(t, state, cmd.CheckStatuses)
	CmdCustomFnExecutor(t, state, cmd.CustomFn)
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

func CmdInitAccountsExecutor(t *testing.T, state *testCaseState, cmd *cmdInitAccounts) {
	if cmd == nil {
		return
	}
	t.Run("InitAccounts", func(t *testing.T) {
		var curID int64
		err := db.QueryRow(`INSERT INTO acca.currencies(key) VALUES ($1) RETURNING curr_id`, cmd.CurrName).Scan(&curID)

		if err != nil {
			if assert.EqualError(t, err, `pq: duplicate key value violates unique constraint "currencies_key_uniq_idx"`) {
				err := db.QueryRow(`SELECT curr_id FROM acca.currencies WHERE key = $1`, cmd.CurrName).Scan(&curID)
				require.NoErrorf(t, err, "Failed get account by key=%v", cmd.CurrName)
			}
		}

		state.AddCurr(curID, cmd.CurrName)

		for _, acc := range cmd.Accounts {
			var accID int64
			err := db.QueryRow(`INSERT INTO acca.accounts(key, curr_id, balance) VALUES ($1, $2, $3) RETURNING acc_id`, acc.AccKey, curID, acc.Balance).Scan(&accID)

			require.NoError(t, err, "Failed insert new account")
			state.AddAccount(accID, acc.AccKey)
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

func CmdTransfersExecutor(t *testing.T, state *testCaseState, cmd *cmdTransfers) {
	if cmd == nil {
		return
	}

	t.Run("Transfer", func(t *testing.T) {
		for index, cmd := range cmd.Transfers {

			for i, tr := range cmd {
				state.FillTransferAccID(&tr)
				cmd[i] = tr
			}

			t.Run("Batch#"+fmt.Sprint(index), func(t *testing.T) {
				var txID int64
				err := db.QueryRow(`SELECT acca.new_transfer($1, $2, $3);`, cmd, "testing", MetaData{"foo": "bar"}).Scan(&txID)
				require.NoErrorf(t, err, "Add new transfers for batch with index '%d'", index)
				if assert.NotEmpty(t, txID) {
					state.AddTxID(txID)
					t.Logf("Recived txID for a batch with index '%d': %d", index, txID)
				}
			})
		}
	})

	return
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

func CmdApproveExecutor(t *testing.T, state *testCaseState, cmd *cmdApprove) {
	if cmd == nil {
		return
	}
	t.Run("Approve", func(t *testing.T) {
		for _, cmdIndex := range cmd.IDs {
			txID := state.LastTxIDs[cmdIndex]
			if txID <= 0 {
				// should not be
				t.Errorf("Expected a positive txID, got %d", txID)
				continue
			}
			t.Run("ForBatchIndex#"+fmt.Sprint(cmdIndex), func(t *testing.T) {
				_, err := db.Exec(`SELECT acca.accept_tx($1)`, txID)
				require.NoError(t, err, "Failed accept tx")
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

func CmdRejectExecutor(t *testing.T, state *testCaseState, cmd *cmdReject) {
	if cmd == nil {
		return
	}
	t.Run("Reject", func(t *testing.T) {
		for _, cmdIndex := range cmd.IDs {
			txID := state.LastTxIDs[cmdIndex]
			if txID <= 0 {
				// should not be
				t.Errorf("Expected a positive txID, got %d", txID)
				continue
			}
			t.Run("ForBatchIndex#"+fmt.Sprint(cmdIndex), func(t *testing.T) {
				_, err := db.Exec(`SELECT acca.reject_tx($1)`, txID)
				require.NoError(t, err, "Failed reject tx")
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

func CmdRollbackExecutor(t *testing.T, state *testCaseState, cmd *cmdRollback) {
	if cmd == nil {
		return
	}
	t.Run("Rollback", func(t *testing.T) {
		for _, cmdIndex := range cmd.IDs {
			txID := state.LastTxIDs[cmdIndex]
			if txID <= 0 {
				// should not be
				t.Errorf("Expected a positive txID, got %d", txID)
				continue
			}
			t.Run("ForBatchIndex#"+fmt.Sprint(cmdIndex), func(t *testing.T) {
				_, err := db.Exec(`SELECT acca.rollback_tx($1)`, txID)
				assert.NoError(t, err, "Failed rollback tx")
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

func CmdCheckStatusesExecutor(t *testing.T, state *testCaseState, cmd *cmdCheckStatuses) {
	if cmd == nil {
		return
	}
	t.Run("CheckStatuses", func(t *testing.T) {
		for index, txID := range state.LastTxIDs {
			if txID == 0 {
				continue
			}

			t.Run("BatchIndex#"+fmt.Sprint(index), func(t *testing.T) {
				var got string
				expected := cmd.Expected[index]
				err := db.QueryRow(`SELECT status FROM acca.transactions WHERE tx_id = $1`, txID).Scan(&got)
				require.NoError(t, err, "Failed load tx by ID")
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

func CmdCustomFn(fn func(t *testing.T, state *testCaseState)) command {
	return command{
		CustomFn: &cmdCustomFn{
			Fn: fn,
		},
	}
}

func CmdCustomFnExecutor(t *testing.T, state *testCaseState, cmd *cmdCustomFn) {
	if cmd == nil || cmd.Fn == nil {
		return
	}
	t.Run("CustomFn", func(t *testing.T) {
		cmd.Fn(t, state)
	})
}

type cmdCustomFn struct {
	Fn func(t *testing.T, state *testCaseState)
}

func newTestCaseState() *testCaseState {
	return &testCaseState{
		AccIDsToKey:  make(map[int64]string),
		AccKeysToIDs: make(map[string]int64),

		CurrIDsToKey:  make(map[int64]string),
		CurrKeysToIDs: make(map[string]int64),
	}
}

type testCaseState struct {
	LastTxIDs []int64

	AccIDsToKey  map[int64]string
	AccKeysToIDs map[string]int64

	CurrIDsToKey  map[int64]string
	CurrKeysToIDs map[string]int64
}

func (t *testCaseState) AddAccount(accID int64, key string) {
	t.AccIDsToKey[accID] = key
	t.AccKeysToIDs[key] = accID
}

func (t *testCaseState) AddCurr(curID int64, key string) {
	t.CurrIDsToKey[curID] = key
	t.CurrKeysToIDs[key] = curID
}

func (t *testCaseState) AddTxID(txID int64) {
	t.LastTxIDs = append(t.LastTxIDs, txID)
}

func (t *testCaseState) FillTransferAccID(tr *transfer) {
	tr.SrcAccID = t.AccKeysToIDs[tr.SrcAcc]
	tr.DstAccID = t.AccKeysToIDs[tr.DstAcc]
	tr.HoldAccID = t.AccKeysToIDs[tr.HoldAcc]

	if tr.SrcAccID == 0 || tr.DstAccID == 0 {
		panic("should be account ID")
	}

	if len(tr.HoldAcc) > 0 && tr.HoldAccID == 0 {
		panic("should be account ID")
	}
}
