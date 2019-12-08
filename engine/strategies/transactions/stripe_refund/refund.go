package stripe_refund

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"

	pkgStripe "github.com/stripe/stripe-go"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/engine/strategies"
	"github.com/gebv/acca/ffsm"
	"github.com/gebv/acca/provider"
	"github.com/gebv/acca/provider/stripe"
)

const nameStrategy strategies.TrStrategyName = "transaction_stripe_refund_strategy"

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

func (s *Strategy) Provider() provider.Provider {
	return stripe.STRIPE
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
					return ctx, errors.New("Transaction strategy not TrStripeStrategy.")
				}
				if tr.Status != engine.DRAFT_TX {
					return ctx, errors.New("Transaction status not draft.")
				}
				fsTx := strategies.GetFirestoreTxFromContext(ctx)
				if fsTx == nil {
					return ctx, errors.New("Not fs transaction in context.")
				}
				fs := strategies.GetFirestoreClientFromContext(ctx)
				if fs == nil {
					return ctx, errors.New("Not fs client in context.")
				}
				if tr.Meta == nil {
					return ctx, errors.New("Transaction not set meta.")
				}
				meta := make(map[string]string)
				if err := json.Unmarshal(*tr.Meta, &meta); err != nil {
					return ctx, errors.Wrap(err, "Failed unmarshal meta in transaction.")
				}
				refTxID, err := strconv.ParseInt(meta["tx_id"], 10, 64)
				if err != nil {
					return ctx, errors.Wrap(err, "Failed convert tx_id from transaction meta.")
				}
				refTx := engine.Transaction{TransactionID: refTxID}
				if err := tx.Reload(&refTx); err != nil {
					return ctx, errors.Wrap(err, "Failed reload transaction by ID.")
				}
				if refTx.Status != engine.ACCEPTED_TX {
					return ctx, errors.New("Transaction status not accepted.")
				}
				if refTx.ProviderOperID == nil {
					return ctx, errors.New("External order ID in stripe is nil.")
				}
				list, err := tx.SelectAllFrom(
					engine.TransactionTable,
					"WHERE strategy = $1 AND provider_oper_id = $2 AND status = $3",
					tr.Strategy,
					*refTx.ProviderOperID,
					engine.ACCEPTED_TX,
				)
				if err != nil {
					return ctx, errors.Wrap(err, "Failed list transaction by invoice ID.")
				}
				var refundAmount int64
				for _, v := range list {
					tr := v.(*engine.Transaction)
					refundAmount += tr.Amount
				}
				if refTx.Amount < (refundAmount + tr.Amount) {
					return ctx, errors.New("No funds to refund.")
				}
				if refTx.ProviderOperStatus == nil || *refTx.ProviderOperStatus != string(pkgStripe.PaymentIntentStatusSucceeded) {
					return ctx, errors.New("Bad status transaction in refund, need succeeded.")
				}
				// Установить статус куда происходит переход
				ns := engine.AUTH_TX
				tr.NextStatus = &ns
				tr.Status = engine.WAUTH_TX
				tr.ProviderOperID = refTx.ProviderOperID
				tr.ProviderOperStatus = refTx.ProviderOperStatus
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				if err := fsTx.Create(fs.Collection("messages").NewDoc(), struct {
					Type      string `firestore:"type"`
					StatusMsg string `firestore:"status_msg"`
					CreatedAt int64  `firestore:"created_at"`
					stripe.MessageToStripe
				}{
					Type:      stripe.SUBJECT,
					StatusMsg: "new",
					CreatedAt: time.Now().UnixNano(),
					MessageToStripe: stripe.MessageToStripe{
						Command:       stripe.Refund,
						ClientID:      tr.ClientID,
						TransactionID: tr.TransactionID,
						Strategy:      tr.Strategy,
						Status:        engine.AUTH_TX,
					},
				}); err != nil {
					return ctx, errors.Wrap(err, "Failed create message.")
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
					return ctx, errors.New("Transaction strategy not TrStripeStrategy.")
				}
				if tr.Status != engine.WAUTH_TX {
					return ctx, errors.New("Transaction status not auth_wait.")
				}
				inv := engine.Invoice{InvoiceID: tr.InvoiceID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				fsTx := strategies.GetFirestoreTxFromContext(ctx)
				if fsTx == nil {
					return ctx, errors.New("Not fs transaction in context.")
				}
				fs := strategies.GetFirestoreClientFromContext(ctx)
				if fs == nil {
					return ctx, errors.New("Not fs client in context.")
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
				if isHold {
					return ctx, errors.Wrap(err, "Support only non hold operation.")
				}
				invStatus := engine.ACCEPTED_I
				tr.Status = engine.ACCEPTED_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				if err := fsTx.Create(fs.Collection("messages").NewDoc(), struct {
					Type      string `firestore:"type"`
					StatusMsg string `firestore:"status_msg"`
					CreatedAt int64  `firestore:"created_at"`
					strategies.MessageUpdateInvoice
				}{
					Type:      strategies.UPDATE_INVOICE_SUBJECT,
					StatusMsg: "new",
					CreatedAt: time.Now().UnixNano(),
					MessageUpdateInvoice: strategies.MessageUpdateInvoice{
						ClientID:  inv.ClientID,
						InvoiceID: inv.InvoiceID,
						Strategy:  inv.Strategy,
						Status:    invStatus,
					},
				}); err != nil {
					return ctx, errors.Wrap(err, "Failed create message.")
				}
				return ctx, nil
			},
			"auth_wait>auth",
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
					return ctx, errors.New("Transaction strategy not TrStripeStrategy.")
				}
				if tr.Status != engine.DRAFT_TX {
					return ctx, errors.New("Transaction status not draft.")
				}
				inv := engine.Invoice{InvoiceID: tr.InvoiceID}
				if err := tx.Reload(&inv); err != nil {
					return ctx, errors.Wrap(err, "Failed reload invoice by ID.")
				}
				fsTx := strategies.GetFirestoreTxFromContext(ctx)
				if fsTx == nil {
					return ctx, errors.New("Not fs transaction in context.")
				}
				fs := strategies.GetFirestoreClientFromContext(ctx)
				if fs == nil {
					return ctx, errors.New("Not fs client in context.")
				}
				// Установить статус куда происходит переход
				tr.Status = engine.REJECTED_TX
				tr.NextStatus = nil
				if err := tx.Save(&tr); err != nil {
					return ctx, errors.Wrap(err, "Failed save transaction by ID.")
				}
				if err := fsTx.Create(fs.Collection("messages").NewDoc(), struct {
					Type      string `firestore:"type"`
					StatusMsg string `firestore:"status_msg"`
					CreatedAt int64  `firestore:"created_at"`
					strategies.MessageUpdateInvoice
				}{
					Type:      strategies.UPDATE_INVOICE_SUBJECT,
					StatusMsg: "new",
					CreatedAt: time.Now().UnixNano(),
					MessageUpdateInvoice: strategies.MessageUpdateInvoice{
						ClientID:  inv.ClientID,
						InvoiceID: inv.InvoiceID,
						Strategy:  inv.Strategy,
						Status:    engine.REJECTED_I,
					},
				}); err != nil {
					return ctx, errors.Wrap(err, "Failed create message.")
				}
				return ctx, nil
			},
			"draft>rejected",
		)
	})
}
