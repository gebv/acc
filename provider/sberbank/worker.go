package sberbank

import (
	"context"
	"encoding/json"
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
	Refund         Command = "refund"
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
					return
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
					return
				}
			}
			if err := tx.Rollback(); err != nil {
				p.l.Error("Failed tx rollback. ", zap.Error(err))
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
					return
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
