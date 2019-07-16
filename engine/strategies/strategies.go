package strategies

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/ffsm"
	"github.com/pkg/errors"
)

var noDbFromTest = newNoDbFromTest()

type noDbFromTestSTR struct {
	rw  sync.RWMutex
	inv map[int64]*engine.Invoice
	tr  map[int64]*engine.Transaction
}

func newNoDbFromTest() *noDbFromTestSTR {
	trKey1 := "1"
	trKey2 := "2"
	return &noDbFromTestSTR{
		inv: map[int64]*engine.Invoice{
			1: {
				InvoiceID: 1,
				Key:       "simple", // TODO определить формат ключа для инвойса.
				Status:    engine.DRAFT_I,
				Strategy:  string(InvSimpleStrategy),
				Meta:      nil,
				Payload:   nil,
				UpdatedAt: time.Time{},
				CreatedAt: time.Time{},
			},
			2: {
				InvoiceID: 2,
				Key:       "recharge", // TODO определить формат ключа для инвойса.
				Status:    engine.DRAFT_I,
				Strategy:  string(InvRechargeStrategy),
				Meta:      nil,
				Payload:   nil,
				UpdatedAt: time.Time{},
				CreatedAt: time.Time{},
			},
		},
		tr: map[int64]*engine.Transaction{
			1: {
				TransactionID:      1,
				InvoiceID:          1,
				Key:                &trKey1,
				Strategy:           string(TrSimpleStrategy),
				Provider:           engine.INTERNAL,
				ProviderOperID:     nil,
				ProviderOperStatus: nil,
				Meta:               nil,
				Status:             engine.DRAFT_TX,
				UpdatedAt:          time.Time{},
				CreatedAt:          time.Time{},
			},
			2: {
				TransactionID:      2,
				InvoiceID:          1,
				Key:                &trKey2,
				Strategy:           string(TrSimpleStrategy),
				Provider:           engine.INTERNAL,
				ProviderOperID:     nil,
				ProviderOperStatus: nil,
				Meta:               nil,
				Status:             engine.DRAFT_TX,
				UpdatedAt:          time.Time{},
				CreatedAt:          time.Time{},
			},
			3: {
				TransactionID:      3,
				InvoiceID:          2,
				Key:                &trKey1,
				Strategy:           string(TrSberbankStrategy),
				Provider:           engine.SBERBANK,
				ProviderOperID:     nil,
				ProviderOperStatus: nil,
				Meta:               nil,
				Status:             engine.DRAFT_TX,
				UpdatedAt:          time.Time{},
				CreatedAt:          time.Time{},
			},
			4: {
				TransactionID:      4,
				InvoiceID:          2,
				Key:                &trKey2,
				Strategy:           string(TrSimpleStrategy),
				Provider:           engine.INTERNAL,
				ProviderOperID:     nil,
				ProviderOperStatus: nil,
				Meta:               nil,
				Status:             engine.DRAFT_TX,
				UpdatedAt:          time.Time{},
				CreatedAt:          time.Time{},
			},
		},
	}
}

func (db *noDbFromTestSTR) GetInv(invID int64) *engine.Invoice {
	db.rw.RLock()
	defer db.rw.RUnlock()
	inv, ok := noDbFromTest.inv[invID]
	if !ok {
		log.Println("Invoice not found id: ", invID)
		return nil
	}
	res := *inv
	return &res
}

func (db *noDbFromTestSTR) SaveInv(inv *engine.Invoice) {
	if inv == nil {
		return
	}
	db.rw.Lock()
	defer db.rw.Unlock()
	noDbFromTest.inv[inv.InvoiceID] = inv
	return
}

func (db *noDbFromTestSTR) ListTr(invID int64) []*engine.Transaction {
	db.rw.RLock()
	defer db.rw.RUnlock()
	res := make([]*engine.Transaction, 0, 2)
	for _, v := range db.tr {
		tr := *v
		if invID == v.InvoiceID {
			res = append(res, &tr)
		}
	}
	return res
}

func (db *noDbFromTestSTR) GetTr(trID int64) *engine.Transaction {
	db.rw.RLock()
	defer db.rw.RUnlock()
	tr, ok := noDbFromTest.tr[trID]
	if !ok {
		log.Println("Transaction not found id: ", trID)
		return nil
	}
	res := *tr
	return &res
}

func (db *noDbFromTestSTR) SaveTr(tr *engine.Transaction) {
	if tr == nil {
		return
	}
	db.rw.Lock()
	defer db.rw.Unlock()
	noDbFromTest.tr[tr.TransactionID] = tr
	return
}

var simRequestToSberbank simRequest

type simRequest struct {
	m    sync.RWMutex
	fail bool
}

func (s *simRequest) setFailRequest(fail bool) {
	s.m.Lock()
	defer s.m.Unlock()
	s.fail = fail
}

func (s *simRequest) requestToSberbank() bool {
	s.m.RLock()
	defer s.m.RUnlock()
	time.Sleep(time.Second)
	if s.fail {
		time.Sleep(time.Second)
	}
	return s.fail
}

var S *Strategy

func init() {
	S = InitStrategies()
}

type StrategyName string

func (s StrategyName) String() string { return string(s) }

const (
	InvSimpleStrategy   StrategyName = "invoice_simple_strategy"
	TrSimpleStrategy    StrategyName = "transaction_simple_strategy"
	InvRechargeStrategy StrategyName = "invoice_recharge_strategy"
	TrSberbankStrategy  StrategyName = "transaction_sberbank_strategy"
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
		TrSimpleStrategy:   InitTransactionSimpleStrategy(),
		TrSberbankStrategy: InitTransactionSberbankStrategy(),
	}
	lInvStrategy := map[StrategyName]InvStrategy{
		InvSimpleStrategy:   InitInvoiceSimpleStrategy(),
		InvRechargeStrategy: InitInvoiceRechargeStrategy(),
	}
	return &Strategy{
		lTrStrategy:  lTrStrategy,
		lInvStrategy: lInvStrategy,
	}
}

