package tsberbank

import (
	"context"
	"log"
	"sync"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/engine/strategies"
	"github.com/gebv/acca/ffsm"
	"github.com/pkg/errors"
)

const nameStrategy strategies.TrStrategyName = "transaction_sberbank_strategy"

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

func (s *Strategy) Name() strategies.TrStrategyName {
	return nameStrategy
}

func (s *Strategy) Provider() engine.Provider {
	return engine.SBERBANK
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
				// TODO load transaction from BD
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
					return ctx, errors.New("Transaction strategy not TrSberbankStrategy.")
				}
				if tr.Status != engine.DRAFT_TX {
					return ctx, errors.New("Transaction status not draft.")
				}
				// Установить статус куда происходит переход
				ns := engine.AUTH_TX
				tr.NextStatus = &ns
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				// TODO добавить необходимые операции по транзакции.
				// Установить статус после проделанных операций
				tr.Status = engine.AUTH_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				return ctx, nil
			},
			"draft>auth",
		)
		s.s.Add(
			ffsm.State(engine.DRAFT_TX),
			ffsm.State(engine.REJECTED_TX),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				// TODO load transaction from BD
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
					return ctx, errors.New("Transaction strategy not TrSberbankStrategy.")
				}
				if tr.Status != engine.DRAFT_TX {
					return ctx, errors.New("Transaction status not draft.")
				}
				// Установить статус куда происходит переход
				ns := engine.REJECTED_TX
				tr.NextStatus = &ns
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				// TODO добавить необходимые операции по транзакции.
				// Установить статус после проделанных операций
				tr.Status = engine.REJECTED_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				return ctx, nil
			},
			"draft>rejected",
		)
		s.s.Add(
			ffsm.State(engine.AUTH_TX),
			ffsm.State(engine.ACCEPTED_TX),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				// TODO load transaction from BD
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
					return ctx, errors.New("Transaction strategy not TrSberbankStrategy.")
				}
				if tr.Status != engine.AUTH_TX {
					return ctx, errors.New("Transaction status not auth.")
				}
				// Установить статус куда происходит переход
				ns := engine.ACCEPTED_TX
				tr.NextStatus = &ns
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				// TODO добавить необходимые операции по транзакции.
				// Установить статус после проделанных операций
				tr.Status = engine.ACCEPTED_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				return ctx, nil
			},
			"auth>accepted",
		)
		s.s.Add(
			ffsm.State(engine.AUTH_TX),
			ffsm.State(engine.REJECTED_TX),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				// TODO load transaction from BD
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
					return ctx, errors.New("Transaction strategy not TrSberbankStrategy.")
				}
				if tr.Status != engine.AUTH_TX {
					return ctx, errors.New("Transaction status not auth.")
				}
				// Установить статус куда происходит переход
				ns := engine.REJECTED_TX
				tr.NextStatus = &ns
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				// TODO добавить необходимые операции по транзакции.
				// Установить статус после проделанных операций
				tr.Status = engine.REJECTED_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				return ctx, nil
			},
			"auth>rejected",
		)
		s.s.Add(
			ffsm.State(engine.DRAFT_TX),
			ffsm.State(engine.FAILED_TX),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				// TODO load transaction from BD
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
					return ctx, errors.New("Transaction strategy not TrSberbankStrategy.")
				}
				if tr.Status != engine.DRAFT_TX {
					return ctx, errors.New("Transaction status not draft.")
				}
				// Установить статус куда происходит переход
				ns := engine.FAILED_TX
				tr.NextStatus = &ns
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				// TODO добавить необходимые операции по транзакции.
				// Установить статус после проделанных операций
				tr.Status = engine.FAILED_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				return ctx, nil
			},
			"draft>failed",
		)
		s.s.Add(
			ffsm.State(engine.AUTH_TX),
			ffsm.State(engine.FAILED_TX),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				// TODO load transaction from BD
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
					return ctx, errors.New("Transaction strategy not TrSberbankStrategy.")
				}
				if tr.Status != engine.AUTH_TX {
					return ctx, errors.New("Transaction status not draft.")
				}
				// Установить статус куда происходит переход
				ns := engine.FAILED_TX
				tr.NextStatus = &ns
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				// TODO добавить необходимые операции по транзакции.
				// Установить статус после проделанных операций
				tr.Status = engine.FAILED_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				return ctx, nil
			},
			"auth>failed",
		)
	})
}
