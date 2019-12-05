package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"cloud.google.com/go/pubsub"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/gebv/acca/engine/strategies"
	"github.com/gebv/acca/ffsm"
	"github.com/gebv/acca/provider/moedelo"
	"github.com/gebv/acca/provider/sberbank"
	"github.com/gebv/acca/provider/stripe"
)

func SubToPB(
	pb *pubsub.Client,
	db *reform.DB,
	providerSber *sberbank.Provider,
	providerMoeDelo *moedelo.Provider,
	providerStripe *stripe.Provider,
) {
	go func() {
		l := zap.L().Named("pb_sub_" + strategies.UPDATE_INVOICE_SUBJECT)
		if err := pb.Subscription(strategies.UPDATE_INVOICE_SUBJECT).Receive(context.Background(), func(ctx context.Context, pbMsg *pubsub.Message) {
			var m strategies.MessageUpdateInvoice
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
				l.Error("Failed unmarshal pubsub message.", zap.Error(err))
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			ctx, span := trace.StartSpan(ctx, "async.fromQueue.UpdateInvoice")
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
			ctx = strategies.SetPubSubToContext(ctx, pb)
			ctx = strategies.SetTXContext(ctx, tx)
			if name := strategies.ExistInvName(m.Strategy); name != strategies.UNDEFINED_INV {
				if str := strategies.GetInvoiceStrategy(name); str != nil {
					err := str.Dispatch(ctx, ffsm.State(m.Status), m.InvoiceID)
					if err != nil {
						log.Println("Failed dispatch invoice strategy.", err)
						if err := tx.Rollback(); err != nil {
							log.Println("Failed tx rollback. ", err)
						}
						okAck = nil
						return
					}
					if err := tx.Commit(); err != nil {
						log.Println("Failed tx commit. ", err)
						return
					}
					// TODO поправить отправку в GetUpdates
					//msg := &updater.Update{
					//	UpdatedInvoice: &updater.UpdatedInvoice{
					//		InvoiceID: m.InvoiceID,
					//		Status:    m.Status,
					//	},
					//}
					//b, err := json.Marshal(msg)
					//if err != nil {
					//	l.Error("Failed json marshal", zap.Error(err))
					//	return
					//}
					//_, err = pb.Topic(updater.SubjectFromInvoice(m.ClientID, m.InvoiceID)).Publish(context.Background(), &pubsub.Message{
					//	Data: b,
					//}).Get(context.Background())
					//if err != nil {
					//	l.Error("Failed publish package.", zap.Error(err))
					//	return
					//}
					*okAck = true
					return
				}
			}
			if err := tx.Rollback(); err != nil {
				l.Error("Failed tx rollback. ", zap.Error(err))
			}
		}); err != nil && status.Code(err) != codes.Canceled {
			l.Error("Failed pubsub Receive. ", zap.Error(err))
		}
	}()

	go func() {
		l := zap.L().Named("pb_sub_" + strategies.UPDATE_TRANSACTION_SUBJECT)
		if err := pb.Subscription(strategies.UPDATE_TRANSACTION_SUBJECT).Receive(context.Background(), func(ctx context.Context, pbMsg *pubsub.Message) {
			var m strategies.MessageUpdateTransaction
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
				l.Error("Failed unmarshal pubsub message.", zap.Error(err))
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			ctx, span := trace.StartSpan(ctx, "async.fromQueue.UpdateTransaction")
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
			ctx = strategies.SetPubSubToContext(ctx, pb)
			ctx = strategies.SetTXContext(ctx, tx)
			if name := strategies.ExistTrName(m.Strategy); name != strategies.UNDEFINED_TR {
				if str := strategies.GetTransactionStrategy(name); str != nil {
					err := str.Dispatch(ctx, ffsm.State(m.Status), m.TransactionID)
					if err != nil {
						log.Println("Failed dispatch transaction strategy. ", err)
						if err := tx.Rollback(); err != nil {
							log.Println("Failed tx rollback. ", err)
						}
						okAck = nil
						return
					}
					if err := tx.Commit(); err != nil {
						log.Println("Failed tx commit. ", err)
						return
					}
					// TODO поправить отправку в GetUpdates
					//if err := nc.Publish(updater.SubjectFromTransaction(m.ClientID, m.TransactionID), &updater.Update{
					//	UpdatedTransaction: &updater.UpdatedTransaction{
					//		TransactionID: m.TransactionID,
					//		Status:        m.Status,
					//	},
					//}); err != nil {
					//	log.Println("Failed publish transaction. ", err)
					//	return
					//}
					*okAck = true
					return
				}
			}
			if err := tx.Rollback(); err != nil {
				l.Error("Failed tx rollback. ", zap.Error(err))
			}
		}); err != nil && status.Code(err) != codes.Canceled {
			l.Error("Failed pubsub Receive. ", zap.Error(err))
		}
	}()

	if providerSber != nil {
		go func() {
			l := zap.L().Named("pb_sub_" + sberbank.SUBJECT)
			if err := pb.Subscription(sberbank.SUBJECT).Receive(context.Background(), providerSber.PubSubHandler()); err != nil && status.Code(err) != codes.Canceled {
				l.Error("Failed pubsub Receive. ", zap.Error(err))
			}
		}()
	}

	if providerStripe != nil {
		go func() {
			l := zap.L().Named("pb_sub_" + stripe.SUBJECT)
			if err := pb.Subscription(stripe.SUBJECT).Receive(context.Background(), providerStripe.PubSubHandler()); err != nil && status.Code(err) != codes.Canceled {
				l.Error("Failed pubsub Receive. ", zap.Error(err))
			}
		}()
	}

	if providerMoeDelo != nil {
		go func() {
			l := zap.L().Named("pb_sub_" + moedelo.SUBJECT)
			if err := pb.Subscription(moedelo.SUBJECT).Receive(context.Background(), providerMoeDelo.PubSubHandler()); err != nil && status.Code(err) != codes.Canceled {
				l.Error("Failed pubsub Receive. ", zap.Error(err))
			}
		}()
	}
}
