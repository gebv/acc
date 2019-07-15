package strategies

import (
	"context"
	"time"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/ffsm"
	"github.com/pkg/errors"
)

var noDbFromTest = struct {
	inv *engine.Invoice
	tr  *engine.Transaction
}{
	inv: &engine.Invoice{
		InvoiceID: 1,
		Key:       "key",
		Status:    engine.DRAFT_I,
		Strategy:  string(SimpleStrategy),
		Meta:      nil,
		Payload:   nil,
		UpdatedAt: time.Time{},
		CreatedAt: time.Time{},
	},
	tr: &engine.Transaction{
		TransactionID:      1,
		InvoiceID:          1,
		Key:                nil,
		Provider:           "prov",
		ProviderOperID:     nil,
		ProviderOperStatus: nil,
		Meta:               nil,
		Status:             engine.DRAFT_TX,
		UpdatedAt:          time.Time{},
		CreatedAt:          time.Time{},
	},
}

var S *Strategy

func init() {
	S = InitStrategies()
}

type StrategyName string

func (s StrategyName) String() string { return string(s) }

const (
	SimpleStrategy StrategyName = "simple_strategy"
)

type TrStrategy interface {
	Change(trID int64, status, nextStatus engine.TransactionStatus) error
}

type InvStrategy interface {
	Change(invID int64, status, nextStatus engine.InvoiceStatus) error
}

type Strategy struct {
	lTrStrategy  map[StrategyName]TrStrategy
	lInvStrategy map[StrategyName]InvStrategy
}

type strategyOfTransaction struct {
	s ffsm.Stack
}

func (s strategyOfTransaction) Change(trID int64, status, nextStatus engine.TransactionStatus) error {
	st := ffsm.State(status)
	fsm := ffsm.MachineFrom(s.s, &st)
	err := fsm.Dispatch(context.Background(), ffsm.State(nextStatus), trID)
	if err != nil {
		return err
	}
	return nil
}

type strategyOfInvoice struct {
	s ffsm.Stack
}

func (s strategyOfInvoice) Change(invID int64, status, nextStatus engine.InvoiceStatus) error {
	st := ffsm.State(status)
	fsm := ffsm.MachineFrom(s.s, &st)
	err := fsm.Dispatch(context.Background(), ffsm.State(nextStatus), invID)
	if err != nil {
		return err
	}
	return nil
}

func InitStrategies() *Strategy {
	lTrStrategy := map[StrategyName]TrStrategy{
		SimpleStrategy: InitTransactionSimpleStrategy(),
	}
	lInvStrategy := map[StrategyName]InvStrategy{
		SimpleStrategy: InitInvoiceSimpleStrategy(),
	}
	return &Strategy{
		lTrStrategy:  lTrStrategy,
		lInvStrategy: lInvStrategy,
	}
}

func SetInvoiceStatus(invID int64, status engine.InvoiceStatus) error {
	// TODO load invoice from BD
	inv := noDbFromTest.inv
	if str, ok := S.lInvStrategy[StrategyName(inv.Strategy)]; ok {
		return str.Change(inv.InvoiceID, inv.Status, status)
	}
	return errors.New("not_found_strategy_from_invoice:" + inv.Strategy)
}

func SetTransactionStatus(trID int64, status engine.TransactionStatus) error {
	// TODO load transaction from BD
	tr := noDbFromTest.tr
	// TODO load invoice from BD
	inv := noDbFromTest.inv
	if str, ok := S.lTrStrategy[StrategyName(inv.Strategy)]; ok {
		return str.Change(tr.TransactionID, tr.Status, status)
	}
	return errors.New("not_found_strategy_from_transaction:" + string(SimpleStrategy))
}