func SetInvoiceStatus(invID int64, status engine.InvoiceStatus) error {
	// TODO load invoice from BD
	inv := noDbFromTest.GetInv(invID)
	if inv == nil {
		return errors.New("Invoice not found")
	}
	if str, ok := S.lInvStrategy[StrategyName(inv.Strategy)]; ok {
		return str.Change(inv.InvoiceID, inv.Status, status)
	}
	return errors.New("not_found_strategy_from_invoice:" + inv.Strategy)
}

func SetTransactionStatus(trID int64, status engine.TransactionStatus) error {
	// TODO load transaction from BD
	tr := noDbFromTest.GetTr(trID)
	if tr == nil {
		return errors.New("Transaction not found")
	}
	if str, ok := S.lTrStrategy[StrategyName(tr.Strategy)]; ok {
		return str.Change(tr.TransactionID, tr.Status, status)
	}
	return errors.New("not_found_strategy_from_transaction:" + string(InvSimpleStrategy))
}

func InitTransactionSimpleStrategy() *strategyOfTransaction {
	s := make(ffsm.Stack)
	s.Add(
		ffsm.State(engine.DRAFT_TX),
		ffsm.State(engine.AUTH_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			trID, ok := payload.(int64)
			if !ok {
				log.Println("Transaction bad Payload: ", payload)
				return
			}
			tr := noDbFromTest.GetTr(trID)
			if tr == nil {
				log.Println("Transaction not found id: ", trID)
				return
			}
			if tr.Strategy != TrSimpleStrategy.String() {
				return ctx, errors.New("Transaction strategy not TrSimpleStrategy.")
			}
			if tr.Status != engine.DRAFT_TX {
				return ctx, errors.New("Transaction status not draft.")
			}
			// Установить статус куда происходит переход
			ns := engine.AUTH_TX
			tr.NextStatus = &ns
			noDbFromTest.SaveTr(tr)
			// TODO добавить необходимые операции по транзакции.
			// Установить статус после проделанных операций
			tr = noDbFromTest.GetTr(trID)
			tr.Status = engine.AUTH_TX
			tr.NextStatus = nil
			noDbFromTest.SaveTr(tr)
			return ctx, nil
		},
		"draft>auth",
	)
	s.Add(
		ffsm.State(engine.AUTH_TX),
		ffsm.State(engine.HOLD_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			trID, ok := payload.(int64)
			if !ok {
				log.Println("Transaction bad Payload: ", payload)
				return
			}
			tr := noDbFromTest.GetTr(trID)
			if tr == nil {
				log.Println("Transaction not found id: ", trID)
				return
			}
			if tr.Strategy != TrSimpleStrategy.String() {
				return ctx, errors.New("Transaction strategy not TrSimpleStrategy.")
			}
			if tr.Status != engine.AUTH_TX {
				return ctx, errors.New("Transaction status not auth.")
			}
			// Установить статус куда происходит переход
			ns := engine.HOLD_TX
			tr.NextStatus = &ns
			noDbFromTest.SaveTr(tr)
			// TODO добавить необходимые операции по транзакции.
			// Установить статус после проделанных операций
			tr = noDbFromTest.GetTr(trID)
			tr.Status = engine.HOLD_TX
			tr.NextStatus = nil
			noDbFromTest.SaveTr(tr)
			return ctx, nil
		},
		"auth>hold",
	)
	s.Add(
		ffsm.State(engine.HOLD_TX),
		ffsm.State(engine.ACCEPTED_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			trID, ok := payload.(int64)
			if !ok {
				log.Println("Transaction bad Payload: ", payload)
				return
			}
			tr := noDbFromTest.GetTr(trID)
			if tr == nil {
				log.Println("Transaction not found id: ", trID)
				return
			}
			if tr.Strategy != TrSimpleStrategy.String() {
				return ctx, errors.New("Transaction strategy not TrSimpleStrategy.")
			}
			if tr.Status != engine.HOLD_TX {
				return ctx, errors.New("Transaction status not hold.")
			}
			// Установить статус куда происходит переход
			ns := engine.ACCEPTED_TX
			tr.NextStatus = &ns
			noDbFromTest.SaveTr(tr)
			// TODO добавить необходимые операции по транзакции.
			// Установить статус после проделанных операций
			tr = noDbFromTest.GetTr(trID)
			tr.Status = engine.ACCEPTED_TX
			tr.NextStatus = nil
			noDbFromTest.SaveTr(tr)
			return ctx, nil
		},
		"hold>accepted",
	)
	s.Add(
		ffsm.State(engine.AUTH_TX),
		ffsm.State(engine.ACCEPTED_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			trID, ok := payload.(int64)
			if !ok {
				log.Println("Transaction bad Payload: ", payload)
				return
			}
			tr := noDbFromTest.GetTr(trID)
			if tr == nil {
				log.Println("Transaction not found id: ", trID)
				return
			}
			if tr.Strategy != TrSimpleStrategy.String() {
				return ctx, errors.New("Transaction strategy not TrSimpleStrategy.")
			}
			if tr.Status != engine.AUTH_TX {
				return ctx, errors.New("Transaction status not auth.")
			}
			// Установить статус куда происходит переход
			ns := engine.ACCEPTED_TX
			tr.NextStatus = &ns
			noDbFromTest.SaveTr(tr)
			// TODO добавить необходимые операции по транзакции.
			// Установить статус после проделанных операций
			tr = noDbFromTest.GetTr(trID)
			tr.Status = engine.ACCEPTED_TX
			tr.NextStatus = nil
			noDbFromTest.SaveTr(tr)
			return ctx, nil
		},
		"auth>accepted",
	)
	s.Add(
		ffsm.State(engine.AUTH_TX),
		ffsm.State(engine.REJECTED_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			trID, ok := payload.(int64)
			if !ok {
				log.Println("Transaction bad Payload: ", payload)
				return
			}
			tr := noDbFromTest.GetTr(trID)
			if tr == nil {
				log.Println("Transaction not found id: ", trID)
				return
			}
			if tr.Strategy != TrSimpleStrategy.String() {
				return ctx, errors.New("Transaction strategy not TrSimpleStrategy.")
			}
			if tr.Status != engine.AUTH_TX {
				return ctx, errors.New("Transaction status not auth.")
			}
			// Установить статус куда происходит переход
			ns := engine.REJECTED_TX
			tr.NextStatus = &ns
			noDbFromTest.SaveTr(tr)
			// TODO добавить необходимые операции по транзакции.
			// Установить статус после проделанных операций
			tr = noDbFromTest.GetTr(trID)
			tr.Status = engine.REJECTED_TX
			tr.NextStatus = nil
			noDbFromTest.SaveTr(tr)
			return ctx, nil
		},
		"auth>rejected",
	)
	s.Add(
		ffsm.State(engine.HOLD_TX),
		ffsm.State(engine.REJECTED_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			trID, ok := payload.(int64)
			if !ok {
				log.Println("Transaction bad Payload: ", payload)
				return
			}
			tr := noDbFromTest.GetTr(trID)
			if tr == nil {
				log.Println("Transaction not found id: ", trID)
				return
			}
			if tr.Strategy != TrSimpleStrategy.String() {
				return ctx, errors.New("Transaction strategy not TrSimpleStrategy.")
			}
			if tr.Status != engine.HOLD_TX {
				return ctx, errors.New("Transaction status not hold.")
			}
			// Установить статус куда происходит переход
			ns := engine.REJECTED_TX
			tr.NextStatus = &ns
			noDbFromTest.SaveTr(tr)
			// TODO добавить необходимые операции по транзакции.
			// Установить статус после проделанных операций
			tr = noDbFromTest.GetTr(trID)
			tr.Status = engine.REJECTED_TX
			tr.NextStatus = nil
			noDbFromTest.SaveTr(tr)
			return ctx, nil
		},
		"hold>rejected",
	)
	s.Add(
		ffsm.State(engine.DRAFT_TX),
		ffsm.State(engine.FAILED_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			trID, ok := payload.(int64)
			if !ok {
				log.Println("Transaction bad Payload: ", payload)
				return
			}
			tr := noDbFromTest.GetTr(trID)
			if tr == nil {
				log.Println("Transaction not found id: ", trID)
				return
			}
			if tr.Strategy != TrSimpleStrategy.String() {
				return ctx, errors.New("Transaction strategy not TrSimpleStrategy.")
			}
			if tr.Status != engine.DRAFT_TX {
				return ctx, errors.New("Transaction status not draft.")
			}
			// Установить статус куда происходит переход
			ns := engine.FAILED_TX
			tr.NextStatus = &ns
			noDbFromTest.SaveTr(tr)
			// TODO добавить необходимые операции по транзакции.
			// Установить статус после проделанных операций
			tr = noDbFromTest.GetTr(trID)
			tr.Status = engine.FAILED_TX
			tr.NextStatus = nil
			noDbFromTest.SaveTr(tr)
			return ctx, nil
		},
		"draft>failed",
	)
	s.Add(
		ffsm.State(engine.DRAFT_TX),
		ffsm.State(engine.REJECTED_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			trID, ok := payload.(int64)
			if !ok {
				log.Println("Transaction bad Payload: ", payload)
				return
			}
			tr := noDbFromTest.GetTr(trID)
			if tr == nil {
				log.Println("Transaction not found id: ", trID)
				return
			}
			if tr.Strategy != TrSimpleStrategy.String() {
				return ctx, errors.New("Transaction strategy not TrSimpleStrategy.")
			}
			if tr.Status != engine.DRAFT_TX {
				return ctx, errors.New("Transaction status not draft.")
			}
			// Установить статус куда происходит переход
			ns := engine.REJECTED_TX
			tr.NextStatus = &ns
			noDbFromTest.SaveTr(tr)
			// TODO добавить необходимые операции по транзакции.
			// Установить статус после проделанных операций
			tr = noDbFromTest.GetTr(trID)
			tr.Status = engine.REJECTED_TX
			tr.NextStatus = nil
			noDbFromTest.SaveTr(tr)
			return ctx, nil
		},
		"draft>rejected",
	)
	s.Add(
		ffsm.State(engine.AUTH_TX),
		ffsm.State(engine.FAILED_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			trID, ok := payload.(int64)
			if !ok {
				log.Println("Transaction bad Payload: ", payload)
				return
			}
			tr := noDbFromTest.GetTr(trID)
			if tr == nil {
				log.Println("Transaction not found id: ", trID)
				return
			}
			if tr.Strategy != TrSimpleStrategy.String() {
				return ctx, errors.New("Transaction strategy not TrSimpleStrategy.")
			}
			if tr.Status != engine.AUTH_TX {
				return ctx, errors.New("Transaction status not draft.")
			}
			// Установить статус куда происходит переход
			ns := engine.FAILED_TX
			tr.NextStatus = &ns
			noDbFromTest.SaveTr(tr)
			// TODO добавить необходимые операции по транзакции.
			// Установить статус после проделанных операций
			tr = noDbFromTest.GetTr(trID)
			tr.Status = engine.FAILED_TX
			tr.NextStatus = nil
			noDbFromTest.SaveTr(tr)
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
			invID, ok := payload.(int64)
			if !ok {
				log.Println("Invoice bad Payload: ", payload)
				return
			}
			inv := noDbFromTest.GetInv(invID)
			if inv == nil {
				log.Println("Invoice not found id: ", invID)
				return
			}
			if inv.Strategy != InvSimpleStrategy.String() {
				return ctx, errors.New("Invoice strategy not InvSimpleStrategy.")
			}
			if inv.Status != engine.DRAFT_I {
				return ctx, errors.New("Invoice status not draft.")
			}
			// Установить статус куда происходит переход
			ns := engine.AUTH_I
			inv.NextStatus = &ns
			noDbFromTest.SaveInv(inv)
			// TODO load transactions from DB
			next := true
			for _, v := range noDbFromTest.ListTr(invID) {
				if !v.Status.Match(engine.AUTH_TX) {
					next = false
					break
				}
			}
			if !next {
				return ctx, nil
			}
			// Установить статус после проделанных операций
			inv = noDbFromTest.GetInv(invID)
			inv.Status = engine.AUTH_I
			inv.NextStatus = nil
			noDbFromTest.SaveInv(inv)
			return ctx, nil
		},
		"draft>auth",
	)
	s.Add(
		ffsm.State(engine.DRAFT_I),
		ffsm.State(engine.REJECTED_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			invID, ok := payload.(int64)
			if !ok {
				log.Println("Invoice bad Payload: ", payload)
				return
			}
			inv := noDbFromTest.GetInv(invID)
			if inv == nil {
				log.Println("Invoice not found id: ", invID)
				return
			}
			if inv.Strategy != InvSimpleStrategy.String() {
				return ctx, errors.New("Invoice strategy not InvSimpleStrategy.")
			}
			if inv.Status != engine.DRAFT_I {
				return ctx, errors.New("Invoice status not draft.")
			}
			// Установить статус куда происходит переход
			ns := engine.REJECTED_I
			inv.NextStatus = &ns
			noDbFromTest.SaveInv(inv)
			// TODO load transactions from DB
			next := true
			for _, v := range noDbFromTest.ListTr(invID) {
				if !v.Status.Match(engine.REJECTED_TX) {
					next = false
					break
				}
			}
			if !next {
				return ctx, nil
			}
			// Установить статус после проделанных операций
			inv = noDbFromTest.GetInv(invID)
			inv.Status = engine.REJECTED_I
			inv.NextStatus = nil
			noDbFromTest.SaveInv(inv)
			return ctx, nil
		},
		"draft>rejected",
	)
	s.Add(
		ffsm.State(engine.AUTH_I),
		ffsm.State(engine.WAIT_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			invID, ok := payload.(int64)
			if !ok {
				log.Println("Invoice bad Payload: ", payload)
				return
			}
			inv := noDbFromTest.GetInv(invID)
			if inv == nil {
				log.Println("Invoice not found id: ", invID)
				return
			}
			if inv.Strategy != InvSimpleStrategy.String() {
				return ctx, errors.New("Invoice strategy not InvSimpleStrategy.")
			}
			if inv.Status != engine.AUTH_I {
				return ctx, errors.New("Invoice status not auth.")
			}
			// Установить статус куда происходит переход
			ns := engine.WAIT_I
			inv.NextStatus = &ns
			noDbFromTest.SaveInv(inv)
			// TODO load transactions from DB
			next := true
			for _, v := range noDbFromTest.ListTr(invID) {
				if !v.Status.Match(engine.HOLD_TX) {
					next = false
					break
				}
			}
			if !next {
				return ctx, nil
			}
			// Установить статус после проделанных операций
			inv = noDbFromTest.GetInv(invID)
			inv.Status = engine.WAIT_I
			inv.NextStatus = nil
			noDbFromTest.SaveInv(inv)
			return ctx, nil
		},
		"auth>wait",
	)
	s.Add(
		ffsm.State(engine.AUTH_I),
		ffsm.State(engine.ACCEPTED_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			invID, ok := payload.(int64)
			if !ok {
				log.Println("Invoice bad Payload: ", payload)
				return
			}
			inv := noDbFromTest.GetInv(invID)
			if inv == nil {
				log.Println("Invoice not found id: ", invID)
				return
			}
			if inv.Strategy != InvSimpleStrategy.String() {
				return ctx, errors.New("Invoice strategy not InvSimpleStrategy.")
			}
			if inv.Status != engine.AUTH_I {
				return ctx, errors.New("Invoice status not auth.")
			}
			// Установить статус куда происходит переход
			ns := engine.ACCEPTED_I
			inv.NextStatus = &ns
			noDbFromTest.SaveInv(inv)
			// TODO load transactions from DB
			next := true
			for _, v := range noDbFromTest.ListTr(invID) {
				if !v.Status.Match(engine.ACCEPTED_TX) {
					next = false
					break
				}
			}
			if !next {
				return ctx, nil
			}
			// Установить статус после проделанных операций
			inv = noDbFromTest.GetInv(invID)
			inv.Status = engine.ACCEPTED_I
			inv.NextStatus = nil
			noDbFromTest.SaveInv(inv)
			return ctx, nil
		},
		"auth>accepted",
	)
	s.Add(
		ffsm.State(engine.AUTH_I),
		ffsm.State(engine.REJECTED_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			invID, ok := payload.(int64)
			if !ok {
				log.Println("Invoice bad Payload: ", payload)
				return
			}
			inv := noDbFromTest.GetInv(invID)
			if inv == nil {
				log.Println("Invoice not found id: ", invID)
				return
			}
			if inv.Strategy != InvSimpleStrategy.String() {
				return ctx, errors.New("Invoice strategy not InvSimpleStrategy.")
			}
			if inv.Status != engine.AUTH_I {
				return ctx, errors.New("Invoice status not auth.")
			}
			// Установить статус куда происходит переход
			ns := engine.REJECTED_I
			inv.NextStatus = &ns
			noDbFromTest.SaveInv(inv)
			// TODO load transactions from DB
			next := true
			for _, v := range noDbFromTest.ListTr(invID) {
				if !v.Status.Match(engine.REJECTED_TX) {
					next = false
					break
				}
			}
			if !next {
				return ctx, nil
			}
			// Установить статус после проделанных операций
			inv = noDbFromTest.GetInv(invID)
			inv.Status = engine.REJECTED_I
			inv.NextStatus = nil
			noDbFromTest.SaveInv(inv)
			return ctx, nil
		},
		"auth>rejected",
	)
	s.Add(
		ffsm.State(engine.WAIT_I),
		ffsm.State(engine.ACCEPTED_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			invID, ok := payload.(int64)
			if !ok {
				log.Println("Invoice bad Payload: ", payload)
				return
			}
			inv := noDbFromTest.GetInv(invID)
			if inv == nil {
				log.Println("Invoice not found id: ", invID)
				return
			}
			if inv.Strategy != InvSimpleStrategy.String() {
				return ctx, errors.New("Invoice strategy not InvSimpleStrategy.")
			}
			if inv.Status != engine.WAIT_I {
				return ctx, errors.New("Invoice status not wait.")
			}
			// Установить статус куда происходит переход
			ns := engine.ACCEPTED_I
			inv.NextStatus = &ns
			noDbFromTest.SaveInv(inv)
			// TODO load transactions from DB
			next := true
			for _, v := range noDbFromTest.ListTr(invID) {
				if !v.Status.Match(engine.ACCEPTED_TX) {
					next = false
					break
				}
			}
			if !next {
				return ctx, nil
			}
			// Установить статус после проделанных операций
			inv = noDbFromTest.GetInv(invID)
			inv.Status = engine.ACCEPTED_I
			inv.NextStatus = nil
			noDbFromTest.SaveInv(inv)
			return ctx, nil
		},
		"wait>accepted",
	)
	s.Add(
		ffsm.State(engine.WAIT_I),
		ffsm.State(engine.REJECTED_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			invID, ok := payload.(int64)
			if !ok {
				log.Println("Invoice bad Payload: ", payload)
				return
			}
			inv := noDbFromTest.GetInv(invID)
			if inv == nil {
				log.Println("Invoice not found id: ", invID)
				return
			}
			if inv.Strategy != InvSimpleStrategy.String() {
				return ctx, errors.New("Invoice strategy not InvSimpleStrategy.")
			}
			if inv.Status != engine.WAIT_I {
				return ctx, errors.New("Invoice status not wait.")
			}
			// Установить статус куда происходит переход
			ns := engine.REJECTED_I
			inv.NextStatus = &ns
			noDbFromTest.SaveInv(inv)
			// TODO load transactions from DB
			next := true
			for _, v := range noDbFromTest.ListTr(invID) {
				if !v.Status.Match(engine.REJECTED_TX) {
					next = false
					break
				}
			}
			if !next {
				return ctx, nil
			}
			// Установить статус после проделанных операций
			inv = noDbFromTest.GetInv(invID)
			inv.Status = engine.REJECTED_I
			inv.NextStatus = nil
			noDbFromTest.SaveInv(inv)
			return ctx, nil
		},
		"wait>rejected",
	)
	s.Add(
		ffsm.State(engine.WAIT_I),
		ffsm.State(engine.DRAFT_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			invID, ok := payload.(int64)
			if !ok {
				log.Println("Invoice bad Payload: ", payload)
				return
			}
			inv := noDbFromTest.GetInv(invID)
			if inv == nil {
				log.Println("Invoice not found id: ", invID)
				return
			}
			if inv.Strategy != InvSimpleStrategy.String() {
				return ctx, errors.New("Invoice strategy not InvSimpleStrategy.")
			}
			if inv.Status != engine.WAIT_I {
				return ctx, errors.New("Invoice status not wait.")
			}
			// Установить статус куда происходит переход
			ns := engine.DRAFT_I
			inv.NextStatus = &ns
			noDbFromTest.SaveInv(inv)
			// TODO load transactions from DB
			next := true
			for _, v := range noDbFromTest.ListTr(invID) {
				if !v.Status.Match(engine.DRAFT_TX) {
					next = false
					break
				}
			}
			if !next {
				return ctx, nil
			}
			// Установить статус после проделанных операций
			inv = noDbFromTest.GetInv(invID)
			inv.Status = engine.DRAFT_I
			inv.NextStatus = nil
			noDbFromTest.SaveInv(inv)
			return ctx, nil
		},
		"wait>draft",
	)
	s.Add(
		ffsm.State(engine.AUTH_TX),
		ffsm.State(engine.DRAFT_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			invID, ok := payload.(int64)
			if !ok {
				log.Println("Invoice bad Payload: ", payload)
				return
			}
			inv := noDbFromTest.GetInv(invID)
			if inv == nil {
				log.Println("Invoice not found id: ", invID)
				return
			}
			if inv.Strategy != InvSimpleStrategy.String() {
				return ctx, errors.New("Invoice strategy not InvSimpleStrategy.")
			}
			if inv.Status != engine.AUTH_I {
				return ctx, errors.New("Invoice status not auth.")
			}
			// Установить статус куда происходит переход
			ns := engine.DRAFT_I
			inv.NextStatus = &ns
			noDbFromTest.SaveInv(inv)
			// TODO load transactions from DB
			next := true
			for _, v := range noDbFromTest.ListTr(invID) {
				if !v.Status.Match(engine.DRAFT_TX) {
					next = false
					break
				}
			}
			if !next {
				return ctx, nil
			}
			// Установить статус после проделанных операций
			inv = noDbFromTest.GetInv(invID)
			inv.Status = engine.DRAFT_I
			inv.NextStatus = nil
			noDbFromTest.SaveInv(inv)
			return ctx, nil
		},
		"auth>draft",
	)
	return &strategyOfInvoice{s: s}
}

