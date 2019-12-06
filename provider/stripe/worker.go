package stripe

import (
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/pkg/errors"
	"github.com/stripe/stripe-go"
	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/engine/strategies"
)

type Command string

const (
	SUBJECT                = "provider_stripe_subject"
	AuthTransfer   Command = "auth_transfer"
	Capture        Command = "capture"
	ReverseForHold Command = "reverse_for_hold"
	Refund         Command = "refund"
)

type MessageToStripe struct {
	Command       Command
	ClientID      *int64
	TransactionID int64
	Strategy      string
	Status        engine.TransactionStatus
}

func (p *Provider) WorkerHandler(fsCtx context.Context) {
	if docs, err := p.fs.Collection("messages").Where("type", "==", SUBJECT).Where("status_msg", "==", "new").OrderBy("created_at", firestore.Asc).Limit(1).Documents(fsCtx).GetAll(); err == nil {
		for _, doc := range docs {
			if _, err := doc.Ref.Update(fsCtx, []firestore.Update{
				{
					Path:  "status_msg",
					Value: "in_progress",
				},
			}); err != nil {
				p.l.Error("Failed update message", zap.Error(err))
				break
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := p.fs.RunTransaction(ctx, func(ctx context.Context, fsTx *firestore.Transaction) error {
				var okProcessing bool
				defer func() {
					status := "error"
					if okProcessing {
						status = "success"
					}
					if err := fsTx.Update(doc.Ref, []firestore.Update{
						{
							Path:  "status_msg",
							Value: status,
						},
					}); err != nil {
						p.l.Error("Failed update status message", zap.Error(err))
					}
				}()
				var m MessageToStripe
				ctx, span := trace.StartSpan(ctx, "async.fromQueue.update."+SUBJECT)
				defer span.End()
				if err := doc.DataTo(&m); err != nil {
					return errors.Wrap(err, "Failed data to message.")
				}
				span.AddAttributes(
					trace.StringAttribute("strategy", m.Strategy),
					trace.StringAttribute("status", string(m.Status)),
				)
				tx, err := p.db.Begin()
				if err != nil {
					return errors.Wrap(err, "Failed begin transaction DB.")
				}
				switch m.Command {
				case AuthTransfer:
					tr := engine.Transaction{TransactionID: m.TransactionID}
					if err := tx.Reload(&tr); err != nil {
						if err := tx.Rollback(); err != nil {
							p.l.Error("Failed tx rollback. ", zap.Error(err))
						}
						return errors.Wrap(err, "Failed dispatch invoice strategy.")
					}
					if tr.Status != engine.WAUTH_TX {
						p.l.Warn(
							"Transaction status not auth_wait.",
							zap.Int64("tr_id", tr.TransactionID),
							zap.String("status", string(tr.Status)),
						)
						if err := tx.Rollback(); err != nil {
							return errors.Wrap(err, "Failed tx rolback.")
						}
						return nil
					}
					if tr.Meta == nil {
						p.l.Warn(
							"Transaction not set meta.",
							zap.Int64("tr_id", tr.TransactionID),
						)
						if err := tx.Rollback(); err != nil {
							p.l.Error("Failed tx rollback. ", zap.Error(err))
						}
						return nil
					}
					meta := make(map[string]string)
					if err := json.Unmarshal(*tr.Meta, &meta); err != nil {
						p.l.Error("Failed unmarshal meta in transaction. ", zap.Error(err))
						if err := tx.Rollback(); err != nil {
							p.l.Error("Failed tx rollback. ", zap.Error(err))
						}
						return nil
					}
					var hold bool
					if err, hold = engine.IsHoldOperations(tx, tr.TransactionID); err != nil {
						p.l.Error("Failed select operation in transaction. ", zap.Error(err))
						if err := tx.Rollback(); err != nil {
							p.l.Error("Failed tx rollback. ", zap.Error(err))
						}
						return nil
					}
					var paymentIntent *stripe.PaymentIntent
					if hold {
						paymentIntent, err = p.PaymentIntentWithHold(
							tr.Amount,
							stripe.CurrencyUSD,
							meta["customer_id"],
							meta["pm_id"],
							false,
						)
					} else {
						paymentIntent, err = p.PaymentIntent(
							tr.Amount,
							stripe.CurrencyUSD,
							meta["customer_id"],
							meta["pm_id"],
							false,
						)
					}
					if err != nil {
						p.l.Error("Failed payment intent in stripe.", zap.Error(err))
						if err := tx.Rollback(); err != nil {
							p.l.Error("Failed tx rollback. ", zap.Error(err))
						}
						return nil
					}
					tr.ProviderOperID = &paymentIntent.ID
					status := string(paymentIntent.Status)
					tr.ProviderOperStatus = &status
					tr.ProviderOperUrl = &paymentIntent.ClientSecret
					if err := tx.Save(&tr); err != nil {
						p.l.Error("Failed save transaction. ", zap.Error(err))
						if err := tx.Rollback(); err != nil {
							p.l.Error("Failed tx rollback. ", zap.Error(err))
						}
						return nil
					}
					if err := tx.Commit(); err != nil {
						return errors.Wrap(err, "Failed tx commit.")
					}
					if err := fsTx.Create(p.fs.Collection("messages").NewDoc(), struct {
						Type      string `firestore:"type"`
						StatusMsg string `firestore:"status_msg"`
						CreatedAt int64  `firestore:"created_at"`
						strategies.MessageUpdateTransaction
					}{
						Type:      strategies.UPDATE_TRANSACTION_SUBJECT,
						StatusMsg: "new",
						CreatedAt: time.Now().UnixNano(),
						MessageUpdateTransaction: strategies.MessageUpdateTransaction{
							ClientID:      m.ClientID,
							TransactionID: m.TransactionID,
							Strategy:      m.Strategy,
							Status:        m.Status,
						},
					}); err != nil {
						return errors.Wrap(err, "Failed create message for update transaction.")
					}
					okProcessing = true
					return nil
				case ReverseForHold:
					tr := engine.Transaction{TransactionID: m.TransactionID}
					if err := tx.Reload(&tr); err != nil {
						if err := tx.Rollback(); err != nil {
							p.l.Error("Failed tx rollback. ", zap.Error(err))
						}
						return errors.Wrap(err, "Failed reload transaction.")
					}
					if tr.Status != engine.WREJECTED_TX {
						p.l.Warn(
							"Transaction status not rejected_wait.",
							zap.Int64("tr_id", tr.TransactionID),
							zap.String("status", string(tr.Status)),
						)
						if err := tx.Rollback(); err != nil {
							return errors.Wrap(err, "Failed tx rollback.")
						}
						return nil
					}
					if tr.ProviderOperID == nil {
						p.l.Warn("External order ID in stripe is nil.")
						if err := tx.Rollback(); err != nil {
							return errors.Wrap(err, "Failed tx rollback.")
						}
						return nil
					}
					if tr.ProviderOperStatus == nil {
						p.l.Warn("status in stripe is nil.")
						if err := tx.Rollback(); err != nil {
							return errors.Wrap(err, "Failed tx rollback.")
						}
						return nil
					}
					if *tr.ProviderOperStatus != string(stripe.PaymentIntentStatusRequiresCapture) {
						p.l.Warn(
							"status in stripe not match.",
							zap.String("provider_oper_status", *tr.ProviderOperStatus),
						)
						if err := tx.Rollback(); err != nil {
							return errors.Wrap(err, "Failed tx rollback.")
						}
						return nil
					}
					err := p.Cancel(
						*tr.ProviderOperID,
					)
					if err != nil {
						if err := tx.Rollback(); err != nil {
							p.l.Error("Failed tx rollback. ", zap.Error(err))
						}
						return errors.Wrap(err, "Failed reverse for hold in stripe.")
					}
					status := string(stripe.PaymentIntentStatusCanceled)
					tr.ProviderOperStatus = &status
					if err := tx.Save(&tr); err != nil {
						if err := tx.Rollback(); err != nil {
							p.l.Error("Failed tx rollback. ", zap.Error(err))
						}
						return errors.Wrap(err, "Failed save transaction.")
					}
					if err := tx.Commit(); err != nil {
						return errors.Wrap(err, "Failed tx commit.")
					}
					if err := fsTx.Create(p.fs.Collection("messages").NewDoc(), struct {
						Type      string `firestore:"type"`
						StatusMsg string `firestore:"status_msg"`
						CreatedAt int64  `firestore:"created_at"`
						strategies.MessageUpdateTransaction
					}{
						Type:      strategies.UPDATE_TRANSACTION_SUBJECT,
						StatusMsg: "new",
						CreatedAt: time.Now().UnixNano(),
						MessageUpdateTransaction: strategies.MessageUpdateTransaction{
							ClientID:      m.ClientID,
							TransactionID: m.TransactionID,
							Strategy:      m.Strategy,
							Status:        m.Status,
						},
					}); err != nil {
						return errors.Wrap(err, "Failed create message for update transaction.")
					}
					okProcessing = true
					return nil
				case Capture:
					tr := engine.Transaction{TransactionID: m.TransactionID}
					if err := tx.Reload(&tr); err != nil {
						if err := tx.Rollback(); err != nil {
							p.l.Error("Failed tx rollback. ", zap.Error(err))
						}
						return errors.Wrap(err, "Failed reload transaction.")
					}
					if tr.Status != engine.WACCEPTED_TX {
						p.l.Warn(
							"Transaction status not rejected_wait.",
							zap.Int64("tr_id", tr.TransactionID),
							zap.String("status", string(tr.Status)),
						)
						if err := tx.Rollback(); err != nil {
							return errors.Wrap(err, "Failed tx rollback.")
						}
						return nil
					}
					if tr.ProviderOperID == nil {
						p.l.Warn("External order ID in stripe is nil.")
						if err := tx.Rollback(); err != nil {
							return errors.Wrap(err, "Failed tx rollback.")
						}
						return nil
					}
					if tr.ProviderOperStatus == nil {
						p.l.Warn("status in stripe is nil.")
						if err := tx.Rollback(); err != nil {
							return errors.Wrap(err, "Failed tx rollback.")
						}
						return nil
					}
					if *tr.ProviderOperStatus != string(stripe.PaymentIntentStatusRequiresCapture) {
						p.l.Warn(
							"status in stripe not match.",
							zap.String("provider_oper_status", *tr.ProviderOperStatus),
						)
						if err := tx.Rollback(); err != nil {
							return errors.Wrap(err, "Failed tx rollback.")
						}
						return nil
					}
					err := p.Capture(
						*tr.ProviderOperID,
						tr.Amount,
					)
					if err != nil {
						if err := tx.Rollback(); err != nil {
							p.l.Error("Failed tx rollback. ", zap.Error(err))
						}
						return errors.Wrap(err, "Failed capture in stripe.")
					}
					status := string(stripe.PaymentIntentStatusSucceeded)
					tr.ProviderOperStatus = &status
					if err := tx.Save(&tr); err != nil {
						if err := tx.Rollback(); err != nil {
							p.l.Error("Failed tx rollback. ", zap.Error(err))
						}
						return errors.Wrap(err, "Failed save transaction.")
					}
					if err := tx.Commit(); err != nil {
						return errors.Wrap(err, "Failed tx commit.")
					}
					if err := fsTx.Create(p.fs.Collection("messages").NewDoc(), struct {
						Type      string `firestore:"type"`
						StatusMsg string `firestore:"status_msg"`
						CreatedAt int64  `firestore:"created_at"`
						strategies.MessageUpdateTransaction
					}{
						Type:      strategies.UPDATE_TRANSACTION_SUBJECT,
						StatusMsg: "new",
						CreatedAt: time.Now().UnixNano(),
						MessageUpdateTransaction: strategies.MessageUpdateTransaction{
							ClientID:      m.ClientID,
							TransactionID: m.TransactionID,
							Strategy:      m.Strategy,
							Status:        m.Status,
						},
					}); err != nil {
						return errors.Wrap(err, "Failed create message for update transaction.")
					}
					okProcessing = true
					return nil
				case Refund:
					tr := engine.Transaction{TransactionID: m.TransactionID}
					if err := tx.Reload(&tr); err != nil {
						if err := tx.Rollback(); err != nil {
							p.l.Error("Failed tx rollback. ", zap.Error(err))
						}
						return errors.Wrap(err, "Failed reload transaction.")
					}
					if tr.Status != engine.WAUTH_TX {
						p.l.Warn(
							"Transaction status not auth_wait.",
							zap.Int64("tr_id", tr.TransactionID),
							zap.String("status", string(tr.Status)),
						)
						if err := tx.Rollback(); err != nil {
							return errors.Wrap(err, "Failed tx rollback.")
						}
						return nil
					}
					if tr.ProviderOperID == nil {
						p.l.Warn("External order ID in stripe is nil.")
						if err := tx.Rollback(); err != nil {
							return errors.Wrap(err, "Failed tx rollback.")
						}
						return nil
					}
					if tr.ProviderOperStatus == nil {
						p.l.Warn("status in stripe is nil.")
						if err := tx.Rollback(); err != nil {
							return errors.Wrap(err, "Failed tx rollback.")
						}
						return nil
					}
					if *tr.ProviderOperStatus != string(stripe.PaymentIntentStatusSucceeded) {
						p.l.Warn(
							"status in stripe not match.",
							zap.String("provider_oper_status", *tr.ProviderOperStatus),
						)
						if err := tx.Rollback(); err != nil {
							return errors.Wrap(err, "Failed tx rollback.")
						}
						return nil
					}
					pi, err := p.GetPaymentIntent(*tr.ProviderOperID)
					if err != nil {
						if err := tx.Rollback(); err != nil {
							p.l.Error("Failed tx rollback. ", zap.Error(err))
						}
						return errors.Wrap(err, "Failed get payment intent in stripe.")
					}
					if len(pi.Charges.Data) > 1 {
						if err := tx.Rollback(); err != nil {
							p.l.Error("Failed tx rollback. ", zap.Error(err))
						}
						return errors.Wrap(err, "Failed len charges payment intent in stripe is > 1.")
					}
					if len(pi.Charges.Data) == 0 {
						p.l.Error("Failed len charges payment intent in stripe is 0.", zap.Error(err))
						if err := tx.Rollback(); err != nil {
							return errors.Wrap(err, "Failed tx rollback.")
						}
						return nil
					}
					err = p.Refund(
						pi.Charges.Data[0].ID,
						&tr.Amount,
					)
					if err != nil {
						p.l.Error("Failed refund payment intent in stripe.",
							zap.String("payment_intent_id", *tr.ProviderOperID),
							zap.String("charge_id", pi.Charges.Data[0].ID),
							zap.Error(err),
						)
						if err := tx.Rollback(); err != nil {
							return errors.Wrap(err, "Failed tx rollback.")
						}
						return nil
					}

					status := string(stripe.PaymentIntentStatusCanceled)
					tr.ProviderOperStatus = &status
					if err := tx.Save(&tr); err != nil {
						p.l.Error("Failed save transaction. ", zap.Error(err))
						if err := tx.Rollback(); err != nil {
							p.l.Error("Failed tx rollback. ", zap.Error(err))
						}
						return nil
					}
					if err := tx.Commit(); err != nil {
						return errors.Wrap(err, "Failed tx commit. ")
					}
					if err := fsTx.Create(p.fs.Collection("messages").NewDoc(), struct {
						Type      string `firestore:"type"`
						StatusMsg string `firestore:"status_msg"`
						CreatedAt int64  `firestore:"created_at"`
						strategies.MessageUpdateTransaction
					}{
						Type:      strategies.UPDATE_TRANSACTION_SUBJECT,
						StatusMsg: "new",
						CreatedAt: time.Now().UnixNano(),
						MessageUpdateTransaction: strategies.MessageUpdateTransaction{
							ClientID:      m.ClientID,
							TransactionID: m.TransactionID,
							Strategy:      m.Strategy,
							Status:        m.Status,
						},
					}); err != nil {
						return errors.Wrap(err, "Failed create message for update transaction.")
					}
					okProcessing = true
					return nil
				default:
					p.l.Warn("Not processed command in message of stripe in nats.")
					if err := tx.Rollback(); err != nil {
						return errors.Wrap(err, "Failed tx rollback.")
					}
					return nil
				}
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return nil
			}); err != nil {
				p.l.Error("Failed run transaction", zap.Error(err))
			}
			cancel()
		}
	} else {
		p.l.Error("Failed get messages. ", zap.Error(err))
	}
}
