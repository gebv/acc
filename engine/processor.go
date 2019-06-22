package engine

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"gopkg.in/reform.v1"
)

type TransactionProcessor struct {
	db        *reform.DB
	wg        sync.WaitGroup
	toProcess chan *transactionProcessorMessage
	l         *zap.Logger
}

type transactionProcessorMessage struct {
	txID          int64
	currentStatus TransactionStatus
	nextStatus    TransactionStatus
	updatedAt     time.Time
}

func (p *TransactionProcessor) runPocessor(ctx context.Context) error {
	defer p.wg.Done()
	var err error
	for m := range p.toProcess {
		err = p.db.InTransaction(func(tx *reform.TX) error {
			currentTx := &Transaction{TransactionID: m.txID}
			if err := tx.Reload(currentTx); err != nil {
				return errors.Wrap(err, "failed find transaction")
			}

			if currentTx.UpdatedAt.UnixNano() != m.updatedAt.UnixNano() {
				return errors.Wrap(err, "transaction is rejected by the processor - not matched updated_at")
			}
			if !currentTx.Status.Match(m.currentStatus) {
				return errors.Wrap(err, "transaction is rejected by the processor - not matched status")
			}
			if !transactionStatusTransitionChart.Allowed(m.currentStatus, m.nextStatus) {
				return errors.Wrap(err, "transaction is rejected by the processor - not allowed transition status")
			}

			return p.process(tx, m)
		})
		if err != nil {
			p.l.Error("failed process", zap.Error(err), zap.Int64("tx_id", m.txID), zap.Time("tx_version_at", m.updatedAt))
			continue
		}
	}
	return nil
}

func (t *TransactionProcessor) process(tx *reform.TX, msg *transactionProcessorMessage) error {
	opers, err := tx.SelectAllFrom((&Operation{}).View(), "WHERE tx_id = $1 ORDER BY oper_id ASC FOR UPDATE", msg.txID)
	if err != nil {
		return errors.Wrap(err, "failed find operations")
	}

	sm := newLowLevelMoneyTransferStrategy()
	for _, ioper := range opers {
		oper := ioper.(*Operation)

		if err := sm.Process(msg.nextStatus, oper); err != nil {
			return errors.Wrapf(err, "failed process operation %d", oper.OperationID)
		}

		// store operation status after process
		if err := tx.UpdateColumns(oper, "updated_at", "status"); err != nil {
			return errors.Wrapf(err, "failed update operation %d after process", oper.OperationID)
		}
	}

	// store changed balances after process
	for accID, balance := range sm.accountBalances {
		if balance == 0 {
			// to skip if the balance didn't change
			continue
		}
		if _, err := tx.Exec(`UPDATE acca.accounts SET balance = $1 WHERE acc_id = $2`, balance, accID); err != nil {
			return errors.Wrapf(err, "failed update balance for account %d", accID)
		}
	}
	for accID, balance := range sm.accountAcceptedBalances {
		if balance == 0 {
			// to skip if the balance didn't change
			continue
		}
		if _, err := tx.Exec(`UPDATE acca.accounts SET balance_accepted = $1 WHERE acc_id = $2`, balance, accID); err != nil {
			return errors.Wrapf(err, "failed update balance_accepted for account %d", accID)
		}
	}

	return nil
}

func (t *TransactionProcessor) Send(txID int64, updatedAt time.Time, currentStatus, nextStatus TransactionStatus) {
	msg := &transactionProcessorMessage{
		txID:          txID,
		updatedAt:     updatedAt,
		currentStatus: currentStatus,
		nextStatus:    nextStatus,
	}
	t.toProcess <- msg
}
