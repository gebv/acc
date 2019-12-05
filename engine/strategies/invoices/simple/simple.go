package simple

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"cloud.google.com/go/pubsub"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/engine/strategies"
	"github.com/gebv/acca/ffsm"
)

const nameStrategy strategies.InvStrategyName = "invoice_simple_strategy"

func init() {
	s := &Strategy{
		s: make(ffsm.Stack),
	}
	s.load()
	strategies.RegInvoiceStrategy(s)
}

type Strategy struct {
	s        ffsm.Stack
	syncOnce sync.Once
}

func (s *Strategy) Name() strategies.InvStrategyName {
	return nameStrategy
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
	ctx, span := trace.StartSpan(ctx, "Dispatch."+s.Name().String())
	defer span.End()
	invID, ok := payload.(int64)
	if !ok {
		return errors.New("bad_payload")
	}
	span.AddAttributes(
		trace.Int64Attribute("invoice_id", invID),
		trace.StringAttribute("state", state.String()),
	)
	tx := strategies.GetTXContext(ctx)
	if tx == nil {
		return errors.New("Not reform tx.")
	}
	inv := engine.Invoice{InvoiceID: invID}
	if err := tx.Reload(&inv); err != nil {
		return errors.Wrap(err, "Failed reload invoice by ID.")
	}
	st := ffsm.State(inv.Status)
	fsm := ffsm.MachineFrom(s.s, &st)
	err := fsm.Dispatch(ctx, state, payload)
	if err != nil {
		return err
	}
	return nil
}

var _ strategies.InvStrategy = (*Strategy)(nil)

