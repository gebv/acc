package strategies

import (
	"context"
	"sync"

	"cloud.google.com/go/firestore"
	"gopkg.in/reform.v1"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/ffsm"
	"github.com/gebv/acca/provider"
)

type TrStrategyName string

func (s TrStrategyName) String() string { return string(s) }

type InvStrategyName string

func (s InvStrategyName) String() string { return string(s) }

const (
	UNDEFINED_TR              TrStrategyName  = ""
	UNDEFINED_INV             InvStrategyName = ""
	contextReformKeyTX                        = "reform_key_tx"
	contextFirestoreTxKey                     = "firestore_tx_key"
	contextFirestoreClientKey                 = "firestore_client_key"

	UPDATE_INVOICE_SUBJECT     = "update_invoice_subject"
	UPDATE_TRANSACTION_SUBJECT = "update_transaction_subject"
)

var mutex sync.RWMutex
var storeTr = make(map[TrStrategyName]TrStrategy)
var storeInv = make(map[InvStrategyName]InvStrategy)

type TrStrategy interface {
	Dispatch(ctx context.Context, state ffsm.State, payload ffsm.Payload) error
	Name() TrStrategyName
	Provider() provider.Provider
	MetaValidation(meta *[]byte) error
}

type InvStrategy interface {
	Dispatch(ctx context.Context, state ffsm.State, payload ffsm.Payload) error
	Name() InvStrategyName
	MetaValidation(meta *[]byte) error
}

func RegTransactionStrategy(s TrStrategy) {
	mutex.Lock()
	defer mutex.Unlock()
	if _, ok := storeTr[s.Name()]; ok {
		panic("name strategy is registered")
	}
	storeTr[s.Name()] = s
}

func RegInvoiceStrategy(s InvStrategy) {
	mutex.Lock()
	defer mutex.Unlock()
	if _, ok := storeInv[s.Name()]; ok {
		panic("name strategy is registered")
	}
	storeInv[s.Name()] = s
}

func ExistTrName(name string) TrStrategyName {
	mutex.Lock()
	defer mutex.Unlock()
	if s, ok := storeTr[TrStrategyName(name)]; ok {
		return s.Name()
	}
	return UNDEFINED_TR
}

func ExistInvName(name string) InvStrategyName {
	mutex.Lock()
	defer mutex.Unlock()
	if s, ok := storeInv[InvStrategyName(name)]; ok {
		return s.Name()
	}
	return UNDEFINED_INV
}

func GetTransactionStrategy(name TrStrategyName) TrStrategy {
	mutex.RLock()
	defer mutex.RUnlock()
	return storeTr[name]
}

func GetInvoiceStrategy(name InvStrategyName) InvStrategy {
	mutex.RLock()
	defer mutex.RUnlock()
	return storeInv[name]
}

func SetTXContext(ctx context.Context, tx *reform.TX) context.Context {
	return context.WithValue(ctx, contextReformKeyTX, tx)
}

func GetTXContext(ctx context.Context) *reform.TX {
	return ctx.Value(contextReformKeyTX).(*reform.TX)
}

func SetFirestoreTxToContext(ctx context.Context, tx *firestore.Transaction) context.Context {
	return context.WithValue(ctx, contextFirestoreTxKey, tx)
}

func GetFirestoreTxFromContext(ctx context.Context) *firestore.Transaction {
	return ctx.Value(contextFirestoreTxKey).(*firestore.Transaction)
}

func SetFirestoreClientToContext(ctx context.Context, fs *firestore.Client) context.Context {
	return context.WithValue(ctx, contextFirestoreClientKey, fs)
}

func GetFirestoreClientFromContext(ctx context.Context) *firestore.Client {
	return ctx.Value(contextFirestoreClientKey).(*firestore.Client)
}

type MessageUpdateTransaction struct {
	ClientID      *int64
	TransactionID int64
	Strategy      string
	Status        engine.TransactionStatus
}

type MessageUpdateInvoice struct {
	ClientID  *int64
	InvoiceID int64
	Strategy  string
	Status    engine.InvoiceStatus
}
