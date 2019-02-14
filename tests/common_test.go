package tests

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadBalances(t *testing.T) map[string]int64 {
	var balances = map[string]int64{}
	var accID int64
	var key string
	var balance int64
	rows, err := db.Query(`SELECT acc_id, key, balance FROM acca.accounts`)
	require.NoError(t, err, "Failed get acccounts")
	if assert.NoError(t, err) {
		defer rows.Close()
		for rows.Next() {
			err = rows.Scan(&accID, &key, &balance)
			if assert.NoError(t, err) {
				balances[key] = balance
			}
		}
	}
	t.Logf("Balances: %+v", balances)
	return balances
}

type handlerResult struct {
	NumOK  *int64
	NumErr *int64
}

func processFromQueue(t *testing.T, limit int) {
	res := handlerResult{}
	err := db.QueryRow(`SELECT t.ok, t.err FROM acca.handle_requests($1) t;`, limit).Scan(&res.NumOK, &res.NumErr)
	if err != nil {
		t.Fatal("Failed process from queue", err)
	}

	t.Logf("Result handle_requests: ok=%v, err=%v", *res.NumOK, *res.NumErr)
}

var (
	Internal string = "internal"
	Recharge string = "recharge"
	Withdraw string = "withdraw"
)

type MetaData map[string]string

func (md MetaData) Value() (driver.Value, error) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(md); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (md *MetaData) Scan(in interface{}) error {
	switch v := in.(type) {
	case []byte:
		buf := bytes.NewBuffer(v)
		return json.NewDecoder(buf).Decode(md)
	case string:
		buf := bytes.NewBufferString(v)
		return json.NewDecoder(buf).Decode(md)
	default:
		return fmt.Errorf("Not expected type %T", in)
	}
}

type transfer struct {
	SrcAccID  int64    `json:"src_acc_id"`
	SrcAcc    string   `json:"-"`
	DstAccID  int64    `json:"dst_acc_id"`
	DstAcc    string   `json:"-"`
	Type      string   `json:"type"`
	Amount    int64    `json:"amount"`
	Reason    string   `json:"reason"`
	Meta      MetaData `json:"meta"`
	Hold      bool     `json:"hold"`
	HoldAccID int64    `json:"hold_acc_id,omitempty"`
	HoldAcc   string   `json:"-"`
}

type transfers []transfer

func (ts transfers) Value() (driver.Value, error) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(ts); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func runTests(t *testing.T, tests []cmdBatch) {
	for _, tt := range tests {
		state := newTestCaseState()
		for index, cmd := range tt.Commands {
			t.Run(tt.Name+"_#"+fmt.Sprint(index+1), func(t *testing.T) {
				cmdApply(t, state, cmd)
			})
		}
		t.Run(tt.Name+"_#Destroy", func(t *testing.T) {
			t.Logf("TestCase state: %+v", state)
		})
	}
}