func InitTransactionSimpleStrategy() *strategyOfTransaction {
	s := make(ffsm.Stack)
	s.Add(
		ffsm.State(engine.DRAFT_TX),
		ffsm.State(engine.AUTH_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			tr := noDbFromTest.tr
			// TODO нет стратегии в транзакции.
			//if tr.Strategy != SimpleStrategy.String() {
			//	return ctx, errors.New("Transaction strategy not SimpleStrategy.")
			//}
			if tr.Status != engine.DRAFT_TX {
				return ctx, errors.New("Transaction status not draft.")
			}
			err := SetInvoiceStatus(tr.InvoiceID, engine.AUTH_I)
			if err != nil {
				return ctx, errors.Wrap(err, "Failed set status auth to invoice from transaction")
			}
			return ctx, nil
		},
		"draft>auth",
	)
	s.Add(
		ffsm.State(engine.AUTH_TX),
		ffsm.State(engine.ACCEPTED_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			tr := noDbFromTest.tr
			// TODO нет стратегии в транзакции.
			//if tr.Strategy != SimpleStrategy.String() {
			//	return ctx, errors.New("Transaction strategy not SimpleStrategy.")
			//}
			if tr.Status != engine.AUTH_TX {
				return ctx, errors.New("Transaction status not auth.")
			}
			err := SetInvoiceStatus(tr.InvoiceID, engine.ACCEPTED_I)
			if err != nil {
				return ctx, errors.Wrap(err, "Failed set status accepted to invoice from transaction")
			}
			return ctx, nil
		},
		"auth>accepted",
	)
	s.Add(
		ffsm.State(engine.AUTH_TX),
		ffsm.State(engine.REJECTED_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			tr := noDbFromTest.tr
			// TODO нет стратегии в транзакции.
			//if tr.Strategy != SimpleStrategy.String() {
			//	return ctx, errors.New("Transaction strategy not SimpleStrategy.")
			//}
			if tr.Status != engine.AUTH_TX {
				return ctx, errors.New("Transaction status not auth.")
			}
			err := SetInvoiceStatus(tr.InvoiceID, engine.REJECTED_I)
			if err != nil {
				return ctx, errors.Wrap(err, "Failed set status rejected to invoice from transaction")
			}
			return ctx, nil
		},
		"auth>rejected",
	)
	s.Add(
		ffsm.State(engine.DRAFT_TX),
		ffsm.State(engine.FAILED_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			tr := noDbFromTest.tr
			// TODO нет стратегии в транзакции.
			//if tr.Strategy != SimpleStrategy.String() {
			//	return ctx, errors.New("Transaction strategy not SimpleStrategy.")
			//}
			if tr.Status != engine.DRAFT_TX {
				return ctx, errors.New("Transaction status not draft.")
			}
			err := SetInvoiceStatus(tr.InvoiceID, engine.REJECTED_I)
			if err != nil {
				return ctx, errors.Wrap(err, "Failed set status rejected to invoice from failed transaction")
			}
			return ctx, nil
		},
		"draft>failed",
	)
	s.Add(
		ffsm.State(engine.AUTH_TX),
		ffsm.State(engine.FAILED_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			tr := noDbFromTest.tr
			// TODO нет стратегии в транзакции.
			//if tr.Strategy != SimpleStrategy.String() {
			//	return ctx, errors.New("Transaction strategy not SimpleStrategy.")
			//}
			if tr.Status != engine.AUTH_TX {
				return ctx, errors.New("Transaction status not draft.")
			}
			err := SetInvoiceStatus(tr.InvoiceID, engine.REJECTED_I)
			if err != nil {
				return ctx, errors.Wrap(err, "Failed set status rejected to invoice from failed transaction")
			}
			return ctx, nil
		},
		"auth>failed",
	)
	return &strategyOfTransaction{s: s}
}

