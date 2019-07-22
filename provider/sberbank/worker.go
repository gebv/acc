package sberbank

import (
	"context"
	"log"
	"time"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/engine/strategies"
	"github.com/gebv/acca/ffsm"
	"go.uber.org/zap"
)

type Command string

const (
	SUBJECT                = "provider_sberbank_subject"
	AuthTransfer   Command = "auth_transfer"
	ReverseForHold Command = "reverse_for_hold"
)

type MessageToSberbank struct {
	Command       Command
	TransactionID int64
	Strategy      string
	Status        engine.TransactionStatus
}

func (p *Provider) NatsHandler() func(m *MessageToSberbank) {
	return func(m *MessageToSberbank) {
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
			if tr.Status != engine.DRAFT_TX {
				p.l.Warn(
					"Transaction status not draft.",
					zap.Int64("tr_id", tr.TransactionID),
					zap.String("status", string(tr.Status)),
				)
				return
			}
			var amount int64 // TODO amount из транзакции после добавления
			extOrderID, urlForm, err := p.AuthTransfer(
				amount,
				TransferInformation{
					ReturnURL:   "",
					FailURL:     "",
					Description: "",
					Email:       "",
				},
				true,
			)
			if err != nil {
				p.l.Error("Failed auth transfer in sberbank.", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			// TODO в транзакцию добавить значения ссылки так как по ней требуется оплата.
			log.Println("URL FROM PAY IN SBERANK: ", urlForm)
			tr.ProviderOperID = &extOrderID
			status := CREATED
			tr.ProviderOperStatus = &status
			if err := tx.Save(&tr); err != nil {
				p.l.Error("Failed reload transaction. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if err := tx.Commit(); err != nil {
				p.l.Error("Failed tx commit. ", zap.Error(err))
			}
			tx, err := p.db.Begin()
			if err != nil {
				p.l.Error("Failed begin transaction DB.", zap.Error(err))
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			ctx = strategies.SetNatsToContext(ctx, p.nc)
			ctx = strategies.SetTXContext(ctx, tx)
			if name := strategies.ExistTrName(m.Strategy); name != strategies.UNDEFINED_TR {
				if str := strategies.GetTransactionStrategy(name); str != nil {
					err := str.Dispatch(ctx, ffsm.State(m.Status), m.TransactionID)
					if err != nil {
						p.l.Error("Failed dispatch transaction strategy. ", zap.Error(err))
						if err := tx.Rollback(); err != nil {
							p.l.Error("Failed tx rollback. ", zap.Error(err))
						}
						return
					}
					if err := tx.Commit(); err != nil {
						p.l.Error("Failed tx commit. ", zap.Error(err))
					}
				}
			}
			if err := tx.Rollback(); err != nil {
				p.l.Error("Failed tx rollback. ", zap.Error(err))
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
			if tr.Status != engine.HOLD_TX {
				p.l.Warn(
					"Transaction status not hold.",
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
			var amount int64 // TODO amount из транзакции после добавления
			err := p.ReverseForHold(
				*tr.ProviderOperID,
				amount,
			)
			if err != nil {
				p.l.Error("Failed reverse for hold in sberbank.", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if err := tx.Commit(); err != nil {
				p.l.Error("Failed tx commit. ", zap.Error(err))
			}
			tx, err := p.db.Begin()
			if err != nil {
				p.l.Error("Failed begin transaction DB.", zap.Error(err))
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			ctx = strategies.SetNatsToContext(ctx, p.nc)
			ctx = strategies.SetTXContext(ctx, tx)
			if name := strategies.ExistTrName(m.Strategy); name != strategies.UNDEFINED_TR {
				if str := strategies.GetTransactionStrategy(name); str != nil {
					err := str.Dispatch(ctx, ffsm.State(m.Status), m.TransactionID)
					if err != nil {
						p.l.Error("Failed dispatch transaction strategy. ", zap.Error(err))
						if err := tx.Rollback(); err != nil {
							p.l.Error("Failed tx rollback. ", zap.Error(err))
						}
						return
					}
					if err := tx.Commit(); err != nil {
						p.l.Error("Failed tx commit. ", zap.Error(err))
					}
				}
			}
			if err := tx.Rollback(); err != nil {
				p.l.Error("Failed tx rollback. ", zap.Error(err))
			}
		default:
			p.l.Warn("Not processed command in message of sberbank in nats.")
		}
	}
}
