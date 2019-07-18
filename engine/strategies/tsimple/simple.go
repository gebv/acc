package tsimple

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/engine/strategies"
	"github.com/gebv/acca/ffsm"
	"github.com/pkg/errors"
)

const simpleStrategy strategies.StrategyName = "transaction_simple_strategy"

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
}

func (s *Strategy) Name() strategies.StrategyName {
	return simpleStrategy
}

func (s *Strategy) Dispatch(ctx context.Context, state ffsm.State, payload ffsm.Payload) error {
	trID, ok := payload.(int64)
	if !ok {
		return errors.New("bad_payload")
	}
	tx := strategies.GetTXContext(ctx)
	if tx == nil {
		return errors.New("Not reform tx.")
	}
	tr := engine.Transaction{TransactionID: trID}
	if err := tx.Reload(&tr); err != nil {
		return errors.Wrap(err, "Failed reload transaction by ID.")
	}
	st := ffsm.State(tr.Status)
	fsm := ffsm.MachineFrom(s.s, &st)
	err := fsm.Dispatch(ctx, state, payload)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

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
				if tr.Strategy != simpleStrategy.String() {
					return ctx, errors.New("Transaction strategy not simpleStrategy.")
				}
				if tr.Status != engine.DRAFT_TX {
					return ctx, errors.New("Transaction status not draft.")
				}
				// Установить статус куда происходит переход
				tr.Status = engine.WAUTH_TX
				ns := engine.AUTH_TX
				tr.NextStatus = &ns
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				if err := engine.ProcessingOperations(tx, &engine.ProcessorCommand{
					TrID:          tr.TransactionID,
					CurrentStatus: engine.DRAFT_TX,
					NextStatus:    engine.AUTH_TX,
					UpdatedAt:     time.Now(),
				}); err != nil {
					return ctx, errors.Wrap(err, "failed operation process")
				}
				tr.Status = engine.AUTH_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				// TODO отправить сообщение в очередь о смене статуса у инвойса.
				log.Println("InvoiceID: ", tr.InvoiceID, ", TransactionID: ", tr.TransactionID)
				return ctx, nil
			},
			"draft>auth",
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
				if tr.Strategy != simpleStrategy.String() {
					return ctx, errors.New("Transaction strategy not simpleStrategy.")
				}
				if tr.Status != engine.AUTH_TX {
					return ctx, errors.New("Transaction status not auth.")
				}
				// Установить статус куда происходит переход
				tr.Status = engine.WHOLD_TX
				ns := engine.HOLD_TX
				tr.NextStatus = &ns
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				if err := engine.ProcessingOperations(tx, &engine.ProcessorCommand{
					TrID:          tr.TransactionID,
					CurrentStatus: engine.AUTH_TX,
					NextStatus:    engine.HOLD_TX,
					UpdatedAt:     time.Now(),
				}); err != nil {
					return ctx, errors.Wrap(err, "failed operation process")
				}
				tr.Status = engine.HOLD_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				// TODO отправить сообщение в очередь о смене статуса у инвойса.
				log.Println("InvoiceID: ", tr.InvoiceID, ", TransactionID: ", tr.TransactionID)
				return ctx, nil
			},
			"auth>hold",
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
				if tr.Strategy != simpleStrategy.String() {
					return ctx, errors.New("Transaction strategy not simpleStrategy.")
				}
				if tr.Status != engine.HOLD_TX {
					return ctx, errors.New("Transaction status not hold.")
				}
				// Установить статус куда происходит переход
				tr.Status = engine.WACCEPTED_TX
				ns := engine.ACCEPTED_TX
				tr.NextStatus = &ns
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				if err := engine.ProcessingOperations(tx, &engine.ProcessorCommand{
					TrID:          tr.TransactionID,
					CurrentStatus: engine.HOLD_TX,
					NextStatus:    engine.ACCEPTED_TX,
					UpdatedAt:     time.Now(),
				}); err != nil {
					return ctx, errors.Wrap(err, "failed operation process")
				}
				tr.Status = engine.ACCEPTED_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				// TODO отправить сообщение в очередь о смене статуса у инвойса.
				log.Println("InvoiceID: ", tr.InvoiceID, ", TransactionID: ", tr.TransactionID)
				return ctx, nil
			},
			"hold>accepted",
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
				if tr.Strategy != simpleStrategy.String() {
					return ctx, errors.New("Transaction strategy not simpleStrategy.")
				}
				if tr.Status != engine.AUTH_TX {
					return ctx, errors.New("Transaction status not auth.")
				}
				// Установить статус куда происходит переход
				tr.Status = engine.WACCEPTED_TX
				ns := engine.ACCEPTED_TX
				tr.NextStatus = &ns
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				if err := engine.ProcessingOperations(tx, &engine.ProcessorCommand{
					TrID:          tr.TransactionID,
					CurrentStatus: engine.AUTH_TX,
					NextStatus:    engine.ACCEPTED_TX,
					UpdatedAt:     time.Now(),
				}); err != nil {
					return ctx, errors.Wrap(err, "failed operation process")
				}
				tr.Status = engine.ACCEPTED_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				// TODO отправить сообщение в очередь о смене статуса у инвойса.
				log.Println("InvoiceID: ", tr.InvoiceID, ", TransactionID: ", tr.TransactionID)
				return ctx, nil
			},
			"auth>accepted",
		)
		s.s.Add(
			ffsm.State(engine.AUTH_TX),
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
				if tr.Strategy != simpleStrategy.String() {
					return ctx, errors.New("Transaction strategy not simpleStrategy.")
				}
				if tr.Status != engine.AUTH_TX {
					return ctx, errors.New("Transaction status not auth.")
				}
				// Установить статус куда происходит переход
				tr.Status = engine.WREJECTED_TX
				ns := engine.REJECTED_TX
				tr.NextStatus = &ns
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				if err := engine.ProcessingOperations(tx, &engine.ProcessorCommand{
					TrID:          tr.TransactionID,
					CurrentStatus: engine.AUTH_TX,
					NextStatus:    engine.REJECTED_TX,
					UpdatedAt:     time.Now(),
				}); err != nil {
					return ctx, errors.Wrap(err, "failed operation process")
				}
				tr.Status = engine.REJECTED_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				// TODO отправить сообщение в очередь о смене статуса у инвойса.
				log.Println("InvoiceID: ", tr.InvoiceID, ", TransactionID: ", tr.TransactionID)
				return ctx, nil
			},
			"auth>rejected",
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
				if tr.Strategy != simpleStrategy.String() {
					return ctx, errors.New("Transaction strategy not simpleStrategy.")
				}
				if tr.Status != engine.HOLD_TX {
					return ctx, errors.New("Transaction status not hold.")
				}
				// Установить статус куда происходит переход
				tr.Status = engine.WREJECTED_TX
				ns := engine.REJECTED_TX
				tr.NextStatus = &ns
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				if err := engine.ProcessingOperations(tx, &engine.ProcessorCommand{
					TrID:          tr.TransactionID,
					CurrentStatus: engine.HOLD_TX,
					NextStatus:    engine.REJECTED_TX,
					UpdatedAt:     time.Now(),
				}); err != nil {
					return ctx, errors.Wrap(err, "failed operation process")
				}
				tr.Status = engine.REJECTED_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				// TODO отправить сообщение в очередь о смене статуса у инвойса.
				log.Println("InvoiceID: ", tr.InvoiceID, ", TransactionID: ", tr.TransactionID)
				return ctx, nil
			},
			"hold>rejected",
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
				if tr.Strategy != simpleStrategy.String() {
					return ctx, errors.New("Transaction strategy not simpleStrategy.")
				}
				if tr.Status != engine.DRAFT_TX {
					return ctx, errors.New("Transaction status not draft.")
				}
				// Установить статус куда происходит переход
				tr.Status = engine.WREJECTED_TX
				ns := engine.REJECTED_TX
				tr.NextStatus = &ns
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				if err := engine.ProcessingOperations(tx, &engine.ProcessorCommand{
					TrID:          tr.TransactionID,
					CurrentStatus: engine.DRAFT_TX,
					NextStatus:    engine.REJECTED_TX,
					UpdatedAt:     time.Now(),
				}); err != nil {
					return ctx, errors.Wrap(err, "failed operation process")
				}
				tr.Status = engine.REJECTED_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				// TODO отправить сообщение в очередь о смене статуса у инвойса.
				log.Println("InvoiceID: ", tr.InvoiceID, ", TransactionID: ", tr.TransactionID)
				return ctx, nil
			},
			"draft>rejected",
		)
	})
}
