package tests

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var accPrefix int

func loadBalances(t *testing.T) map[string]uint64 {
	var balances = map[string]uint64{}
	var accID string
	var balance uint64
	rows, err := db.Query(`SELECT acc_id, balance FROM acca.accounts`)
	if assert.NoError(t, err) {
		defer rows.Close()
		for rows.Next() {
			err = rows.Scan(&accID, &balance)
			if assert.NoError(t, err) {
				balances[accID] = balance
			}
		}
	}
	t.Logf("Balances: %+v", balances)
	return balances
}

func processFromQueue(t *testing.T, limit int) {
	_, err := db.Exec(`SELECT acca.handle_requests($1);`, limit)
	if err != nil {
		t.Fatal("Failed process from queue", err)
	}
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
	SrcAccID  string   `json:"src_acc_id"`
	DstAccID  string   `json:"dst_acc_id"`
	Type      string   `json:"type"`
	Amount    uint64   `json:"amount"`
	Reason    string   `json:"reason"`
	Meta      MetaData `json:"meta"`
	Hold      bool     `json:"hold"`
	HoldAccID string   `json:"hold_acc_id,omitempty"`
}

type transfers []transfer

func (ts transfers) Value() (driver.Value, error) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(ts); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
