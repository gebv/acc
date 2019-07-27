package engine

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

//go:generate reform

//reform:acca.view_balance_changes
type ViewBalanceChanges struct {
	ChID            int64                         `reform:"ch_id"`
	TxID            int64                         `reform:"tx_id"`
	AccID           int64                         `reform:"acc_id"`
	Amount          int64                         `reform:"amount"`
	Balance         int64                         `reform:"balance"`
	BalanceAccepted int64                         `reform:"balance_accepted"`
	Account         AccountFromBalanceChanges     `reform:"account"`
	Currency        CurrencyFromBalanceChanges    `reform:"currency"`
	Invoice         InvoiceFromBalanceChanges     `reform:"invoice"`
	Transaction     TransactionFromBalanceChanges `reform:"transaction"`
	Operations      *Operations                   `reform:"operations"`
}

type AccountFromBalanceChanges struct {
	AccID           int64  `json:"acc_id"`
	Key             string `json:"key"`
	Balance         int64  `json:"balance"`
	BalanceAccepted int64  `json:"balance_accepted"`
}

type CurrencyFromBalanceChanges struct {
	CurrID int64  `json:"curr_id"`
	Key    string `json:"key"`
}

type InvoiceFromBalanceChanges struct {
	InvoiceID int64         `json:"invoice_id"`
	Key       string        `json:"key"`
	Strategy  string        `json:"strategy"`
	Status    InvoiceStatus `json:"status"`
}

type TransactionFromBalanceChanges struct {
	TxID               int64             `json:"tx_id"`
	Key                *string           `json:"key"`
	Strategy           string            `json:"strategy"`
	Status             TransactionStatus `json:"status"`
	Provider           Provider          `json:"provider"`
	ProviderOperID     *string           `json:"provider_oper_id"`
	ProviderOperStatus *string           `json:"provider_oper_status"`
	ProviderOperUrl    *string           `json:"provider_oper_url"`
}

type Operations []OperationFromBalanceChanges

type OperationFromBalanceChanges struct {
	OperID    int64             `json:"oper_id"`
	SrcAccID  int64             `json:"src_acc_id"`
	DstAccID  int64             `json:"dst_acc_id"`
	Strategy  OperationStrategy `json:"strategy"`
	Key       *string           `json:"key"`
	Hold      bool              `json:"hold"`
	HoldAccID *int64            `json:"hold_acc_id"`
	Status    OperationStatus   `json:"status"`
}

func (o Operations) Value() (driver.Value, error) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(o); err != nil {
		return nil, errors.Wrap(err, "Failed encode Operations.")
	}
	return buf.Bytes(), nil
}

func (o *Operations) Scan(in interface{}) error {
	switch v := in.(type) {
	case nil:
		return nil
	case []byte:
		buf := bytes.NewBuffer(v)
		err := json.NewDecoder(buf).Decode(o)
		return errors.Wrap(err, "Failed decode Operations.")
	case string:
		buf := bytes.NewBufferString(v)
		err := json.NewDecoder(buf).Decode(o)
		return errors.Wrap(err, "Failed decode Operations.")
	default:
		return fmt.Errorf(" Operations: not expected type %T", in)
	}
}
