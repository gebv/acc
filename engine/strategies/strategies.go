package strategies

import (
	"context"

	"github.com/gebv/acca/engine/strategies/for_testing"
	"github.com/gebv/acca/engine/strategies/store"
	"github.com/gebv/acca/ffsm"
	"github.com/pkg/errors"
)

func DispatchInvoice(ctx context.Context, invID int64, status ffsm.State) error {
	inv := for_testing.NoDbFromTest.GetInv(invID)
	if str := store.Get(store.StrategyName(inv.Strategy)); str != nil {
		return str.Dispatch(ctx, status, invID)
	}
	return errors.New("not_found_strategy_from_invoice:" + inv.Strategy)
}

func DispatchTransaction(ctx context.Context, txID int64, status ffsm.State) error {
	tx := for_testing.NoDbFromTest.GetTr(txID)
	if str := store.Get(store.StrategyName(tx.Strategy)); str != nil {
		return str.Dispatch(ctx, status, txID)
	}
	return errors.New("not_found_strategy_from_transaction:" + tx.Strategy)
}