func InitInvoiceSimpleStrategy() *strategyOfInvoice {
	s := make(ffsm.Stack)
	s.Add(
		ffsm.State(engine.DRAFT_I),
		ffsm.State(engine.AUTH_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			inv := noDbFromTest.inv
			if inv.Strategy != SimpleStrategy.String() {
				return ctx, errors.New("Invoice strategy not SimpleStrategy.")
			}
			if inv.Status != engine.DRAFT_I {
				return ctx, errors.New("Transaction status not draft.")
			}
			// TODO load transaction from BD
			tr := noDbFromTest.tr
			// TODO нет стратегии в транзакции.
			if tr.Status != engine.DRAFT_TX {
				return ctx, errors.New("Transaction status not draft.")
			}
			tr.Status = engine.AUTH_TX
			inv.Status = engine.AUTH_I
			return ctx, nil
		},
		"draft>auth",
	)
	s.Add(
		ffsm.State(engine.AUTH_I),
		ffsm.State(engine.WAIT_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			inv := noDbFromTest.inv
			if inv.Strategy != SimpleStrategy.String() {
				return ctx, errors.New("Invoice strategy not SimpleStrategy.")
			}
			if inv.Status != engine.AUTH_I {
				return ctx, errors.New("Transaction status not auth.")
			}
			// TODO load transaction from BD
			tr := noDbFromTest.tr
			// TODO нет стратегии в транзакции.
			if tr.Status != engine.AUTH_TX {
				return ctx, errors.New("Transaction status not auth.")
			}
			tr.Status = engine.AUTH_TX
			inv.Status = engine.WAIT_I
			return ctx, nil
		},
		"draft>wait",
	)
	s.Add(
		ffsm.State(engine.AUTH_I),
		ffsm.State(engine.ACCEPTED_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			inv := noDbFromTest.inv
			if inv.Strategy != SimpleStrategy.String() {
				return ctx, errors.New("Invoice strategy not SimpleStrategy.")
			}
			if inv.Status != engine.AUTH_I {
				return ctx, errors.New("Transaction status not auth.")
			}
			// TODO load transaction from BD
			tr := noDbFromTest.tr
			// TODO нет стратегии в транзакции.
			if tr.Status != engine.AUTH_TX {
				return ctx, errors.New("Transaction status not auth.")
			}
			tr.Status = engine.ACCEPTED_TX
			inv.Status = engine.ACCEPTED_I
			return ctx, nil
		},
		"auth>accepted",
	)
	s.Add(
		ffsm.State(engine.AUTH_I),
		ffsm.State(engine.REJECTED_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			inv := noDbFromTest.inv
			if inv.Strategy != SimpleStrategy.String() {
				return ctx, errors.New("Invoice strategy not SimpleStrategy.")
			}
			if inv.Status != engine.AUTH_I {
				return ctx, errors.New("Transaction status not auth.")
			}
			// TODO load transaction from BD
			tr := noDbFromTest.tr
			// TODO нет стратегии в транзакции.
			if tr.Status != engine.AUTH_TX {
				return ctx, errors.New("Transaction status not auth.")
			}
			tr.Status = engine.REJECTED_TX
			inv.Status = engine.REJECTED_I
			return ctx, nil
		},
		"auth>rejected",
	)
	s.Add(
		ffsm.State(engine.WAIT_I),
		ffsm.State(engine.ACCEPTED_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			inv := noDbFromTest.inv
			if inv.Strategy != SimpleStrategy.String() {
				return ctx, errors.New("Invoice strategy not SimpleStrategy.")
			}
			if inv.Status != engine.WAIT_I {
				return ctx, errors.New("Transaction status not wait.")
			}
			// TODO load transaction from BD
			tr := noDbFromTest.tr
			// TODO нет стратегии в транзакции.
			if tr.Status != engine.AUTH_TX {
				return ctx, errors.New("Transaction status not auth.")
			}
			tr.Status = engine.ACCEPTED_TX
			inv.Status = engine.ACCEPTED_I
			return ctx, nil
		},
		"wait>accepted",
	)
	s.Add(
		ffsm.State(engine.WAIT_I),
		ffsm.State(engine.REJECTED_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			inv := noDbFromTest.inv
			if inv.Strategy != SimpleStrategy.String() {
				return ctx, errors.New("Invoice strategy not SimpleStrategy.")
			}
			if inv.Status != engine.WAIT_I {
				return ctx, errors.New("Transaction status not wait.")
			}
			// TODO load transaction from BD
			tr := noDbFromTest.tr
			// TODO нет стратегии в транзакции.
			if tr.Status != engine.AUTH_TX {
				return ctx, errors.New("Transaction status not auth.")
			}
			tr.Status = engine.REJECTED_TX
			inv.Status = engine.REJECTED_I
			return ctx, nil
		},
		"wait>rejected",
	)
	s.Add(
		ffsm.State(engine.WAIT_I),
		ffsm.State(engine.DRAFT_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			inv := noDbFromTest.inv
			if inv.Strategy != SimpleStrategy.String() {
				return ctx, errors.New("Invoice strategy not SimpleStrategy.")
			}
			if inv.Status != engine.WAIT_I {
				return ctx, errors.New("Transaction status not wait.")
			}
			// TODO load transaction from BD
			tr := noDbFromTest.tr
			// TODO нет стратегии в транзакции.
			if tr.Status != engine.AUTH_TX {
				return ctx, errors.New("Transaction status not auth.")
			}
			tr.Status = engine.DRAFT_TX
			inv.Status = engine.DRAFT_I
			return ctx, nil
		},
		"wait>draft",
	)
	s.Add(
		ffsm.State(engine.AUTH_TX),
		ffsm.State(engine.DRAFT_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			inv := noDbFromTest.inv
			if inv.Strategy != SimpleStrategy.String() {
				return ctx, errors.New("Invoice strategy not SimpleStrategy.")
			}
			if inv.Status != engine.AUTH_I {
				return ctx, errors.New("Transaction status not auth.")
			}
			// TODO load transaction from BD
			tr := noDbFromTest.tr
			// TODO нет стратегии в транзакции.
			if tr.Status != engine.AUTH_TX {
				return ctx, errors.New("Transaction status not auth.")
			}
			tr.Status = engine.DRAFT_TX
			inv.Status = engine.DRAFT_I
			return ctx, nil
		},
		"auth>draft",
	)
	return &strategyOfInvoice{s: s}
}
