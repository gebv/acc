package engine

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/gebv/acca/provider"
	"github.com/pkg/errors"
)

//go:generate reform

//reform:acca.view_balance_changes
type ViewBalanceChanges struct {
	ChID              int64                         `reform:"ch_id"`
	TxID              int64                         `reform:"tx_id"`
	AccID             int64                         `reform:"acc_id"`
	CurrID            int64                         `reform:"curr_id"`
	Amount            int64                         `reform:"amount"`
	Balance           int64                         `reform:"balance"`
	BalanceAccepted   int64                         `reform:"balance_accepted"`
	Invoice           InvoiceFromBalanceChanges     `reform:"invoice"`
	Transaction       TransactionFromBalanceChanges `reform:"transaction"`
	Operations        *Operations                   `reform:"operations"`
	Account           AccountFromBalanceChanges     `reform:"actual_account"`
	ActualTransaction TransactionFromBalanceChanges `reform:"actual_transaction"`
}

type AccountFromBalanceChanges struct {
	AccID           int64  `json:"acc_id"`
	Key             string `json:"key"`
	Balance         int64  `json:"balance"`
	BalanceAccepted int64  `json:"balance_accepted"`
}

func (a *AccountFromBalanceChanges) Scan(in interface{}) error {
	switch v := in.(type) {
	case []byte:
		buf := bytes.NewBuffer(v)
		err := json.NewDecoder(buf).Decode(a)
		return errors.Wrap(err, "Failed decode AccountFromBalanceChanges.")
	case string:
		buf := bytes.NewBufferString(v)
		err := json.NewDecoder(buf).Decode(a)
		return errors.Wrap(err, "Failed decode AccountFromBalanceChanges.")
	default:
		return fmt.Errorf("AccountFromBalanceChanges: not expected type %T", in)
	}
}

type InvoiceFromBalanceChanges struct {
	InvoiceID int64         `json:"invoice_id"`
	Key       string        `json:"key"`
	Meta      *[]byte       `json:"meta"`
	Strategy  string        `json:"strategy"`
	Status    InvoiceStatus `json:"status"`
}

func (i *InvoiceFromBalanceChanges) Scan(in interface{}) error {
	switch v := in.(type) {
	case []byte:
		buf := bytes.NewBuffer(v)
		err := json.NewDecoder(buf).Decode(i)
		return errors.Wrap(err, "Failed decode InvoiceFromBalanceChanges.")
	case string:
		buf := bytes.NewBufferString(v)
		err := json.NewDecoder(buf).Decode(i)
		return errors.Wrap(err, "Failed decode InvoiceFromBalanceChanges.")
	default:
		return fmt.Errorf("InvoiceFromBalanceChanges: not expected type %T", in)
	}
}

type TransactionFromBalanceChanges struct {
	TxID               int64             `json:"tx_id"`
	Key                *string           `json:"key"`
	Meta               *[]byte           `json:"meta"`
	Strategy           string            `json:"strategy"`
	Status             TransactionStatus `json:"status"`
	Provider           provider.Provider `json:"provider"`
	ProviderOperID     *string           `json:"provider_oper_id"`
	ProviderOperStatus *string           `json:"provider_oper_status"`
	ProviderOperUrl    *string           `json:"provider_oper_url"`
}

func (t *TransactionFromBalanceChanges) Scan(in interface{}) error {
	switch v := in.(type) {
	case []byte:
		buf := bytes.NewBuffer(v)
		err := json.NewDecoder(buf).Decode(t)
		return errors.Wrap(err, "Failed decode TransactionFromBalanceChanges.")
	case string:
		buf := bytes.NewBufferString(v)
		err := json.NewDecoder(buf).Decode(t)
		return errors.Wrap(err, "Failed decode TransactionFromBalanceChanges.")
	default:
		return fmt.Errorf("TransactionFromBalanceChanges: not expected type %T", in)
	}
}

type Operations []OperationFromBalanceChanges

type OperationFromBalanceChanges struct {
	OperID    int64             `json:"oper_id"`
	SrcAccID  int64             `json:"src_acc_id"`
	DstAccID  int64             `json:"dst_acc_id"`
	Amount    int64             `json:"amount"`
	Strategy  OperationStrategy `json:"strategy"`
	Key       *string           `json:"key"`
	Meta      *[]byte           `json:"meta"`
	Hold      bool              `json:"hold"`
	HoldAccID *int64            `json:"hold_acc_id"`
	Status    OperationStatus   `json:"status"`
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
