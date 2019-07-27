package moedelo

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/engine/strategies"
	"github.com/gebv/acca/ffsm"
	"github.com/gebv/acca/provider"
	"go.uber.org/zap"
	"gopkg.in/reform.v1"
)

func (p *Provider) RunCheckStatusListener(ctx context.Context) {
	_l := p.l.Named("moe_delo_check_status_listener")
	_l.Info("Started")
	store := &provider.Store{DB: p.db}
	tm, err := store.GetLastTimeFromPaymentSystemName(MOEDELO)
	if err != nil {
		_l.Panic("Failed get last time from noncash", zap.Error(err))
	}
	afterDate := *tm
	var nextDate time.Time
	var skip bool
	var push bool
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for ctx.Err() == nil {
		<-ticker.C
		_l.Debug("Get data in MD")
		nextDate = time.Now()
		listBill, err := p.GetListBills(&afterDate, &nextDate)
		if err != nil {
			if err == ErrProviderNotSet {
				_l.Warn("Failed get list bills from moe delo", zap.Error(err))
				return
			}
			_l.Warn("Failed get list bills from moe delo", zap.Error(err))
			continue
		}
		_l.Debug("Get list bills", zap.Int("len", len(listBill.ResourceList)))
		skip = false
		for _, bill := range listBill.ResourceList {
			if skip {
				break
			}
			_l.Debug("Check Bill", zap.Int64("bill_id", bill.ID), zap.Int("bill_status", int(bill.Status)))
			orderID := strconv.FormatInt(bill.ID, 10)
			var oldStatus BillStatus
			billData, err := store.GetByOrderID(orderID, MOEDELO)
			switch err {
			case reform.ErrNoRows:
				err = store.NewOrderWithExtUpdate(orderID, MOEDELO, strconv.Itoa(int(bill.Status)), nextDate)
				if err != nil {
					_l.Warn("Failed set status in bill from moe delo", zap.Error(err))
					skip = true
					continue
				}
			case nil:
				if i, err := strconv.Atoi(billData.RawOrderStatus); err == nil {
					oldStatus = BillStatus(i)
				} else {
					_l.Warn("Failed convert raw_order_status from moe delo", zap.Error(err))
					skip = true
					continue
				}
			default:
				_l.Warn("Failed get ext order status in bill from moe delo", zap.Error(err))
				skip = true
				continue
			}
			push = false
			var status ffsm.State
			switch bill.Status {
			case Paid:
				if oldStatus != Paid {
					status = ffsm.State(engine.ACCEPTED_TX)
					push = true
				}
			case PartiallyPaid:
				if oldStatus != PartiallyPaid {
					status = ffsm.State(engine.HOLD_TX)
					push = true
				}
			}

			if !skip && push {
				var tr engine.Transaction
				err := p.db.SelectOneTo(&tr, "WHERE provider = $1 AND provider_oper_id = $2", MOEDELO, strconv.FormatInt(bill.ID, 10))
				if err != nil {
					if err == reform.ErrNoRows {
						_l.Warn("Not found related transactions by bill ID", zap.Int64("bill_id", bill.ID))
						continue
					}
					_l.Warn("Failed find related transactions by bill ID ", zap.Int64("bill_id", bill.ID), zap.Error(err))
					skip = true
					continue
				}
				statusStr := bill.Status.String()
				tr.ProviderOperStatus = &statusStr
				if err := p.db.Save(&tr); err != nil {
					_l.Warn("Failed save transaction. ", zap.Error(err))
					skip = true
					continue
				}
				tx, err := p.db.Begin()
				if err != nil {
					_l.Error("Failed begin transaction DB.", zap.Error(err))
					skip = true
					continue
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				ctx = strategies.SetNatsToContext(ctx, p.nc)
				ctx = strategies.SetTXContext(ctx, tx)
				if name := strategies.ExistTrName(tr.Strategy); name != strategies.UNDEFINED_TR {
					if str := strategies.GetTransactionStrategy(name); str != nil {
						err := str.Dispatch(ctx, status, tr.TransactionID)
						if err != nil {
							_l.Error("Failed dispatch transaction strategy. ", zap.Error(err))
							if err := tx.Rollback(); err != nil {
								_l.Error("Failed tx rollback. ", zap.Error(err))
							}
							skip = true
							continue
						}
						if err := tx.Commit(); err != nil {
							_l.Error("Failed tx commit. ", zap.Error(err))
							skip = true
							continue
						}
						if err := store.SetStatus(orderID, MOEDELO, strconv.Itoa(int(bill.Status))); err != nil {
							_l.Error("Failed set status to external transaction. ", zap.Error(err))
						}
						continue
					}
				}
				if err := tx.Rollback(); err != nil {
					_l.Error("Failed tx rollback. ", zap.Error(err))
					skip = true
					continue
				}
			}
		}
		if skip {
			continue
		}
		afterDate = nextDate
	}
	_l.Info("Stoped")
}

func newPaymentOrderID(kontragentID, billID int64) string {
	b := make([]byte, 3)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(err)
	}
	return fmt.Sprintf(
		"app-md-%d-%d",
		kontragentID,
		billID)
}
