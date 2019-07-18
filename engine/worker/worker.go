package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/gebv/acca/engine/strategies"
	"github.com/gebv/acca/ffsm"
	"github.com/nats-io/nats.go"
	"gopkg.in/reform.v1"
)

func SubToNATS(nc *nats.Conn, db *reform.DB) {
	nc.QueueSubscribe(strategies.UPDATE_INVOICE_SUBJECT, "queue", func(msg *nats.Msg) {
		var m strategies.MessageUpdateInvoice
		err := json.Unmarshal(msg.Data, &m)
		if err != nil {
			log.Println("Failed unmarshal msg.")
			return
		}
		tx, err := db.Begin()
		if err != nil {
			log.Println("Failed begin transaction DB.")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ctx = strategies.SetNatsToContext(ctx, nc)
		ctx = strategies.SetTXContext(ctx, tx)
		if name := strategies.ExistInvName(m.Strategy); name != strategies.UNDEFINED_INV {
			if str := strategies.GetInvoiceStrategy(name); str != nil {
				err := str.Dispatch(ctx, ffsm.State(m.Status), m.InvoiceID)
				if err != nil {
					log.Println("Failed dispatch invoice strategy.")
					if err := tx.Rollback(); err != nil {
						log.Println("Failed tx rollback. ", err)
					}
					return
				}
				if err := tx.Commit(); err != nil {
					log.Println("Failed tx commit. ", err)
				}
			}
		}
	})
	nc.QueueSubscribe(strategies.UPDATE_TRANSACTION_SUBJECT, "queue", func(msg *nats.Msg) {
		var m strategies.MessageUpdateTransaction
		err := json.Unmarshal(msg.Data, &m)
		if err != nil {
			log.Println("Failed unmarshal msg.")
			return
		}
		tx, err := db.Begin()
		if err != nil {
			log.Println("Failed begin transaction DB.")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
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
				}
			}
		}
	})
}
