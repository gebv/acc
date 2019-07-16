package tsberbank

import (
	"context"
	"log"
	"sync"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/engine/strategies/for_testing"
	"github.com/gebv/acca/engine/strategies/store"
	"github.com/gebv/acca/ffsm"
	"github.com/pkg/errors"
)

const TrSberbankStrategy store.StrategyName = "transaction_sberbank_strategy"

func init() {
	s := &Strategy{
		s: make(ffsm.Stack),
	}
	s.load()
	store.Reg(TrSberbankStrategy, s)
}

type Strategy struct {
	s        ffsm.Stack
	syncOnce sync.Once
}

func (s *Strategy) Dispatch(ctx context.Context, state ffsm.State, payload ffsm.Payload) error {
	txID, ok := payload.(int64)
	if !ok {
		return errors.New("bad_payload")
	}
	tr := for_testing.NoDbFromTest.GetTr(txID)
	st := ffsm.State(tr.Status)
	fsm := ffsm.MachineFrom(s.s, &st)
	err := fsm.Dispatch(ctx, state, payload)
	if err != nil {
		return err
	}
	return nil
}

var _ store.Strategy = (*Strategy)(nil)

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
				tr := for_testing.NoDbFromTest.GetTr(trID)
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
				for_testing.NoDbFromTest.SaveTr(tr)
				// TODO добавить необходимые операции по транзакции.
				if for_testing.SimRequestToSberbank.RequestToSberbank() {
					log.Println("Fail request to sberbank")
					return ctx, nil
				}
				// Установить статус после проделанных операций
				tr = for_testing.NoDbFromTest.GetTr(trID)
				tr.Status = engine.AUTH_TX
				tr.NextStatus = nil
				for_testing.NoDbFromTest.SaveTr(tr)
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
				tr := for_testing.NoDbFromTest.GetTr(trID)
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
				for_testing.NoDbFromTest.SaveTr(tr)
				// TODO добавить необходимые операции по транзакции.
				if for_testing.SimRequestToSberbank.RequestToSberbank() {
					log.Println("Fail request to sberbank")
					return ctx, nil
				}
				// Установить статус после проделанных операций
				tr = for_testing.NoDbFromTest.GetTr(trID)
				tr.Status = engine.REJECTED_TX
				tr.NextStatus = nil
				for_testing.NoDbFromTest.SaveTr(tr)
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
				tr := for_testing.NoDbFromTest.GetTr(trID)
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
				for_testing.NoDbFromTest.SaveTr(tr)
				// TODO добавить необходимые операции по транзакции.
				if for_testing.SimRequestToSberbank.RequestToSberbank() {
					log.Println("Fail request to sberbank")
					return ctx, nil
				}
				// Установить статус после проделанных операций
				tr = for_testing.NoDbFromTest.GetTr(trID)
				tr.Status = engine.ACCEPTED_TX
				tr.NextStatus = nil
				for_testing.NoDbFromTest.SaveTr(tr)
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
				tr := for_testing.NoDbFromTest.GetTr(trID)
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
				for_testing.NoDbFromTest.SaveTr(tr)
				// TODO добавить необходимые операции по транзакции.
				if for_testing.SimRequestToSberbank.RequestToSberbank() {
					log.Println("Fail request to sberbank")
					return ctx, nil
				}
				// Установить статус после проделанных операций
				tr = for_testing.NoDbFromTest.GetTr(trID)
				tr.Status = engine.REJECTED_TX
				tr.NextStatus = nil
				for_testing.NoDbFromTest.SaveTr(tr)
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
				tr := for_testing.NoDbFromTest.GetTr(trID)
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
				for_testing.NoDbFromTest.SaveTr(tr)
				// TODO добавить необходимые операции по транзакции.
				if for_testing.SimRequestToSberbank.RequestToSberbank() {
					log.Println("Fail request to sberbank")
					return ctx, nil
				}
				// Установить статус после проделанных операций
				tr = for_testing.NoDbFromTest.GetTr(trID)
				tr.Status = engine.FAILED_TX
				tr.NextStatus = nil
				for_testing.NoDbFromTest.SaveTr(tr)
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
				tr := for_testing.NoDbFromTest.GetTr(trID)
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
				for_testing.NoDbFromTest.SaveTr(tr)
				// TODO добавить необходимые операции по транзакции.
				if for_testing.SimRequestToSberbank.RequestToSberbank() {
					log.Println("Fail request to sberbank")
					return ctx, nil
				}
				// Установить статус после проделанных операций
				tr = for_testing.NoDbFromTest.GetTr(trID)
				tr.Status = engine.FAILED_TX
				tr.NextStatus = nil
				for_testing.NoDbFromTest.SaveTr(tr)
				return ctx, nil
			},
			"auth>failed",
		)
	})
}