func InitInvoiceRechargeStrategy() *strategyOfInvoice {
	s := make(ffsm.Stack)
	s.Add(
		ffsm.State(engine.DRAFT_I),
		ffsm.State(engine.AUTH_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			invID, ok := payload.(int64)
			if !ok {
				log.Println("Invoice bad Payload: ", payload)
				return
			}
			inv := noDbFromTest.GetInv(invID)
			if inv == nil {
				log.Println("Invoice not found id: ", invID)
				return
			}
			if inv.Strategy != InvRechargeStrategy.String() {
				return ctx, errors.New("Invoice strategy not InvRechargeStrategy.")
			}
			if inv.Status != engine.DRAFT_I {
				return ctx, errors.New("Invoice status not draft.")
			}
			// Установить статус куда происходит переход
			ns := engine.AUTH_I
			inv.NextStatus = &ns
			noDbFromTest.SaveInv(inv)
			// TODO load transactions from DB
			next := true
			for _, v := range noDbFromTest.ListTr(invID) {
				if !v.Status.Match(engine.AUTH_TX) {
					next = false
					break
				}
			}
			if !next {
				return ctx, nil
			}
			// Установить статус после проделанных операций
			inv = noDbFromTest.GetInv(invID)
			inv.Status = engine.AUTH_I
			inv.NextStatus = nil
			noDbFromTest.SaveInv(inv)
			return ctx, nil
		},
		"draft>auth",
	)
	s.Add(
		ffsm.State(engine.DRAFT_I),
		ffsm.State(engine.REJECTED_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			invID, ok := payload.(int64)
			if !ok {
				log.Println("Invoice bad Payload: ", payload)
				return
			}
			inv := noDbFromTest.GetInv(invID)
			if inv == nil {
				log.Println("Invoice not found id: ", invID)
				return
			}
			if inv.Strategy != InvRechargeStrategy.String() {
				return ctx, errors.New("Invoice strategy not InvRechargeStrategy.")
			}
			if inv.Status != engine.DRAFT_I {
				return ctx, errors.New("Invoice status not draft.")
			}
			// Установить статус куда происходит переход
			ns := engine.REJECTED_I
			inv.NextStatus = &ns
			noDbFromTest.SaveInv(inv)
			// TODO load transactions from DB
			next := true
			for _, v := range noDbFromTest.ListTr(invID) {
				if !v.Status.Match(engine.REJECTED_TX) {
					next = false
					break
				}
			}
			if !next {
				return ctx, nil
			}
			// Установить статус после проделанных операций
			inv = noDbFromTest.GetInv(invID)
			inv.Status = engine.REJECTED_I
			inv.NextStatus = nil
			noDbFromTest.SaveInv(inv)
			return ctx, nil
		},
		"draft>rejected",
	)
	s.Add(
		ffsm.State(engine.AUTH_I),
		ffsm.State(engine.WAIT_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			invID, ok := payload.(int64)
			if !ok {
				log.Println("Invoice bad Payload: ", payload)
				return
			}
			inv := noDbFromTest.GetInv(invID)
			if inv == nil {
				log.Println("Invoice not found id: ", invID)
				return
			}
			if inv.Strategy != InvRechargeStrategy.String() {
				return ctx, errors.New("Invoice strategy not InvRechargeStrategy.")
			}
			if inv.Status != engine.AUTH_I {
				return ctx, errors.New("Invoice status not auth.")
			}
			// Установить статус куда происходит переход
			ns := engine.WAIT_I
			inv.NextStatus = &ns
			noDbFromTest.SaveInv(inv)
			// TODO load transactions from DB
			next := true
			for _, v := range noDbFromTest.ListTr(invID) {
				if !v.Status.Match(engine.HOLD_TX) {
					next = false
					break
				}
			}
			if !next {
				return ctx, nil
			}
			// Установить статус после проделанных операций
			inv = noDbFromTest.GetInv(invID)
			inv.Status = engine.WAIT_I
			inv.NextStatus = nil
			noDbFromTest.SaveInv(inv)
			return ctx, nil
		},
		"auth>wait",
	)
	s.Add(
		ffsm.State(engine.AUTH_I),
		ffsm.State(engine.ACCEPTED_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			invID, ok := payload.(int64)
			if !ok {
				log.Println("Invoice bad Payload: ", payload)
				return
			}
			inv := noDbFromTest.GetInv(invID)
			if inv == nil {
				log.Println("Invoice not found id: ", invID)
				return
			}
			if inv.Strategy != InvRechargeStrategy.String() {
				return ctx, errors.New("Invoice strategy not InvRechargeStrategy.")
			}
			if inv.Status != engine.AUTH_I {
				return ctx, errors.New("Invoice status not auth.")
			}
			// Установить статус куда происходит переход
			ns := engine.ACCEPTED_I
			inv.NextStatus = &ns
			noDbFromTest.SaveInv(inv)
			// TODO load transactions from DB
			next := true
			for _, v := range noDbFromTest.ListTr(invID) {
				if !v.Status.Match(engine.ACCEPTED_TX) {
					next = false
					break
				}
			}
			if !next {
				return ctx, nil
			}
			// Установить статус после проделанных операций
			inv = noDbFromTest.GetInv(invID)
			inv.Status = engine.ACCEPTED_I
			inv.NextStatus = nil
			noDbFromTest.SaveInv(inv)
			return ctx, nil
		},
		"auth>accepted",
	)
	s.Add(
		ffsm.State(engine.AUTH_I),
		ffsm.State(engine.REJECTED_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			invID, ok := payload.(int64)
			if !ok {
				log.Println("Invoice bad Payload: ", payload)
				return
			}
			inv := noDbFromTest.GetInv(invID)
			if inv == nil {
				log.Println("Invoice not found id: ", invID)
				return
			}
			if inv.Strategy != InvRechargeStrategy.String() {
				return ctx, errors.New("Invoice strategy not InvRechargeStrategy.")
			}
			if inv.Status != engine.AUTH_I {
				return ctx, errors.New("Invoice status not auth.")
			}
			// Установить статус куда происходит переход
			ns := engine.REJECTED_I
			inv.NextStatus = &ns
			noDbFromTest.SaveInv(inv)
			// TODO load transactions from DB
			next := true
			for _, v := range noDbFromTest.ListTr(invID) {
				if !v.Status.Match(engine.REJECTED_TX) {
					next = false
					break
				}
			}
			if !next {
				return ctx, nil
			}
			// Установить статус после проделанных операций
			inv = noDbFromTest.GetInv(invID)
			inv.Status = engine.REJECTED_I
			inv.NextStatus = nil
			noDbFromTest.SaveInv(inv)
			return ctx, nil
		},
		"auth>rejected",
	)
	s.Add(
		ffsm.State(engine.WAIT_I),
		ffsm.State(engine.ACCEPTED_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			invID, ok := payload.(int64)
			if !ok {
				log.Println("Invoice bad Payload: ", payload)
				return
			}
			inv := noDbFromTest.GetInv(invID)
			if inv == nil {
				log.Println("Invoice not found id: ", invID)
				return
			}
			if inv.Strategy != InvRechargeStrategy.String() {
				return ctx, errors.New("Invoice strategy not InvRechargeStrategy.")
			}
			if inv.Status != engine.WAIT_I {
				return ctx, errors.New("Invoice status not wait.")
			}
			// Установить статус куда происходит переход
			ns := engine.ACCEPTED_I
			inv.NextStatus = &ns
			noDbFromTest.SaveInv(inv)
			// TODO load transactions from DB
			next := true
			for _, v := range noDbFromTest.ListTr(invID) {
				if !v.Status.Match(engine.ACCEPTED_TX) {
					next = false
					break
				}
			}
			if !next {
				return ctx, nil
			}
			// Установить статус после проделанных операций
			inv = noDbFromTest.GetInv(invID)
			inv.Status = engine.ACCEPTED_I
			inv.NextStatus = nil
			noDbFromTest.SaveInv(inv)
			return ctx, nil
		},
		"wait>accepted",
	)
	s.Add(
		ffsm.State(engine.WAIT_I),
		ffsm.State(engine.REJECTED_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			invID, ok := payload.(int64)
			if !ok {
				log.Println("Invoice bad Payload: ", payload)
				return
			}
			inv := noDbFromTest.GetInv(invID)
			if inv == nil {
				log.Println("Invoice not found id: ", invID)
				return
			}
			if inv.Strategy != InvRechargeStrategy.String() {
				return ctx, errors.New("Invoice strategy not InvRechargeStrategy.")
			}
			if inv.Status != engine.WAIT_I {
				return ctx, errors.New("Invoice status not wait.")
			}
			// Установить статус куда происходит переход
			ns := engine.REJECTED_I
			inv.NextStatus = &ns
			noDbFromTest.SaveInv(inv)
			// TODO load transactions from DB
			next := true
			for _, v := range noDbFromTest.ListTr(invID) {
				if !v.Status.Match(engine.REJECTED_TX) {
					next = false
					break
				}
			}
			if !next {
				return ctx, nil
			}
			// Установить статус после проделанных операций
			inv = noDbFromTest.GetInv(invID)
			inv.Status = engine.REJECTED_I
			inv.NextStatus = nil
			noDbFromTest.SaveInv(inv)
			return ctx, nil
		},
		"wait>rejected",
	)
	s.Add(
		ffsm.State(engine.WAIT_I),
		ffsm.State(engine.DRAFT_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			invID, ok := payload.(int64)
			if !ok {
				log.Println("Invoice bad Payload: ", payload)
				return
			}
			inv := noDbFromTest.GetInv(invID)
			if inv == nil {
				log.Println("Invoice not found id: ", invID)
				return
			}
			if inv.Strategy != InvRechargeStrategy.String() {
				return ctx, errors.New("Invoice strategy not InvRechargeStrategy.")
			}
			if inv.Status != engine.WAIT_I {
				return ctx, errors.New("Invoice status not wait.")
			}
			// Установить статус куда происходит переход
			ns := engine.DRAFT_I
			inv.NextStatus = &ns
			noDbFromTest.SaveInv(inv)
			// TODO load transactions from DB
			next := true
			for _, v := range noDbFromTest.ListTr(invID) {
				if !v.Status.Match(engine.DRAFT_TX) {
					next = false
					break
				}
			}
			if !next {
				return ctx, nil
			}
			// Установить статус после проделанных операций
			inv = noDbFromTest.GetInv(invID)
			inv.Status = engine.DRAFT_I
			inv.NextStatus = nil
			noDbFromTest.SaveInv(inv)
			return ctx, nil
		},
		"wait>draft",
	)
	s.Add(
		ffsm.State(engine.AUTH_I),
		ffsm.State(engine.DRAFT_I),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load invoice from DB
			invID, ok := payload.(int64)
			if !ok {
				log.Println("Invoice bad Payload: ", payload)
				return
			}
			inv := noDbFromTest.GetInv(invID)
			if inv == nil {
				log.Println("Invoice not found id: ", invID)
				return
			}
			if inv.Strategy != InvRechargeStrategy.String() {
				return ctx, errors.New("Invoice strategy not InvRechargeStrategy.")
			}
			if inv.Status != engine.AUTH_I {
				return ctx, errors.New("Invoice status not auth.")
			}
			// Установить статус куда происходит переход
			ns := engine.DRAFT_I
			inv.NextStatus = &ns
			noDbFromTest.SaveInv(inv)
			// TODO load transactions from DB
			next := true
			for _, v := range noDbFromTest.ListTr(invID) {
				if !v.Status.Match(engine.DRAFT_TX) {
					next = false
					break
				}
			}
			if !next {
				return ctx, nil
			}
			// Установить статус после проделанных операций
			inv = noDbFromTest.GetInv(invID)
			inv.Status = engine.DRAFT_I
			inv.NextStatus = nil
			noDbFromTest.SaveInv(inv)
			return ctx, nil
		},
		"auth>draft",
	)
	return &strategyOfInvoice{s: s}
}