func (s *Strategy) load() {
	s.syncOnce.Do(func() {
		s.s.Add(
			ffsm.State(engine.DRAFT_I),
			ffsm.State(engine.AUTH_I),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				invID, ok := payload.(int64)
				if !ok {
					log.Println("Invoice bad Payload: ", payload)
					return
				}
				ctx, span := trace.StartSpan(ctx, "ChangeState."+s.Name().String())
				defer span.End()
				span.AddAttributes(
					trace.Int64Attribute("invoice_id", invID),
					trace.StringAttribute("src_status", string(engine.DRAFT_I)),
					trace.StringAttribute("dst_status", string(engine.AUTH_I)),
				)
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				inv := engine.Invoice{InvoiceID: invID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				if inv.Strategy != nameStrategy.String() {
					return ctx, errors.New("Invoice strategy not nameStrategy.")
				}
				if inv.Status != engine.DRAFT_I {
					return ctx, errors.New("Invoice status not draft.")
				}
				// Установить статус куда происходит переход
				ns := engine.AUTH_I
				inv.NextStatus = &ns
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				list, err := tx.SelectAllFrom(engine.TransactionTable, "WHERE invoice_id = $1", invID)
				if err != nil {
					return ctx, errors.Wrap(err, "Failed list transaction by invoice ID.")
				}
				pb := strategies.GetPubSubFromContext(ctx)
				if pb == nil {
					return ctx, errors.New("Not pubsub client in context.")
				}
				for _, v := range list {
					tr := v.(*engine.Transaction)
					if err != nil {
						log.Println("Failed publish to nats. InvoiceID: ", invID,
							", TransactionID: ", tr.TransactionID,
							", err: ", err)
						continue
					}
					b, err := json.Marshal(&strategies.MessageUpdateTransaction{
						ClientID:      tr.ClientID,
						TransactionID: tr.TransactionID,
						Strategy:      tr.Strategy,
						Status:        engine.AUTH_TX,
					})
					if err != nil {
						zap.L().Error("Failed json marshal for publish to pubsub.",
							zap.Int64("InvoiceID", invID),
							zap.Int64("TransactionID", tr.TransactionID),
							zap.Error(err))
						continue
					}
					if _, err := pb.Topic(strategies.UPDATE_TRANSACTION_SUBJECT).Publish(ctx, &pubsub.Message{
						Data: b,
					}).Get(ctx); err != nil {
						zap.L().Error("Failed publish to pubsub.",
							zap.Int64("InvoiceID", invID),
							zap.Int64("TransactionID", tr.TransactionID),
							zap.Error(err))
						continue
					}
				}
				inv.Status = engine.AUTH_I
				inv.NextStatus = nil
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				return ctx, nil
			},
			"draft>auth",
		)
		s.s.Add(
			ffsm.State(engine.DRAFT_I),
			ffsm.State(engine.ACCEPTED_I),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				invID, ok := payload.(int64)
				if !ok {
					log.Println("Invoice bad Payload: ", payload)
					return
				}
				ctx, span := trace.StartSpan(ctx, "ChangeState."+s.Name().String())
				defer span.End()
				span.AddAttributes(
					trace.Int64Attribute("invoice_id", invID),
					trace.StringAttribute("src_status", string(engine.DRAFT_I)),
					trace.StringAttribute("dst_status", string(engine.ACCEPTED_I)),
				)
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				inv := engine.Invoice{InvoiceID: invID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				if inv.Strategy != nameStrategy.String() {
					return ctx, errors.New("Invoice strategy not nameStrategy.")
				}
				if inv.Status != engine.DRAFT_I {
					return ctx, errors.New("Invoice status not auth.")
				}
				// Установить статус куда происходит переход
				ns := engine.ACCEPTED_I
				inv.NextStatus = &ns
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				next := true
				list, err := tx.SelectAllFrom(engine.TransactionTable, "WHERE invoice_id = $1", invID)
				if err != nil {
					return ctx, errors.Wrap(err, "Failed list transaction by invoice ID.")
				}
				for _, v := range list {
					tr := v.(*engine.Transaction)
					if !tr.Status.Match(engine.ACCEPTED_TX) {
						next = false
						break
					}
				}
				if !next {
					return ctx, nil
				}
				// Установить статус после проделанных операций
				inv.Status = engine.ACCEPTED_I
				inv.NextStatus = nil
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				return ctx, nil
			},
			"draft>accepted",
		)
		s.s.Add(
			ffsm.State(engine.DRAFT_I),
			ffsm.State(engine.WAIT_I),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				invID, ok := payload.(int64)
				if !ok {
					log.Println("Invoice bad Payload: ", payload)
					return
				}
				ctx, span := trace.StartSpan(ctx, "ChangeState."+s.Name().String())
				defer span.End()
				span.AddAttributes(
					trace.Int64Attribute("invoice_id", invID),
					trace.StringAttribute("src_status", string(engine.DRAFT_I)),
					trace.StringAttribute("dst_status", string(engine.WAIT_I)),
				)
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				inv := engine.Invoice{InvoiceID: invID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				if inv.Strategy != nameStrategy.String() {
					return ctx, errors.New("Invoice strategy not nameStrategy.")
				}
				if inv.Status != engine.DRAFT_I {
					return ctx, errors.New("Invoice status not auth.")
				}
				// Установить статус куда происходит переход
				ns := engine.WAIT_I
				inv.NextStatus = &ns
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				next := true
				list, err := tx.SelectAllFrom(engine.TransactionTable, "WHERE invoice_id = $1", invID)
				if err != nil {
					return ctx, errors.Wrap(err, "Failed list transaction by invoice ID.")
				}
				for _, v := range list {
					tr := v.(*engine.Transaction)
					if !tr.Status.Match(engine.HOLD_TX) {
						next = false
						break
					}
				}
				if !next {
					return ctx, nil
				}
				// Установить статус после проделанных операций
				inv.Status = engine.WAIT_I
				inv.NextStatus = nil
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				return ctx, nil
			},
			"draft>wait",
		)
		s.s.Add(
			ffsm.State(engine.DRAFT_I),
			ffsm.State(engine.REJECTED_I),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				invID, ok := payload.(int64)
				if !ok {
					log.Println("Invoice bad Payload: ", payload)
					return
				}
				ctx, span := trace.StartSpan(ctx, "ChangeState."+s.Name().String())
				defer span.End()
				span.AddAttributes(
					trace.Int64Attribute("invoice_id", invID),
					trace.StringAttribute("src_status", string(engine.DRAFT_I)),
					trace.StringAttribute("dst_status", string(engine.REJECTED_I)),
				)
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				inv := engine.Invoice{InvoiceID: invID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				if inv.Strategy != nameStrategy.String() {
					return ctx, errors.New("Invoice strategy not nameStrategy.")
				}
				if inv.Status != engine.DRAFT_I {
					return ctx, errors.New("Invoice status not draft.")
				}
				// Установить статус куда происходит переход
				ns := engine.REJECTED_I
				inv.NextStatus = &ns
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				next := true
				list, err := tx.SelectAllFrom(engine.TransactionTable, "WHERE invoice_id = $1", invID)
				if err != nil {
					return ctx, errors.Wrap(err, "Failed list transaction by invoice ID.")
				}
				for _, v := range list {
					tr := v.(*engine.Transaction)
					if !tr.Status.Match(engine.REJECTED_TX) {
						next = false
						break
					}
				}
				if !next {
					return ctx, nil
				}
				// Установить статус после проделанных операций
				inv.Status = engine.REJECTED_I
				inv.NextStatus = nil
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				return ctx, nil
			},
			"draft>rejected",
		)
		s.s.Add(
			ffsm.State(engine.AUTH_I),
			ffsm.State(engine.WAIT_I),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				invID, ok := payload.(int64)
				if !ok {
					log.Println("Invoice bad Payload: ", payload)
					return
				}
				ctx, span := trace.StartSpan(ctx, "ChangeState."+s.Name().String())
				defer span.End()
				span.AddAttributes(
					trace.Int64Attribute("invoice_id", invID),
					trace.StringAttribute("src_status", string(engine.AUTH_I)),
					trace.StringAttribute("dst_status", string(engine.WAIT_I)),
				)
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				inv := engine.Invoice{InvoiceID: invID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				if inv.Strategy != nameStrategy.String() {
					return ctx, errors.New("Invoice strategy not nameStrategy.")
				}
				if inv.Status != engine.AUTH_I {
					return ctx, errors.New("Invoice status not auth.")
				}
				// Установить статус куда происходит переход
				ns := engine.WAIT_I
				inv.NextStatus = &ns
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				next := true
				list, err := tx.SelectAllFrom(engine.TransactionTable, "WHERE invoice_id = $1", invID)
				if err != nil {
					return ctx, errors.Wrap(err, "Failed list transaction by invoice ID.")
				}
				for _, v := range list {
					tr := v.(*engine.Transaction)
					if !tr.Status.Match(engine.HOLD_TX) {
						next = false
						break
					}
				}
				if !next {
					return ctx, nil
				}
				// Установить статус после проделанных операций
				inv.Status = engine.WAIT_I
				inv.NextStatus = nil
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				return ctx, nil
			},
			"auth>wait",
		)
		s.s.Add(
			ffsm.State(engine.AUTH_I),
			ffsm.State(engine.ACCEPTED_I),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				invID, ok := payload.(int64)
				if !ok {
					log.Println("Invoice bad Payload: ", payload)
					return
				}
				ctx, span := trace.StartSpan(ctx, "ChangeState."+s.Name().String())
				defer span.End()
				span.AddAttributes(
					trace.Int64Attribute("invoice_id", invID),
					trace.StringAttribute("src_status", string(engine.AUTH_I)),
					trace.StringAttribute("dst_status", string(engine.ACCEPTED_I)),
				)
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				inv := engine.Invoice{InvoiceID: invID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				if inv.Strategy != nameStrategy.String() {
					return ctx, errors.New("Invoice strategy not nameStrategy.")
				}
				if inv.Status != engine.AUTH_I {
					return ctx, errors.New("Invoice status not auth.")
				}
				// Установить статус куда происходит переход
				ns := engine.ACCEPTED_I
				inv.NextStatus = &ns
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				next := true
				list, err := tx.SelectAllFrom(engine.TransactionTable, "WHERE invoice_id = $1", invID)
				if err != nil {
					return ctx, errors.Wrap(err, "Failed list transaction by invoice ID.")
				}
				for _, v := range list {
					tr := v.(*engine.Transaction)
					if !tr.Status.Match(engine.ACCEPTED_TX) {
						next = false
						break
					}
				}
				if !next {
					return ctx, nil
				}
				// Установить статус после проделанных операций
				inv.Status = engine.ACCEPTED_I
				inv.NextStatus = nil
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				return ctx, nil
			},
			"auth>accepted",
		)
		s.s.Add(
			ffsm.State(engine.AUTH_I),
			ffsm.State(engine.REJECTED_I),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				invID, ok := payload.(int64)
				if !ok {
					log.Println("Invoice bad Payload: ", payload)
					return
				}
				ctx, span := trace.StartSpan(ctx, "ChangeState."+s.Name().String())
				defer span.End()
				span.AddAttributes(
					trace.Int64Attribute("invoice_id", invID),
					trace.StringAttribute("src_status", string(engine.AUTH_I)),
					trace.StringAttribute("dst_status", string(engine.REJECTED_I)),
				)
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				inv := engine.Invoice{InvoiceID: invID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				if inv.Strategy != nameStrategy.String() {
					return ctx, errors.New("Invoice strategy not nameStrategy.")
				}
				if inv.Status != engine.AUTH_I {
					return ctx, errors.New("Invoice status not auth.")
				}
				// Установить статус куда происходит переход
				ns := engine.REJECTED_I
				inv.NextStatus = &ns
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				next := true
				list, err := tx.SelectAllFrom(engine.TransactionTable, "WHERE invoice_id = $1", invID)
				if err != nil {
					return ctx, errors.Wrap(err, "Failed list transaction by invoice ID.")
				}
				for _, v := range list {
					tr := v.(*engine.Transaction)
					if !tr.Status.Match(engine.REJECTED_TX) {
						next = false
						break
					}
				}
				if !next {
					return ctx, nil
				}
				// Установить статус после проделанных операций
				inv.Status = engine.REJECTED_I
				inv.NextStatus = nil
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				return ctx, nil
			},
			"auth>rejected",
		)
		s.s.Add(
			ffsm.State(engine.WAIT_I),
			ffsm.State(engine.ACCEPTED_I),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				invID, ok := payload.(int64)
				if !ok {
					log.Println("Invoice bad Payload: ", payload)
					return
				}
				ctx, span := trace.StartSpan(ctx, "ChangeState."+s.Name().String())
				defer span.End()
				span.AddAttributes(
					trace.Int64Attribute("invoice_id", invID),
					trace.StringAttribute("src_status", string(engine.WAIT_I)),
					trace.StringAttribute("dst_status", string(engine.ACCEPTED_I)),
				)
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				inv := engine.Invoice{InvoiceID: invID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				if inv.Strategy != nameStrategy.String() {
					return ctx, errors.New("Invoice strategy not nameStrategy.")
				}
				if inv.Status != engine.WAIT_I {
					return ctx, errors.New("Invoice status not wait.")
				}
				// Установить статус куда происходит переход
				ns := engine.ACCEPTED_I
				inv.NextStatus = &ns
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				next := true
				list, err := tx.SelectAllFrom(engine.TransactionTable, "WHERE invoice_id = $1", invID)
				if err != nil {
					return ctx, errors.Wrap(err, "Failed list transaction by invoice ID.")
				}
				for _, v := range list {
					tr := v.(*engine.Transaction)
					if !tr.Status.Match(engine.ACCEPTED_TX) {
						next = false
						break
					}
				}
				if !next {
					return ctx, nil
				}
				// Установить статус после проделанных операций
				inv.Status = engine.ACCEPTED_I
				inv.NextStatus = nil
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				return ctx, nil
			},
			"wait>accepted",
		)
		s.s.Add(
			ffsm.State(engine.WAIT_I),
			ffsm.State(engine.REJECTED_I),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				invID, ok := payload.(int64)
				if !ok {
					log.Println("Invoice bad Payload: ", payload)
					return
				}
				ctx, span := trace.StartSpan(ctx, "ChangeState."+s.Name().String())
				defer span.End()
				span.AddAttributes(
					trace.Int64Attribute("invoice_id", invID),
					trace.StringAttribute("src_status", string(engine.WAIT_I)),
					trace.StringAttribute("dst_status", string(engine.REJECTED_I)),
				)
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				inv := engine.Invoice{InvoiceID: invID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				if inv.Strategy != nameStrategy.String() {
					return ctx, errors.New("Invoice strategy not nameStrategy.")
				}
				if inv.Status != engine.WAIT_I {
					return ctx, errors.New("Invoice status not wait.")
				}
				// Установить статус куда происходит переход
				ns := engine.REJECTED_I
				inv.NextStatus = &ns
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				next := true
				list, err := tx.SelectAllFrom(engine.TransactionTable, "WHERE invoice_id = $1", invID)
				if err != nil {
					return ctx, errors.Wrap(err, "Failed list transaction by invoice ID.")
				}
				for _, v := range list {
					tr := v.(*engine.Transaction)
					if !tr.Status.Match(engine.REJECTED_TX) {
						next = false
						break
					}
				}
				if !next {
					return ctx, nil
				}
				// Установить статус после проделанных операций
				inv.Status = engine.REJECTED_I
				inv.NextStatus = nil
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				return ctx, nil
			},
			"wait>rejected",
		)
		s.s.Add(
			ffsm.State(engine.DRAFT_I),
			ffsm.State(engine.MREJECTED_I),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				invID, ok := payload.(int64)
				if !ok {
					log.Println("Invoice bad Payload: ", payload)
					return
				}
				ctx, span := trace.StartSpan(ctx, "ChangeState."+s.Name().String())
				defer span.End()
				span.AddAttributes(
					trace.Int64Attribute("invoice_id", invID),
					trace.StringAttribute("src_status", string(engine.DRAFT_I)),
					trace.StringAttribute("dst_status", string(engine.MREJECTED_I)),
				)
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				inv := engine.Invoice{InvoiceID: invID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				if inv.Strategy != nameStrategy.String() {
					return ctx, errors.New("Invoice strategy not nameStrategy.")
				}
				if inv.Status != engine.DRAFT_I {
					return ctx, errors.New("Invoice status not draft.")
				}
				// Установить статус куда происходит переход
				ns := engine.REJECTED_I
				inv.NextStatus = &ns
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				list, err := tx.SelectAllFrom(engine.TransactionTable, "WHERE invoice_id = $1", invID)
				if err != nil {
					return ctx, errors.Wrap(err, "Failed list transaction by invoice ID.")
				}
				pb := strategies.GetPubSubFromContext(ctx)
				if pb == nil {
					return ctx, errors.New("Not pubsub client in context.")
				}
				for _, v := range list {
					tr := v.(*engine.Transaction)

					b, err := json.Marshal(&strategies.MessageUpdateTransaction{
						ClientID:      tr.ClientID,
						TransactionID: tr.TransactionID,
						Strategy:      tr.Strategy,
						Status:        engine.REJECTED_TX,
					})
					if err != nil {
						zap.L().Error("Failed json marshal for publish to pubsub.",
							zap.Int64("InvoiceID", invID),
							zap.Int64("TransactionID", tr.TransactionID),
							zap.Error(err))
						continue
					}
					if _, err := pb.Topic(strategies.UPDATE_TRANSACTION_SUBJECT).Publish(ctx, &pubsub.Message{
						Data: b,
					}).Get(ctx); err != nil {
						zap.L().Error("Failed publish to pubsub.",
							zap.Int64("InvoiceID", invID),
							zap.Int64("TransactionID", tr.TransactionID),
							zap.Error(err))
						continue
					}
				}
				return ctx, nil
			},
			"draft>manual_rejected",
		)
		s.s.Add(
			ffsm.State(engine.AUTH_I),
			ffsm.State(engine.MACCEPTED_I),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				invID, ok := payload.(int64)
				if !ok {
					log.Println("Invoice bad Payload: ", payload)
					return
				}
				ctx, span := trace.StartSpan(ctx, "ChangeState."+s.Name().String())
				defer span.End()
				span.AddAttributes(
					trace.Int64Attribute("invoice_id", invID),
					trace.StringAttribute("src_status", string(engine.AUTH_I)),
					trace.StringAttribute("dst_status", string(engine.MACCEPTED_I)),
				)
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				inv := engine.Invoice{InvoiceID: invID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				if inv.Strategy != nameStrategy.String() {
					return ctx, errors.New("Invoice strategy not nameStrategy.")
				}
				if inv.Status != engine.AUTH_I {
					return ctx, errors.New("Invoice status not auth.")
				}
				// Установить статус куда происходит переход
				ns := engine.ACCEPTED_I
				inv.NextStatus = &ns
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				list, err := tx.SelectAllFrom(engine.TransactionTable, "WHERE invoice_id = $1", invID)
				if err != nil {
					return ctx, errors.Wrap(err, "Failed list transaction by invoice ID.")
				}
				pb := strategies.GetPubSubFromContext(ctx)
				if pb == nil {
					return ctx, errors.New("Not pubsub client in context.")
				}
				for _, v := range list {
					tr := v.(*engine.Transaction)
					b, err := json.Marshal(&strategies.MessageUpdateTransaction{
						ClientID:      tr.ClientID,
						TransactionID: tr.TransactionID,
						Strategy:      tr.Strategy,
						Status:        engine.ACCEPTED_TX,
					})
					if err != nil {
						zap.L().Error("Failed json marshal for publish to pubsub.",
							zap.Int64("InvoiceID", invID),
							zap.Int64("TransactionID", tr.TransactionID),
							zap.Error(err))
						continue
					}
					if _, err := pb.Topic(strategies.UPDATE_TRANSACTION_SUBJECT).Publish(ctx, &pubsub.Message{
						Data: b,
					}).Get(ctx); err != nil {
						zap.L().Error("Failed publish to pubsub.",
							zap.Int64("InvoiceID", invID),
							zap.Int64("TransactionID", tr.TransactionID),
							zap.Error(err))
						continue
					}
				}
				return ctx, nil
			},
			"auth>manual_accepted",
		)
		s.s.Add(
			ffsm.State(engine.AUTH_I),
			ffsm.State(engine.MREJECTED_I),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				invID, ok := payload.(int64)
				if !ok {
					log.Println("Invoice bad Payload: ", payload)
					return
				}
				ctx, span := trace.StartSpan(ctx, "ChangeState."+s.Name().String())
				defer span.End()
				span.AddAttributes(
					trace.Int64Attribute("invoice_id", invID),
					trace.StringAttribute("src_status", string(engine.AUTH_I)),
					trace.StringAttribute("dst_status", string(engine.MREJECTED_I)),
				)
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				inv := engine.Invoice{InvoiceID: invID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				if inv.Strategy != nameStrategy.String() {
					return ctx, errors.New("Invoice strategy not nameStrategy.")
				}
				if inv.Status != engine.AUTH_I {
					return ctx, errors.New("Invoice status not auth.")
				}
				// Установить статус куда происходит переход
				ns := engine.REJECTED_I
				inv.NextStatus = &ns
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				list, err := tx.SelectAllFrom(engine.TransactionTable, "WHERE invoice_id = $1", invID)
				if err != nil {
					return ctx, errors.Wrap(err, "Failed list transaction by invoice ID.")
				}
				pb := strategies.GetPubSubFromContext(ctx)
				if pb == nil {
					return ctx, errors.New("Not pubsub client in context.")
				}
				for _, v := range list {
					tr := v.(*engine.Transaction)
					b, err := json.Marshal(&strategies.MessageUpdateTransaction{
						ClientID:      tr.ClientID,
						TransactionID: tr.TransactionID,
						Strategy:      tr.Strategy,
						Status:        engine.REJECTED_TX,
					})
					if err != nil {
						zap.L().Error("Failed json marshal for publish to pubsub.",
							zap.Int64("InvoiceID", invID),
							zap.Int64("TransactionID", tr.TransactionID),
							zap.Error(err))
						continue
					}
					if _, err := pb.Topic(strategies.UPDATE_TRANSACTION_SUBJECT).Publish(ctx, &pubsub.Message{
						Data: b,
					}).Get(ctx); err != nil {
						zap.L().Error("Failed publish to pubsub.",
							zap.Int64("InvoiceID", invID),
							zap.Int64("TransactionID", tr.TransactionID),
							zap.Error(err))
						continue
					}
				}
				return ctx, nil
			},
			"auth>manual_rejected",
		)
		s.s.Add(
			ffsm.State(engine.WAIT_I),
			ffsm.State(engine.MACCEPTED_I),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				invID, ok := payload.(int64)
				if !ok {
					log.Println("Invoice bad Payload: ", payload)
					return
				}
				ctx, span := trace.StartSpan(ctx, "ChangeState."+s.Name().String())
				defer span.End()
				span.AddAttributes(
					trace.Int64Attribute("invoice_id", invID),
					trace.StringAttribute("src_status", string(engine.WAIT_I)),
					trace.StringAttribute("dst_status", string(engine.MACCEPTED_I)),
				)
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				inv := engine.Invoice{InvoiceID: invID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				if inv.Strategy != nameStrategy.String() {
					return ctx, errors.New("Invoice strategy not nameStrategy.")
				}
				if inv.Status != engine.WAIT_I {
					return ctx, errors.New("Invoice status not wait.")
				}
				// Установить статус куда происходит переход
				ns := engine.ACCEPTED_I
				inv.NextStatus = &ns
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				list, err := tx.SelectAllFrom(engine.TransactionTable, "WHERE invoice_id = $1", invID)
				if err != nil {
					return ctx, errors.Wrap(err, "Failed list transaction by invoice ID.")
				}
				pb := strategies.GetPubSubFromContext(ctx)
				if pb == nil {
					return ctx, errors.New("Not pubsub client in context.")
				}
				for _, v := range list {
					tr := v.(*engine.Transaction)
					b, err := json.Marshal(&strategies.MessageUpdateTransaction{
						ClientID:      tr.ClientID,
						TransactionID: tr.TransactionID,
						Strategy:      tr.Strategy,
						Status:        engine.ACCEPTED_TX,
					})
					if err != nil {
						zap.L().Error("Failed json marshal for publish to pubsub.",
							zap.Int64("InvoiceID", invID),
							zap.Int64("TransactionID", tr.TransactionID),
							zap.Error(err))
						continue
					}
					if _, err := pb.Topic(strategies.UPDATE_TRANSACTION_SUBJECT).Publish(ctx, &pubsub.Message{
						Data: b,
					}).Get(ctx); err != nil {
						zap.L().Error("Failed publish to pubsub.",
							zap.Int64("InvoiceID", invID),
							zap.Int64("TransactionID", tr.TransactionID),
							zap.Error(err))
						continue
					}
				}
				return ctx, nil
			},
			"wait>manual_accepted",
		)
		s.s.Add(
			ffsm.State(engine.WAIT_I),
			ffsm.State(engine.MREJECTED_I),
			func(ctx context.Context, payload ffsm.Payload) (context context.Context, e error) {
				invID, ok := payload.(int64)
				if !ok {
					log.Println("Invoice bad Payload: ", payload)
					return
				}
				ctx, span := trace.StartSpan(ctx, "ChangeState."+s.Name().String())
				defer span.End()
				span.AddAttributes(
					trace.Int64Attribute("invoice_id", invID),
					trace.StringAttribute("src_status", string(engine.WAIT_I)),
					trace.StringAttribute("dst_status", string(engine.MREJECTED_I)),
				)
				tx := strategies.GetTXContext(ctx)
				if tx == nil {
					return ctx, errors.New("Not reform tx in context.")
				}
				inv := engine.Invoice{InvoiceID: invID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				if inv.Strategy != nameStrategy.String() {
					return ctx, errors.New("Invoice strategy not nameStrategy.")
				}
				if inv.Status != engine.WAIT_I {
					return ctx, errors.New("Invoice status not wait.")
				}
				// Установить статус куда происходит переход
				ns := engine.REJECTED_I
				inv.NextStatus = &ns
				if err := tx.Save(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed save invoice by ID.")
				}
				list, err := tx.SelectAllFrom(engine.TransactionTable, "WHERE invoice_id = $1", invID)
				if err != nil {
					return ctx, errors.Wrap(err, "Failed list transaction by invoice ID.")
				}
				pb := strategies.GetPubSubFromContext(ctx)
				if pb == nil {
					return ctx, errors.New("Not pubsub client in context.")
				}
				for _, v := range list {
					tr := v.(*engine.Transaction)
					b, err := json.Marshal(&strategies.MessageUpdateTransaction{
						ClientID:      tr.ClientID,
						TransactionID: tr.TransactionID,
						Strategy:      tr.Strategy,
						Status:        engine.REJECTED_TX,
					})
					if err != nil {
						zap.L().Error("Failed json marshal for publish to pubsub.",
							zap.Int64("InvoiceID", invID),
							zap.Int64("TransactionID", tr.TransactionID),
							zap.Error(err))
						continue
					}
					if _, err := pb.Topic(strategies.UPDATE_TRANSACTION_SUBJECT).Publish(ctx, &pubsub.Message{
						Data: b,
					}).Get(ctx); err != nil {
						zap.L().Error("Failed publish to pubsub.",
							zap.Int64("InvoiceID", invID),
							zap.Int64("TransactionID", tr.TransactionID),
							zap.Error(err))
						continue
					}
				}
				return ctx, nil
			},
			"wait>manual_rejected",
		)
	})
}
