package moedelo

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/engine/strategies"
	"github.com/gebv/acca/ffsm"
	"github.com/gebv/acca/provider/moedelo"
	"github.com/pkg/errors"
)

const nameStrategy strategies.TrStrategyName = "transaction_moedelo_strategy"

func init() {
	s := &Strategy{
		s: make(ffsm.Stack),
	}
	s.load()
	strategies.RegTransactionStrategy(s)
}

type Strategy struct {
	s        ffsm.Stack
	syncOnce sync.Once
	config   moedelo.Config
}

func (s *Strategy) Name() strategies.TrStrategyName {
	return nameStrategy
}

func (s *Strategy) Provider() engine.Provider {
	return engine.MOEDELO
}

func (s *Strategy) MetaValidation(meta *[]byte) error {
	if meta == nil {
		return nil
	}
	// TODO добавить проверку структуры в meta для стратегии
	//  и проверку полученных полей.
	return nil
}

func (s *Strategy) Dispatch(ctx context.Context, state ffsm.State, payload ffsm.Payload) error {
	txID, ok := payload.(int64)
	if !ok {
		return errors.New("bad_payload")
	}
	tx := strategies.GetTXContext(ctx)
	if tx == nil {
		return errors.New("Not reform tx.")
	}
	tr := engine.Transaction{TransactionID: txID}
	if err := tx.Reload(&tr); err != nil {
		return errors.Wrap(err, "Failed reload transaction by ID.")
	}
	st := ffsm.State(tr.Status)
	fsm := ffsm.MachineFrom(s.s, &st)
	err := fsm.Dispatch(ctx, state, payload)
	if err != nil {
		return err
	}
	return nil
}

var _ strategies.TrStrategy = (*Strategy)(nil)

