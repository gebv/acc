package stripe

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/pubsub"
	"github.com/stripe/stripe-go"
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

func (p *Provider) PubSubHandler() func(ctx context.Context, pbMsg *pubsub.Message) {
	return func(ctx context.Context, pbMsg *pubsub.Message) {
		var m MessageToStripe
		var nack bool
		okAck := &nack
		defer func() {
			if okAck == nil {
				return
			}
			if *okAck {
				pbMsg.Ack()
			} else {
				pbMsg.Nack()
			}
		}()
		if err := json.Unmarshal(pbMsg.Data, &m); err != nil {
			p.l.Error("Failed unmarshal pubsub message.", zap.Error(err))
			return
		}
		tx, err := p.db.Begin()
		if err != nil {
			p.l.Error("Failed begin transaction DB.", zap.Error(err))
			return
		}
		switch m.Command {
		case AuthTransfer:
			tr := engine.Transaction{TransactionID: m.TransactionID}
			if err := tx.Reload(&tr); err != nil {
				p.l.Error("Failed reload transaction. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if tr.Status != engine.WAUTH_TX {
				p.l.Warn(
					"Transaction status not auth_wait.",
					zap.Int64("tr_id", tr.TransactionID),
					zap.String("status", string(tr.Status)),
				)
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				okAck = nil
				return
			}
			if tr.Meta == nil {
				p.l.Warn(
					"Transaction not set meta.",
					zap.Int64("tr_id", tr.TransactionID),
				)
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			meta := make(map[string]string)
			if err := json.Unmarshal(*tr.Meta, &meta); err != nil {
				p.l.Error("Failed unmarshal meta in transaction. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			var hold bool
			if err, hold = engine.IsHoldOperations(tx, tr.TransactionID); err != nil {
				p.l.Error("Failed select operation in transaction. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
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
				return
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
				return
			}
			if err := tx.Commit(); err != nil {
				p.l.Error("Failed tx commit. ", zap.Error(err))
				return
			}
			b, err := json.Marshal(&strategies.MessageUpdateTransaction{
				ClientID:      m.ClientID,
				TransactionID: m.TransactionID,
				Strategy:      m.Strategy,
				Status:        m.Status,
			})
			if err != nil {
				p.l.Error("Failed json marshal for publish update transaction.", zap.Error(err))
				return
			}
			if _, err := p.pb.Topic(strategies.UPDATE_TRANSACTION_SUBJECT).Publish(context.Background(), &pubsub.Message{
				Data: b,
			}).Get(context.Background()); err != nil {
				p.l.Error("Failed publish update transaction.", zap.Error(err))
			}
			*okAck = true
		case ReverseForHold:
			tr := engine.Transaction{TransactionID: m.TransactionID}
			if err := tx.Reload(&tr); err != nil {
				p.l.Error("Failed reload transaction. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if tr.Status != engine.WREJECTED_TX {
				p.l.Warn(
					"Transaction status not rejected_wait.",
					zap.Int64("tr_id", tr.TransactionID),
					zap.String("status", string(tr.Status)),
				)
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				okAck = nil
				return
			}
			if tr.ProviderOperID == nil {
				p.l.Warn("External order ID in stripe is nil.")
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if tr.ProviderOperStatus == nil {
				p.l.Warn("status in stripe is nil.")
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if *tr.ProviderOperStatus != string(stripe.PaymentIntentStatusRequiresCapture) {
				p.l.Warn(
					"status in stripe not match.",
					zap.String("provider_oper_status", *tr.ProviderOperStatus),
				)
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			err := p.Cancel(
				*tr.ProviderOperID,
			)
			if err != nil {
				p.l.Error("Failed reverse for hold in stripe.", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			status := string(stripe.PaymentIntentStatusCanceled)
			tr.ProviderOperStatus = &status
			if err := tx.Save(&tr); err != nil {
				p.l.Error("Failed save transaction. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if err := tx.Commit(); err != nil {
				p.l.Error("Failed tx commit. ", zap.Error(err))
				return
			}
			b, err := json.Marshal(&strategies.MessageUpdateTransaction{
				ClientID:      m.ClientID,
				TransactionID: m.TransactionID,
				Strategy:      m.Strategy,
				Status:        m.Status,
			})
			if err != nil {
				p.l.Error("Failed json marshal for publish update transaction.", zap.Error(err))
				return
			}
			if _, err := p.pb.Topic(strategies.UPDATE_TRANSACTION_SUBJECT).Publish(context.Background(), &pubsub.Message{
				Data: b,
			}).Get(context.Background()); err != nil {
				p.l.Error("Failed publish update transaction.", zap.Error(err))
			}
			*okAck = true
		case Capture:
			tr := engine.Transaction{TransactionID: m.TransactionID}
			if err := tx.Reload(&tr); err != nil {
				p.l.Error("Failed reload transaction. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if tr.Status != engine.WACCEPTED_TX {
				p.l.Warn(
					"Transaction status not rejected_wait.",
					zap.Int64("tr_id", tr.TransactionID),
					zap.String("status", string(tr.Status)),
				)
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				okAck = nil
				return
			}
			if tr.ProviderOperID == nil {
				p.l.Warn("External order ID in stripe is nil.")
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if tr.ProviderOperStatus == nil {
				p.l.Warn("status in stripe is nil.")
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if *tr.ProviderOperStatus != string(stripe.PaymentIntentStatusRequiresCapture) {
				p.l.Warn(
					"status in stripe not match.",
					zap.String("provider_oper_status", *tr.ProviderOperStatus),
				)
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			err := p.Capture(
				*tr.ProviderOperID,
				tr.Amount,
			)
			if err != nil {
				p.l.Error("Failed capture in stripe.", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			status := string(stripe.PaymentIntentStatusSucceeded)
			tr.ProviderOperStatus = &status
			if err := tx.Save(&tr); err != nil {
				p.l.Error("Failed save transaction. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if err := tx.Commit(); err != nil {
				p.l.Error("Failed tx commit. ", zap.Error(err))
				return
			}
			b, err := json.Marshal(&strategies.MessageUpdateTransaction{
				ClientID:      m.ClientID,
				TransactionID: m.TransactionID,
				Strategy:      m.Strategy,
				Status:        m.Status,
			})
			if err != nil {
				p.l.Error("Failed json marshal for publish update transaction.", zap.Error(err))
				return
			}
			if _, err := p.pb.Topic(strategies.UPDATE_TRANSACTION_SUBJECT).Publish(context.Background(), &pubsub.Message{
				Data: b,
			}).Get(context.Background()); err != nil {
				p.l.Error("Failed publish update transaction.", zap.Error(err))
			}
			*okAck = true
		case Refund:
			tr := engine.Transaction{TransactionID: m.TransactionID}
			if err := tx.Reload(&tr); err != nil {
				p.l.Error("Failed reload transaction. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if tr.Status != engine.WAUTH_TX {
				p.l.Warn(
					"Transaction status not auth_wait.",
					zap.Int64("tr_id", tr.TransactionID),
					zap.String("status", string(tr.Status)),
				)
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				okAck = nil
				return
			}
			if tr.ProviderOperID == nil {
				p.l.Warn("External order ID in stripe is nil.")
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if tr.ProviderOperStatus == nil {
				p.l.Warn("status in stripe is nil.")
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if *tr.ProviderOperStatus != string(stripe.PaymentIntentStatusSucceeded) {
				p.l.Warn(
					"status in stripe not match.",
					zap.String("provider_oper_status", *tr.ProviderOperStatus),
				)
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			pi, err := p.GetPaymentIntent(*tr.ProviderOperID)
			if err != nil {
				p.l.Error("Failed get payment intent in stripe.", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if len(pi.Charges.Data) > 1 {
				p.l.Error("Failed len charges payment intent in stripe is > 1.", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if len(pi.Charges.Data) == 0 {
				p.l.Error("Failed len charges payment intent in stripe is 0.", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
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
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}

			status := string(stripe.PaymentIntentStatusCanceled)
			tr.ProviderOperStatus = &status
			if err := tx.Save(&tr); err != nil {
				p.l.Error("Failed save transaction. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if err := tx.Commit(); err != nil {
				p.l.Error("Failed tx commit. ", zap.Error(err))
				return
			}
			b, err := json.Marshal(&strategies.MessageUpdateTransaction{
				ClientID:      m.ClientID,
				TransactionID: m.TransactionID,
				Strategy:      m.Strategy,
				Status:        m.Status,
			})
			if err != nil {
				p.l.Error("Failed json marshal for publish update transaction.", zap.Error(err))
				return
			}
			if _, err := p.pb.Topic(strategies.UPDATE_TRANSACTION_SUBJECT).Publish(context.Background(), &pubsub.Message{
				Data: b,
			}).Get(context.Background()); err != nil {
				p.l.Error("Failed publish update transaction.", zap.Error(err))
			}
			*okAck = true
		default:
			p.l.Warn("Not processed command in message of stripe in nats.")
			if err := tx.Rollback(); err != nil {
				p.l.Error("Failed tx rollback. ", zap.Error(err))
			}
		}
	}
}
