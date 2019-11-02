package stripe

import (
	"encoding/json"

	"github.com/stripe/stripe-go"
	"go.uber.org/zap"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/engine/strategies"
)

type Command string

const (
	SUBJECT                = "provider_stripe_subject"
	AuthTransfer   Command = "auth_transfer"
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

func (p *Provider) NatsHandler() func(m *MessageToStripe) {
	return func(m *MessageToStripe) {
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
				return
			}
			if tr.Meta == nil {
				p.l.Warn(
					"Transaction not set meta.",
					zap.Int64("tr_id", tr.TransactionID),
				)
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
			_ = hold // TODO проверить hold и выставить счет с холдом
			paymentIntent, err := p.PaymentIntent(
				tr.Amount,
				stripe.CurrencyUSD,
				nil,
				nil,
				nil,
			)
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
			if err := p.nc.Publish(strategies.UPDATE_TRANSACTION_SUBJECT, &strategies.MessageUpdateTransaction{
				ClientID:      m.ClientID,
				TransactionID: m.TransactionID,
				Strategy:      m.Strategy,
				Status:        m.Status,
			}); err != nil {
				p.l.Error("Failed publish update transaction.", zap.Error(err))
			}
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
			err := p.ReverseForHold() // TODO поправить вызов
			if err != nil {
				p.l.Error("Failed reverse for hold in stripe.", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			status := "??? REVERSED" // TODO поправить статус платежа
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
			if err := p.nc.Publish(strategies.UPDATE_TRANSACTION_SUBJECT, &strategies.MessageUpdateTransaction{
				ClientID:      m.ClientID,
				TransactionID: m.TransactionID,
				Strategy:      m.Strategy,
				Status:        m.Status,
			}); err != nil {
				p.l.Error("Failed publish update transaction.", zap.Error(err))
			}
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
			err := p.Refund() // TODO поправить вызов
			if err != nil {
				p.l.Error("Failed reverse for hold in stripe.", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			*tr.ProviderOperStatus = "??? REFUNDED" // TODO поправить статус платежа
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
			if err := p.nc.Publish(strategies.UPDATE_TRANSACTION_SUBJECT, &strategies.MessageUpdateTransaction{
				ClientID:      m.ClientID,
				TransactionID: m.TransactionID,
				Strategy:      m.Strategy,
				Status:        m.Status,
			}); err != nil {
				p.l.Error("Failed publish update transaction.", zap.Error(err))
			}
		default:
			p.l.Warn("Not processed command in message of stripe in nats.")
		}
	}
}