func (s *Strategy) load() {
	s.syncOnce.Do(func() {
		s.s.Add(
			ffsm.State(engine.DRAFT_TX),
			ffsm.State(engine.AUTH_TX),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				trID, ok := payload.(int64)
				if !ok {
					log.Println("Transaction bad Payload: ", payload)
					return
				}
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				tr := engine.Transaction{TransactionID: trID}
				if err := tx.Reload(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed reload transaction by ID.")
				}
				if tr.Strategy != nameStrategy.String() {
					return ctx, errors.New("Transaction strategy not TrmoedeloStrategy.")
				}
				if tr.Status != engine.DRAFT_TX {
					return ctx, errors.New("Transaction status not draft.")
				}
				nc := strategies.GetNatsFromContext(ctx)
				if nc == nil {
					return ctx, errors.New("Not nats connection in context.")
				}
				// Установить статус куда происходит переход
				ns := engine.AUTH_TX
				tr.NextStatus = &ns
				tr.Status = engine.WAUTH_TX
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				err := nc.Publish(moedelo.SUBJECT, &moedelo.MessageToMoedelo{
					Command:       moedelo.CreateBill,
					TransactionID: tr.TransactionID,
					Strategy:      tr.Strategy,
					Status:        engine.AUTH_TX,
				})
				if err != nil {
					return ctx, errors.Wrap(err, "Failed publish to nats.")
				}
				return ctx, nil
			},
			"draft>auth",
		)
		s.s.Add(
			ffsm.State(engine.WAUTH_TX),
			ffsm.State(engine.AUTH_TX),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				trID, ok := payload.(int64)
				if !ok {
					log.Println("Transaction bad Payload: ", payload)
					return
				}
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				tr := engine.Transaction{TransactionID: trID}
				if err := tx.Reload(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed reload transaction by ID.")
				}
				if tr.Strategy != nameStrategy.String() {
					return ctx, errors.New("Transaction strategy not TrmoedeloStrategy.")
				}
				if tr.Status != engine.WAUTH_TX {
					return ctx, errors.New("Transaction status not auth_wait.")
				}
				inv := engine.Invoice{InvoiceID: tr.InvoiceID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				nc := strategies.GetNatsFromContext(ctx)
				if nc == nil {
					return ctx, errors.New("Not nats connection in context.")
				}
				// Установить статус куда происходит переход
				tr.Status = engine.AUTH_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				return ctx, nil
			},
			"auth_wait>auth",
		)
		s.s.Add(
			ffsm.State(engine.AUTH_TX),
			ffsm.State(engine.HOLD_TX),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				trID, ok := payload.(int64)
				if !ok {
					log.Println("Transaction bad Payload: ", payload)
					return
				}
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				tr := engine.Transaction{TransactionID: trID}
				if err := tx.Reload(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed reload transaction by ID.")
				}
				if tr.Strategy != nameStrategy.String() {
					return ctx, errors.New("Transaction strategy not TrmoedeloStrategy.")
				}
				if tr.Status != engine.AUTH_TX {
					return ctx, errors.New("Transaction status not auth_wait.")
				}
				inv := engine.Invoice{InvoiceID: tr.InvoiceID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				nc := strategies.GetNatsFromContext(ctx)
				if nc == nil {
					return ctx, errors.New("Not nats connection in context.")
				}
				// Установить статус куда происходит переход
				tr.Status = engine.AUTH_TX
				ns := engine.AUTH_TX
				tr.NextStatus = &ns
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				err, isHold := engine.ProcessingOperations(tx, &engine.ProcessorCommand{
					TrID:          tr.TransactionID,
					CurrentStatus: engine.DRAFT_TX,
					NextStatus:    engine.AUTH_TX,
					UpdatedAt:     time.Now(),
				})
				if err != nil {
					return ctx, errors.Wrap(err, "failed operation process")
				}
				if !isHold {
					return ctx, errors.Wrap(err, "Support only hold operation.")
				}
				invStatus := engine.ACCEPTED_I
				tr.Status = engine.ACCEPTED_TX
				if isHold {
					tr.Status = engine.HOLD_TX
					invStatus = engine.WAIT_I
				}
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				err = nc.Publish(strategies.UPDATE_INVOICE_SUBJECT, &strategies.MessageUpdateInvoice{
					InvoiceID: inv.InvoiceID,
					Strategy:  inv.Strategy,
					Status:    invStatus,
				})
				if err != nil {
					return ctx, errors.Wrap(err, "Failed publish to nats.")
				}
				return ctx, nil
			},
			"auth>hold",
		)
		s.s.Add(
			ffsm.State(engine.DRAFT_TX),
			ffsm.State(engine.REJECTED_TX),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				trID, ok := payload.(int64)
				if !ok {
					log.Println("Transaction bad Payload: ", payload)
					return
				}
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				tr := engine.Transaction{TransactionID: trID}
				if err := tx.Reload(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed reload transaction by ID.")
				}
				if tr.Strategy != nameStrategy.String() {
					return ctx, errors.New("Transaction strategy not TrmoedeloStrategy.")
				}
				if tr.Status != engine.DRAFT_TX {
					return ctx, errors.New("Transaction status not draft.")
				}
				inv := engine.Invoice{InvoiceID: tr.InvoiceID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				nc := strategies.GetNatsFromContext(ctx)
				if nc == nil {
					return ctx, errors.New("Not nats connection in context.")
				}
				// Установить статус куда происходит переход
				tr.Status = engine.REJECTED_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				err := nc.Publish(strategies.UPDATE_INVOICE_SUBJECT, &strategies.MessageUpdateInvoice{
					InvoiceID: inv.InvoiceID,
					Strategy:  inv.Strategy,
					Status:    engine.REJECTED_I,
				})
				if err != nil {
					return ctx, errors.Wrap(err, "Failed publish to nats.")
				}
				return ctx, nil
			},
			"draft>rejected",
		)
		s.s.Add(
			ffsm.State(engine.HOLD_TX),
			ffsm.State(engine.REJECTED_TX),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				trID, ok := payload.(int64)
				if !ok {
					log.Println("Transaction bad Payload: ", payload)
					return
				}
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				tr := engine.Transaction{TransactionID: trID}
				if err := tx.Reload(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed reload transaction by ID.")
				}
				if tr.Strategy != nameStrategy.String() {
					return ctx, errors.New("Transaction strategy not TrmoedeloStrategy.")
				}
				if tr.Status != engine.HOLD_TX {
					return ctx, errors.New("Transaction status not draft.")
				}
				nc := strategies.GetNatsFromContext(ctx)
				if nc == nil {
					return ctx, errors.New("Not nats connection in context.")
				}
				// Установить статус куда происходит переход
				ns := engine.REJECTED_TX
				tr.NextStatus = &ns
				tr.Status = engine.WREJECTED_TX
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				// Нет способа отменить и вернуть деньги по безналу из моего дела.
				err := nc.Publish(moedelo.SUBJECT, &moedelo.MessageToMoedelo{
					Command:       moedelo.ReverseForHold,
					TransactionID: tr.TransactionID,
					Strategy:      tr.Strategy,
					Status:        engine.REJECTED_TX,
				})
				if err != nil {
					return ctx, errors.Wrap(err, "Failed publish to nats.")
				}
				return ctx, nil
			},
			"hold>rejected",
		)
		s.s.Add(
			ffsm.State(engine.WREJECTED_TX),
			ffsm.State(engine.REJECTED_TX),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				trID, ok := payload.(int64)
				if !ok {
					log.Println("Transaction bad Payload: ", payload)
					return
				}
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				tr := engine.Transaction{TransactionID: trID}
				if err := tx.Reload(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed reload transaction by ID.")
				}
				if tr.Strategy != nameStrategy.String() {
					return ctx, errors.New("Transaction strategy not TrmoedeloStrategy.")
				}
				if tr.Status != engine.WREJECTED_TX {
					return ctx, errors.New("Transaction status not rejected_wait.")
				}
				inv := engine.Invoice{InvoiceID: tr.InvoiceID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				nc := strategies.GetNatsFromContext(ctx)
				if nc == nil {
					return ctx, errors.New("Not nats connection in context.")
				}
				// Установить статус куда происходит переход
				ns := engine.REJECTED_TX
				tr.NextStatus = &ns
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				err, _ := engine.ProcessingOperations(tx, &engine.ProcessorCommand{
					TrID:          tr.TransactionID,
					CurrentStatus: engine.HOLD_TX,
					NextStatus:    engine.REJECTED_TX,
					UpdatedAt:     time.Now(),
				})
				if err != nil {
					return ctx, errors.Wrap(err, "failed operation process")
				}
				tr.Status = engine.REJECTED_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				err = nc.Publish(strategies.UPDATE_INVOICE_SUBJECT, &strategies.MessageUpdateInvoice{
					InvoiceID: inv.InvoiceID,
					Strategy:  inv.Strategy,
					Status:    engine.REJECTED_I,
				})
				if err != nil {
					return ctx, errors.Wrap(err, "Failed publish to nats.")
				}
				return ctx, nil
			},
			"rejected_wait>rejected",
		)
		s.s.Add(
			ffsm.State(engine.AUTH_TX),
			ffsm.State(engine.ACCEPTED_TX),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				trID, ok := payload.(int64)
				if !ok {
					log.Println("Transaction bad Payload: ", payload)
					return
				}
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				tr := engine.Transaction{TransactionID: trID}
				if err := tx.Reload(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed reload transaction by ID.")
				}
				if tr.Strategy != nameStrategy.String() {
					return ctx, errors.New("Transaction strategy not TrmoedeloStrategy.")
				}
				if tr.Status != engine.AUTH_TX {
					return ctx, errors.New("Transaction status not auth.")
				}
				inv := engine.Invoice{InvoiceID: tr.InvoiceID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				nc := strategies.GetNatsFromContext(ctx)
				if nc == nil {
					return ctx, errors.New("Not nats connection in context.")
				}
				// Установить статус куда происходит переход
				tr.Status = engine.ACCEPTED_TX
				ns := engine.ACCEPTED_TX
				tr.NextStatus = &ns
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				err, isHold := engine.ProcessingOperations(tx, &engine.ProcessorCommand{
					TrID:          tr.TransactionID,
					CurrentStatus: engine.DRAFT_TX,
					NextStatus:    engine.AUTH_TX,
					UpdatedAt:     time.Now(),
				})
				if err != nil {
					return ctx, errors.Wrap(err, "failed operation process")
				}
				invStatus := engine.ACCEPTED_I
				tr.Status = engine.ACCEPTED_TX
				if isHold {
					tr.Status = engine.HOLD_TX
					invStatus = engine.WAIT_I
				}
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				err = nc.Publish(strategies.UPDATE_INVOICE_SUBJECT, &strategies.MessageUpdateInvoice{
					InvoiceID: inv.InvoiceID,
					Strategy:  inv.Strategy,
					Status:    invStatus,
				})
				if err != nil {
					return ctx, errors.Wrap(err, "Failed publish to nats.")
				}
				return ctx, nil
			},
			"auth>accepted",
		)
		s.s.Add(
			ffsm.State(engine.HOLD_TX),
			ffsm.State(engine.ACCEPTED_TX),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				trID, ok := payload.(int64)
				if !ok {
					log.Println("Transaction bad Payload: ", payload)
					return
				}
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				tr := engine.Transaction{TransactionID: trID}
				if err := tx.Reload(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed reload transaction by ID.")
				}
				if tr.Strategy != nameStrategy.String() {
					return ctx, errors.New("Transaction strategy not TrmoedeloStrategy.")
				}
				if tr.Status != engine.HOLD_TX {
					return ctx, errors.New("Transaction status not hold.")
				}
				inv := engine.Invoice{InvoiceID: tr.InvoiceID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				nc := strategies.GetNatsFromContext(ctx)
				if nc == nil {
					return ctx, errors.New("Not nats connection in context.")
				}
				// Установить статус куда происходит переход
				ns := engine.ACCEPTED_TX
				tr.NextStatus = &ns
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				err, _ := engine.ProcessingOperations(tx, &engine.ProcessorCommand{
					TrID:          tr.TransactionID,
					CurrentStatus: engine.HOLD_TX,
					NextStatus:    engine.ACCEPTED_TX,
					UpdatedAt:     time.Now(),
				})
				if err != nil {
					return ctx, errors.Wrap(err, "failed operation process")
				}
				tr.Status = engine.ACCEPTED_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				err = nc.Publish(strategies.UPDATE_INVOICE_SUBJECT, &strategies.MessageUpdateInvoice{
					InvoiceID: inv.InvoiceID,
					Strategy:  inv.Strategy,
					Status:    engine.ACCEPTED_I,
				})
				if err != nil {
					return ctx, errors.Wrap(err, "Failed publish to nats.")
				}
				return ctx, nil
			},
			"hold>accepted",
		)
	})
}
