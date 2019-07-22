package strategies

import (
	"context"
	"sync"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/ffsm"
	"github.com/nats-io/nats.go"
	"gopkg.in/reform.v1"
)

type TrStrategyName string

func (s TrStrategyName) String() string { return string(s) }

type InvStrategyName string

func (s InvStrategyName) String() string { return string(s) }

const (
	UNDEFINED_TR       TrStrategyName  = ""
	UNDEFINED_INV      InvStrategyName = ""
	contextReformKeyTX                 = "reform_key_tx"
	contextNatsKey                     = "nats_key"

	UPDATE_INVOICE_SUBJECT     = "update_invoice_subject"
	UPDATE_TRANSACTION_SUBJECT = "update_transaction_subject"
)

var mutex sync.RWMutex
var storeTr = make(map[TrStrategyName]TrStrategy)
var storeInv = make(map[InvStrategyName]InvStrategy)

type TrStrategy interface {
	Dispatch(ctx context.Context, state ffsm.State, payload ffsm.Payload) error
	Name() TrStrategyName
	Provider() engine.Provider
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

func SetNatsToContext(ctx context.Context, nc *nats.EncodedConn) context.Context {
	return context.WithValue(ctx, contextNatsKey, nc)
}

func GetNatsFromContext(ctx context.Context) *nats.EncodedConn {
	return ctx.Value(contextNatsKey).(*nats.EncodedConn)
}

type MessageUpdateTransaction struct {
	TransactionID int64
	Strategy      string
	Status        engine.TransactionStatus
}

type MessageUpdateInvoice struct {
	InvoiceID int64
	Strategy  string
	Status    engine.InvoiceStatus
}
