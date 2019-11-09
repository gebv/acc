package sberbank

import (
	"context"
	"encoding/json"

	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/engine/strategies"
)

type Command string

const (
	SUBJECT                = "provider_sberbank_subject"
	AuthTransfer   Command = "auth_transfer"
	ReverseForHold Command = "reverse_for_hold"
	Refund         Command = "refund"
)

type MessageToSberbank struct {
	Command       Command
	ClientID      *int64
	TransactionID int64
	Strategy      string
	Status        engine.TransactionStatus
}

func (p *Provider) NatsHandler() func(m *MessageToSberbank) {
	return func(m *MessageToSberbank) {
		_, span := trace.StartSpan(context.Background(), "async.fromQueue.ProviderSberbank")
		defer span.End()
		var clientID int64
		if m.ClientID != nil {
			clientID = *m.ClientID
		}
		span.AddAttributes(
			trace.Int64Attribute("client_id", clientID),
			trace.Int64Attribute("transaction_id", m.TransactionID),
			trace.StringAttribute("strategy", m.Strategy),
			trace.StringAttribute("status", string(m.Status)),
			trace.StringAttribute("command", string(m.Command)),
		)
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

			extOrderID, urlForm, err := p.AuthTransfer(
				tr.Amount,
				TransferInformation{
					ReturnURL:   "localhost?result=success" + "&callback=" + meta["callback"],
					FailURL:     "localhost?result=false" + "&callback=" + meta["callback"],
					Description: meta["description"],
					Email:       meta["email"],
				},
				hold,
			)
			if err != nil {
				p.l.Error("Failed auth transfer in sberbank.", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			tr.ProviderOperID = &extOrderID
			status := CREATED
			tr.ProviderOperStatus = &status
			tr.ProviderOperUrl = &urlForm
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
				p.l.Warn("External order ID in sberbank is nil.")
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if tr.ProviderOperStatus == nil {
				p.l.Warn("status in sberbank is nil.")
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if *tr.ProviderOperStatus != APPROVED {
				p.l.Warn(
					"status in sberbank not match.",
					zap.String("provider_oper_status", *tr.ProviderOperStatus),
				)
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			err := p.ReverseForHold(
				*tr.ProviderOperID,
				tr.Amount,
			)
			if err != nil {
				p.l.Error("Failed reverse for hold in sberbank.", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			status := REVERSED
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
				p.l.Warn("External order ID in sberbank is nil.")
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if tr.ProviderOperStatus == nil {
				p.l.Warn("status in sberbank is nil.")
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if *tr.ProviderOperStatus != DEPOSITED {
				p.l.Warn(
					"status in sberbank not match.",
					zap.String("provider_oper_status", *tr.ProviderOperStatus),
				)
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			err := p.Refund(
				*tr.ProviderOperID,
				tr.Amount,
			)
			if err != nil {
				p.l.Error("Failed reverse for hold in sberbank.", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			*tr.ProviderOperStatus = REFUNDED
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
			p.l.Warn("Not processed command in message of sberbank in nats.")
		}
	}
}
