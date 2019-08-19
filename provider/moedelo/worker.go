package moedelo

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/engine/strategies"
	"go.uber.org/zap"
)

type Command string

const (
	SUBJECT                = "provider_moedelo_subject"
	CreateBill     Command = "create_bill"
	HoldBill       Command = "hold_bill"
	PayBill        Command = "pay_bill"
	ReverseForHold Command = "reverse_for_hold"
)

type MessageToMoedelo struct {
	Command       Command
	ClientID      *int64
	TransactionID int64
	Strategy      string
	Status        engine.TransactionStatus
}

func (p *Provider) NatsHandler() func(m *MessageToMoedelo) {
	return func(m *MessageToMoedelo) {
		tx, err := p.db.Begin()
		if err != nil {
			p.l.Error("Failed begin transaction DB.", zap.Error(err))
			return
		}
		switch m.Command {
		case CreateBill:
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
			kontragentID, err := strconv.ParseInt(meta["kontragent_id"], 10, 64)
			if err != nil {
				p.l.Error("Failed parse kontragent_id in transaction meta. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			count, err := strconv.ParseFloat(meta["count"], 10)
			if err != nil {
				p.l.Error("Failed parse count in transaction meta. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			price, err := strconv.ParseFloat(meta["price"], 10)
			if err != nil {
				p.l.Error("Failed parse price in transaction meta. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}

			items := []SalesDocumentItemModel{
				{
					Name:    meta["title"],
					Count:   count,
					Unit:    meta["unit"],
					Type:    Service,
					Price:   price,
					NdsType: Nds0,
				},
			}

			billID, urlForm, err := p.CreateBill(
				kontragentID,
				time.Now(),
				items,
			)
			if err != nil {
				p.l.Error("Failed auth transfer in moedelo.", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			extOrderID := strconv.FormatInt(*billID, 10)
			tr.ProviderOperID = &extOrderID
			status := NotPaid.String()
			tr.ProviderOperStatus = &status
			tr.ProviderOperUrl = urlForm
			if err := tx.Save(&tr); err != nil {
				p.l.Error("Failed reload transaction. ", zap.Error(err))
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
		case HoldBill:
			tr := engine.Transaction{TransactionID: m.TransactionID}
			if err := tx.Reload(&tr); err != nil {
				p.l.Error("Failed reload transaction. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			if tr.Status != engine.AUTH_TX {
				p.l.Warn(
					"Transaction status not auth.",
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
			kontragentID, err := strconv.ParseInt(meta["kontragent_id"], 10, 64)
			if err != nil {
				p.l.Error("Failed parse kontragent_id in transaction meta. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			count, err := strconv.ParseFloat(meta["count"], 10)
			if err != nil {
				p.l.Error("Failed parse count in transaction meta. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			price, err := strconv.ParseFloat(meta["price"], 10)
			if err != nil {
				p.l.Error("Failed parse price in transaction meta. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}

			items := []SalesDocumentItemModel{
				{
					Name:    meta["title"],
					Count:   count,
					Unit:    meta["unit"],
					Type:    Service,
					Price:   price,
					NdsType: Nds0,
				},
			}

			billID, err := strconv.ParseInt(*tr.ProviderOperID, 10, 64)
			if err != nil {
				p.l.Error("Failed parse billID in transaction. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			status := PartiallyPaid
			err = p.UpdateBill(
				billID,
				kontragentID,
				time.Now(),
				items,
				&status,
			)
			if err != nil {
				p.l.Error("Failed update bill from auth transfer in moedelo.", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			statusStr := status.String()
			tr.ProviderOperStatus = &statusStr
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
		case PayBill:
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
					"Transaction status not auth.",
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
			kontragentID, err := strconv.ParseInt(meta["kontragent_id"], 10, 64)
			if err != nil {
				p.l.Error("Failed parse kontragent_id in transaction meta. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			count, err := strconv.ParseFloat(meta["count"], 10)
			if err != nil {
				p.l.Error("Failed parse count in transaction meta. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			price, err := strconv.ParseFloat(meta["price"], 10)
			if err != nil {
				p.l.Error("Failed parse price in transaction meta. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}

			items := []SalesDocumentItemModel{
				{
					Name:    meta["title"],
					Count:   count,
					Unit:    meta["unit"],
					Type:    Service,
					Price:   price,
					NdsType: Nds0,
				},
			}

			billID, err := strconv.ParseInt(*tr.ProviderOperID, 10, 64)
			if err != nil {
				p.l.Error("Failed parse billID in transaction. ", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			status := Paid
			err = p.UpdateBill(
				billID,
				kontragentID,
				time.Now(),
				items,
				&status,
			)
			if err != nil {
				p.l.Error("Failed update bill from auth transfer in moedelo.", zap.Error(err))
				if err := tx.Rollback(); err != nil {
					p.l.Error("Failed tx rollback. ", zap.Error(err))
				}
				return
			}
			statusStr := status.String()
			tr.ProviderOperStatus = &statusStr
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
		case ReverseForHold:
			if err := p.nc.Publish(strategies.UPDATE_TRANSACTION_SUBJECT, &strategies.MessageUpdateTransaction{
				ClientID:      m.ClientID,
				TransactionID: m.TransactionID,
				Strategy:      m.Strategy,
				Status:        m.Status,
			}); err != nil {
				p.l.Error("Failed publish update transaction.", zap.Error(err))
			}
		default:
			p.l.Warn("Not processed command in message of moe delo in nats.")
		}
	}
}
