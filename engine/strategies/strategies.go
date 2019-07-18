package strategies

import (
	"context"
	"sync"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/ffsm"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

type StrategyName string

func (s StrategyName) String() string { return string(s) }

const (
	UNDEFINED          StrategyName = ""
	contextReformKeyTX              = "reform_key_tx"
)

var mutex sync.RWMutex
var storeTr map[StrategyName]Strategy
var storeInv map[StrategyName]Strategy

type Strategy interface {
	Dispatch(ctx context.Context, state ffsm.State, payload ffsm.Payload) error
	Name() StrategyName
}

func RegTransactionStrategy(s Strategy) {
	mutex.Lock()
	defer mutex.Unlock()
	if _, ok := storeTr[s.Name()]; ok {
		panic("name strategy is registered")
	}
	storeTr[s.Name()] = s
}

func RegInvoiceStrategy(s Strategy) {
	mutex.Lock()
	defer mutex.Unlock()
	if _, ok := storeInv[s.Name()]; ok {
		panic("name strategy is registered")
	}
	storeInv[s.Name()] = s
}

func ExistTrName(name string) StrategyName {
	mutex.Lock()
	defer mutex.Unlock()
	if s, ok := storeTr[StrategyName(name)]; ok {
		return s.Name()
	}
	return UNDEFINED
}

func ExistInvName(name string) StrategyName {
	mutex.Lock()
	defer mutex.Unlock()
	if s, ok := storeInv[StrategyName(name)]; ok {
		return s.Name()
	}
	return UNDEFINED
}

func GetTransactionStrategy(name StrategyName) Strategy {
	mutex.RLock()
	defer mutex.RUnlock()
	return storeTr[name]
}

func GetInvoiceStrategy(name StrategyName) Strategy {
	mutex.RLock()
	defer mutex.RUnlock()
	return storeInv[name]
}

// TODO убрать метод, содержимое перенести в место вызова.
func DispatchInvoice(ctx context.Context, invID int64, status ffsm.State) error {
	inv := engine.Invoice{}
	if name := ExistInvName(inv.Strategy); name != UNDEFINED {
		if str := GetInvoiceStrategy(name); str != nil {
			return str.Dispatch(ctx, status, invID)
		}
	}
	return errors.New("not_found_strategy_from_invoice:" + inv.Strategy)
}

// TODO убрать метод, содержимое перенести в место вызова.
func DispatchTransaction(ctx context.Context, txID int64, status ffsm.State) error {
	tr := engine.Transaction{}
	if name := ExistTrName(tr.Strategy); name != UNDEFINED {
		if str := GetTransactionStrategy(name); str != nil {
			return str.Dispatch(ctx, status, txID)
		}
	}
	return errors.New("not_found_strategy_from_transaction:" + tr.Strategy)
}

func SetTXContext(ctx context.Context, tx *reform.TX) context.Context {
	return context.WithValue(ctx, contextReformKeyTX, tx)
}

func GetTXContext(ctx context.Context) *reform.TX {
	return ctx.Value(contextReformKeyTX).(*reform.TX)
}
