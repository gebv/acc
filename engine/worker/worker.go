package worker

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
	"gopkg.in/reform.v1"

	"github.com/gebv/acca/engine/strategies"
	"github.com/gebv/acca/ffsm"
	"github.com/gebv/acca/provider/moedelo"
	"github.com/gebv/acca/provider/sberbank"
	"github.com/gebv/acca/provider/stripe"
)

func Run(
	c context.Context,
	fs *firestore.Client,
	db *reform.DB,
	providerSber *sberbank.Provider,
	providerMoeDelo *moedelo.Provider,
	providerStripe *stripe.Provider,
) {
	go func() {
		l := zap.L().Named("pb_sub_" + strategies.UPDATE_INVOICE_SUBJECT)
		tm := time.NewTicker(time.Second)
		defer tm.Stop()
		for {
			select {
			case <-tm.C:
				fsCtx, fsCancel := context.WithCancel(context.Background())
				if docs, err := fs.Collection("messages").Where("type", "==", strategies.UPDATE_INVOICE_SUBJECT).Where("status_msg", "==", "new").OrderBy("created_at", firestore.Asc).Limit(1).Documents(fsCtx).GetAll(); err == nil {
					for _, doc := range docs {
						if _, err := doc.Ref.Update(fsCtx, []firestore.Update{
							{
								Path:  "status_msg",
								Value: "in_progress",
							},
						}); err != nil {
							l.Error("Failed update message", zap.Error(err))
							break
						}
						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						if err := fs.RunTransaction(ctx, func(ctx context.Context, fsTx *firestore.Transaction) error {
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
									l.Error("Failed update status message", zap.Error(err))
								}
							}()
							var m strategies.MessageUpdateInvoice
							ctx, span := trace.StartSpan(ctx, "async.fromQueue.UpdateInvoice")
							defer span.End()
							if err := doc.DataTo(&m); err != nil {
								return errors.Wrap(err, "Failed data to message.")
							}
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
								return errors.Wrap(err, "Failed begin transaction DB.")
							}
							ctx = strategies.SetFirestoreClientToContext(ctx, fs)
							ctx = strategies.SetFirestoreTxToContext(ctx, fsTx)
							ctx = strategies.SetTXContext(ctx, tx)
							if name := strategies.ExistInvName(m.Strategy); name != strategies.UNDEFINED_INV {
								if str := strategies.GetInvoiceStrategy(name); str != nil {
									err := str.Dispatch(ctx, ffsm.State(m.Status), m.InvoiceID)
									if err != nil {
										if err := tx.Rollback(); err != nil {
											l.Error("Failed tx rollback. ", zap.Error(err))
										}
										return errors.Wrap(err, "Failed dispatch invoice strategy.")
									}
									if err := tx.Commit(); err != nil {
										return errors.Wrap(err, "Failed tx commit.")
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
									okProcessing = true
									return nil
								}
							}
							if err := tx.Rollback(); err != nil {
								l.Error("Failed tx rollback. ", zap.Error(err))
							}
							return nil
						}); err != nil {
							l.Error("Failed run transaction", zap.Error(err))
						}
						cancel()
					}
				} else {
					l.Error("Failed get messages. ", zap.Error(err))
				}
				fsCancel()
			case <-c.Done():
				return
			}
		}
	}()

	go func() {
		l := zap.L().Named("pb_sub_" + strategies.UPDATE_TRANSACTION_SUBJECT)
		tm := time.NewTicker(time.Second)
		defer tm.Stop()
		for {
			select {
			case <-tm.C:
				fsCtx, fsCancel := context.WithCancel(context.Background())
				if docs, err := fs.Collection("messages").Where("type", "==", strategies.UPDATE_TRANSACTION_SUBJECT).Where("status_msg", "==", "new").OrderBy("created_at", firestore.Asc).Limit(1).Documents(fsCtx).GetAll(); err == nil {
					for _, doc := range docs {
						if _, err := doc.Ref.Update(fsCtx, []firestore.Update{
							{
								Path:  "status_msg",
								Value: "in_progress",
							},
						}); err != nil {
							l.Error("Failed update message", zap.Error(err))
							break
						}
						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						if err := fs.RunTransaction(ctx, func(ctx context.Context, fsTx *firestore.Transaction) error {
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
									l.Error("Failed update status message", zap.Error(err))
								}
							}()
							var m strategies.MessageUpdateTransaction
							ctx, span := trace.StartSpan(ctx, "async.fromQueue.UpdateTransaction")
							defer span.End()
							if err := doc.DataTo(&m); err != nil {
								return errors.Wrap(err, "Failed data to message.")
							}
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
								return errors.Wrap(err, "Failed begin transaction DB.")
							}
							ctx = strategies.SetFirestoreClientToContext(ctx, fs)
							ctx = strategies.SetFirestoreTxToContext(ctx, fsTx)
							ctx = strategies.SetTXContext(ctx, tx)
							if name := strategies.ExistTrName(m.Strategy); name != strategies.UNDEFINED_TR {
								if str := strategies.GetTransactionStrategy(name); str != nil {
									err := str.Dispatch(ctx, ffsm.State(m.Status), m.TransactionID)
									if err != nil {
										if err := tx.Rollback(); err != nil {
											l.Error("Failed tx rollback. ", zap.Error(err))
										}
										return errors.Wrap(err, "Failed dispatch invoice strategy.")
									}
									if err := tx.Commit(); err != nil {
										return errors.Wrap(err, "Failed tx commit.")
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
									okProcessing = true
									return nil
								}
							}
							if err := tx.Rollback(); err != nil {
								l.Error("Failed tx rollback. ", zap.Error(err))
							}
							return nil
						}); err != nil {
							l.Error("Failed run transaction", zap.Error(err))
						}
						cancel()
					}
				} else {
					l.Error("Failed get messages. ", zap.Error(err))
				}
				fsCancel()
			case <-c.Done():
				return
			}
		}
	}()

	if providerSber != nil {
		//go func() {
		//	l := zap.L().Named("pb_sub_" + sberbank.SUBJECT)
		//	if err := pb.Subscription(sberbank.SUBJECT).Receive(context.Background(), providerSber.WorkerHandler()); err != nil && status.Code(err) != codes.Canceled {
		//		l.Error("Failed pubsub Receive. ", zap.Error(err))
		//	}
		//}()
	}

	if providerStripe != nil {
		go func() {
			tm := time.NewTicker(time.Second)
			defer tm.Stop()
			for {
				select {
				case <-tm.C:
					fsCtx, fsCancel := context.WithCancel(context.Background())
					providerStripe.WorkerHandler(fsCtx)
					fsCancel()
				case <-c.Done():
					return
				}
			}
			//if err := pb.Subscription(stripe.SUBJECT).Receive(context.Background(), ); err != nil && status.Code(err) != codes.Canceled {
			//	l.Error("Failed pubsub Receive. ", zap.Error(err))
			//}
		}()
	}

	if providerMoeDelo != nil {
		//go func() {
		//	l := zap.L().Named("pb_sub_" + moedelo.SUBJECT)
		//	if err := pb.Subscription(moedelo.SUBJECT).Receive(context.Background(), providerMoeDelo.WorkerHandler()); err != nil && status.Code(err) != codes.Canceled {
		//		l.Error("Failed pubsub Receive. ", zap.Error(err))
		//	}
		//}()
	}
}