func InitTransactionSberbankStrategy() *strategyOfTransaction {
	s := make(ffsm.Stack)
	s.Add(
		ffsm.State(engine.DRAFT_TX),
		ffsm.State(engine.AUTH_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			trID, ok := payload.(int64)
			if !ok {
				log.Println("Transaction bad Payload: ", payload)
				return
			}
			tr := noDbFromTest.GetTr(trID)
			if tr == nil {
				log.Println("Transaction not found id: ", trID)
				return
			}
			if tr.Strategy != TrSberbankStrategy.String() {
				return ctx, errors.New("Transaction strategy not TrSberbankStrategy.")
			}
			if tr.Status != engine.DRAFT_TX {
				return ctx, errors.New("Transaction status not draft.")
			}
			// Установить статус куда происходит переход
			ns := engine.AUTH_TX
			tr.NextStatus = &ns
			noDbFromTest.SaveTr(tr)
			// TODO добавить необходимые операции по транзакции.
			if simRequestToSberbank.requestToSberbank() {
				log.Println("Fail request to sberbank")
				return ctx, nil
			}
			// Установить статус после проделанных операций
			tr = noDbFromTest.GetTr(trID)
			tr.Status = engine.AUTH_TX
			tr.NextStatus = nil
			noDbFromTest.SaveTr(tr)
			return ctx, nil
		},
		"draft>auth",
	)
	s.Add(
		ffsm.State(engine.DRAFT_TX),
		ffsm.State(engine.REJECTED_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			trID, ok := payload.(int64)
			if !ok {
				log.Println("Transaction bad Payload: ", payload)
				return
			}
			tr := noDbFromTest.GetTr(trID)
			if tr == nil {
				log.Println("Transaction not found id: ", trID)
				return
			}
			if tr.Strategy != TrSberbankStrategy.String() {
				return ctx, errors.New("Transaction strategy not TrSberbankStrategy.")
			}
			if tr.Status != engine.DRAFT_TX {
				return ctx, errors.New("Transaction status not draft.")
			}
			// Установить статус куда происходит переход
			ns := engine.REJECTED_TX
			tr.NextStatus = &ns
			noDbFromTest.SaveTr(tr)
			// TODO добавить необходимые операции по транзакции.
			if simRequestToSberbank.requestToSberbank() {
				log.Println("Fail request to sberbank")
				return ctx, nil
			}
			// Установить статус после проделанных операций
			tr = noDbFromTest.GetTr(trID)
			tr.Status = engine.REJECTED_TX
			tr.NextStatus = nil
			noDbFromTest.SaveTr(tr)
			return ctx, nil
		},
		"draft>rejected",
	)
	s.Add(
		ffsm.State(engine.AUTH_TX),
		ffsm.State(engine.ACCEPTED_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			trID, ok := payload.(int64)
			if !ok {
				log.Println("Transaction bad Payload: ", payload)
				return
			}
			tr := noDbFromTest.GetTr(trID)
			if tr == nil {
				log.Println("Transaction not found id: ", trID)
				return
			}
			if tr.Strategy != TrSberbankStrategy.String() {
				return ctx, errors.New("Transaction strategy not TrSberbankStrategy.")
			}
			if tr.Status != engine.AUTH_TX {
				return ctx, errors.New("Transaction status not auth.")
			}
			// Установить статус куда происходит переход
			ns := engine.ACCEPTED_TX
			tr.NextStatus = &ns
			noDbFromTest.SaveTr(tr)
			// TODO добавить необходимые операции по транзакции.
			if simRequestToSberbank.requestToSberbank() {
				log.Println("Fail request to sberbank")
				return ctx, nil
			}
			// Установить статус после проделанных операций
			tr = noDbFromTest.GetTr(trID)
			tr.Status = engine.ACCEPTED_TX
			tr.NextStatus = nil
			noDbFromTest.SaveTr(tr)
			return ctx, nil
		},
		"auth>accepted",
	)
	s.Add(
		ffsm.State(engine.AUTH_TX),
		ffsm.State(engine.REJECTED_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			trID, ok := payload.(int64)
			if !ok {
				log.Println("Transaction bad Payload: ", payload)
				return
			}
			tr := noDbFromTest.GetTr(trID)
			if tr == nil {
				log.Println("Transaction not found id: ", trID)
				return
			}
			if tr.Strategy != TrSberbankStrategy.String() {
				return ctx, errors.New("Transaction strategy not TrSberbankStrategy.")
			}
			if tr.Status != engine.AUTH_TX {
				return ctx, errors.New("Transaction status not auth.")
			}
			// Установить статус куда происходит переход
			ns := engine.REJECTED_TX
			tr.NextStatus = &ns
			noDbFromTest.SaveTr(tr)
			// TODO добавить необходимые операции по транзакции.
			if simRequestToSberbank.requestToSberbank() {
				log.Println("Fail request to sberbank")
				return ctx, nil
			}
			// Установить статус после проделанных операций
			tr = noDbFromTest.GetTr(trID)
			tr.Status = engine.REJECTED_TX
			tr.NextStatus = nil
			noDbFromTest.SaveTr(tr)
			return ctx, nil
		},
		"auth>rejected",
	)
	s.Add(
		ffsm.State(engine.DRAFT_TX),
		ffsm.State(engine.FAILED_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			trID, ok := payload.(int64)
			if !ok {
				log.Println("Transaction bad Payload: ", payload)
				return
			}
			tr := noDbFromTest.GetTr(trID)
			if tr == nil {
				log.Println("Transaction not found id: ", trID)
				return
			}
			if tr.Strategy != TrSberbankStrategy.String() {
				return ctx, errors.New("Transaction strategy not TrSberbankStrategy.")
			}
			if tr.Status != engine.DRAFT_TX {
				return ctx, errors.New("Transaction status not draft.")
			}
			// Установить статус куда происходит переход
			ns := engine.FAILED_TX
			tr.NextStatus = &ns
			noDbFromTest.SaveTr(tr)
			// TODO добавить необходимые операции по транзакции.
			if simRequestToSberbank.requestToSberbank() {
				log.Println("Fail request to sberbank")
				return ctx, nil
			}
			// Установить статус после проделанных операций
			tr = noDbFromTest.GetTr(trID)
			tr.Status = engine.FAILED_TX
			tr.NextStatus = nil
			noDbFromTest.SaveTr(tr)
			return ctx, nil
		},
		"draft>failed",
	)
	s.Add(
		ffsm.State(engine.AUTH_TX),
		ffsm.State(engine.FAILED_TX),
		func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
			// TODO load transaction from BD
			trID, ok := payload.(int64)
			if !ok {
				log.Println("Transaction bad Payload: ", payload)
				return
			}
			tr := noDbFromTest.GetTr(trID)
			if tr == nil {
				log.Println("Transaction not found id: ", trID)
				return
			}
			if tr.Strategy != TrSberbankStrategy.String() {
				return ctx, errors.New("Transaction strategy not TrSberbankStrategy.")
			}
			if tr.Status != engine.AUTH_TX {
				return ctx, errors.New("Transaction status not draft.")
			}
			// Установить статус куда происходит переход
			ns := engine.FAILED_TX
			tr.NextStatus = &ns
			noDbFromTest.SaveTr(tr)
			// TODO добавить необходимые операции по транзакции.
			if simRequestToSberbank.requestToSberbank() {
				log.Println("Fail request to sberbank")
				return ctx, nil
			}
			// Установить статус после проделанных операций
			tr = noDbFromTest.GetTr(trID)
			tr.Status = engine.FAILED_TX
			tr.NextStatus = nil
			noDbFromTest.SaveTr(tr)
			return ctx, nil
		},
		"auth>failed",
	)
	return &strategyOfTransaction{s: s}
}
