package worker

import (
	"context"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"go.opencensus.io/trace"
	"gopkg.in/reform.v1"

	"github.com/gebv/acca/engine/strategies"
	"github.com/gebv/acca/ffsm"
	"github.com/gebv/acca/provider/moedelo"
	"github.com/gebv/acca/provider/sberbank"
	"github.com/gebv/acca/services/updater"
)

func SubToNATS(
	nc *nats.EncodedConn,
	db *reform.DB,
	providerSber *sberbank.Provider,
	providerMoeDelo *moedelo.Provider,
) {
	nc.QueueSubscribe(strategies.UPDATE_INVOICE_SUBJECT, "queue", func(m *strategies.MessageUpdateInvoice) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ctx, span := trace.StartSpan(ctx, "UpdateInvoice")
		defer span.End()
		var clientID int64
		if m.ClientID != nil {
			clientID = *m.ClientID
		}
		span.AddAttributes(
			trace.Int64Attribute("client_id", clientID),
			trace.Int64Attribute("invoice_id", m.InvoiceID),
			trace.StringAttribute("strategy", m.Strategy),
			trace.StringAttribute("status", string(m.Status)),
		)
		tx, err := db.Begin()
		if err != nil {
			log.Println("Failed begin transaction DB.")
			return
		}
		ctx = strategies.SetNatsToContext(ctx, nc)
		ctx = strategies.SetTXContext(ctx, tx)
		if name := strategies.ExistInvName(m.Strategy); name != strategies.UNDEFINED_INV {
			if str := strategies.GetInvoiceStrategy(name); str != nil {
				err := str.Dispatch(ctx, ffsm.State(m.Status), m.InvoiceID)
				if err != nil {
					log.Println("Failed dispatch invoice strategy.", err)
					if err := tx.Rollback(); err != nil {
						log.Println("Failed tx rollback. ", err)
					}
					return
				}
				if err := tx.Commit(); err != nil {
					log.Println("Failed tx commit. ", err)
					return
				}
				if err := nc.Publish(updater.SubjectFromInvoice(m.ClientID, m.InvoiceID), &updater.Update{
					UpdatedInvoice: &updater.UpdatedInvoice{
						InvoiceID: m.InvoiceID,
						Status:    m.Status,
					},
				}); err != nil {
					log.Println("Failed publish invoice. ", err)
					return
				}
				return
			}
		}
		if err := tx.Rollback(); err != nil {
			log.Println("Failed tx rollback. ", err)
		}
	})
	nc.QueueSubscribe(strategies.UPDATE_TRANSACTION_SUBJECT, "queue", func(m *strategies.MessageUpdateTransaction) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ctx, span := trace.StartSpan(ctx, "UpdateTransaction")
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
		)
		tx, err := db.Begin()
		if err != nil {
			log.Println("Failed begin transaction DB.")
			return
		}
		ctx = strategies.SetNatsToContext(ctx, nc)
		ctx = strategies.SetTXContext(ctx, tx)
		if name := strategies.ExistTrName(m.Strategy); name != strategies.UNDEFINED_TR {
			if str := strategies.GetTransactionStrategy(name); str != nil {
				err := str.Dispatch(ctx, ffsm.State(m.Status), m.TransactionID)
				if err != nil {
					log.Println("Failed dispatch transaction strategy. ", err)
					if err := tx.Rollback(); err != nil {
						log.Println("Failed tx rollback. ", err)
					}
					return
				}
				if err := tx.Commit(); err != nil {
					log.Println("Failed tx commit. ", err)
					return
				}
				if err := nc.Publish(updater.SubjectFromTransaction(m.ClientID, m.TransactionID), &updater.Update{
					UpdatedTransaction: &updater.UpdatedTransaction{
						TransactionID: m.TransactionID,
						Status:        m.Status,
					},
				}); err != nil {
					log.Println("Failed publish transaction. ", err)
					return
				}
				return
			}
		}
		if err := tx.Rollback(); err != nil {
			log.Println("Failed tx rollback. ", err)
		}
	})
	nc.QueueSubscribe(sberbank.SUBJECT, "queue", providerSber.NatsHandler())
	nc.QueueSubscribe(moedelo.SUBJECT, "queue", providerMoeDelo.NatsHandler())
}
