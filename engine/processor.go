package engine

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"gopkg.in/reform.v1"
)

const (
	toProcessCap = 1024
)

func NewTransactionProcessor(db *reform.DB) *transactionProcessor {
	p := &transactionProcessor{
		db:        db,
		toProcess: make(chan *processorCommand, toProcessCap),
		l:         zap.L().Named("tx_processor"),
		// TODO: add prometheus metrics
	}
	p.l.Info("Started.")
	p.wg.Add(1)
	go p.runPocessor()
	return p
}

type transactionProcessor struct {
	db        *reform.DB
	wg        sync.WaitGroup
	toProcess chan *processorCommand
	l         *zap.Logger
}

type processorCommand struct {
	txID          int64
	currentStatus TransactionStatus
	nextStatus    TransactionStatus
	updatedAt     time.Time

	// TODO: расширить модель и добавить
	// - смена статуса для транзакции
	// - проверка закрытия всех транзакций в инвйосе и смена статуса инвойса
}

func (p *transactionProcessor) runPocessor() error {
	defer p.wg.Done()
	var err error
	for cmd := range p.toProcess {
		err = p.db.InTransaction(func(tx *reform.TX) error {
			currentTx := &Transaction{TransactionID: cmd.txID}
			if err := tx.Reload(currentTx); err != nil {
				return errors.Wrap(err, "failed find transaction")
			}

			if currentTx.UpdatedAt.UnixNano() != cmd.updatedAt.UnixNano() {
				return errors.New("transaction is rejected by the processor - not matched updated_at")
			}
			if !currentTx.Status.Match(cmd.currentStatus) {
				return errors.New("transaction is rejected by the processor - not matched status")
			}
			if !transactionStatusTransitionChart.Allowed(cmd.currentStatus, cmd.nextStatus) {
				return errors.New("transaction is rejected by the processor - not allowed transition status")
			}

			// TODO: в зависимости от провайдера процедура
			// - Сбербанк, в случае статуса AUTH авторизация операции в сбербанке (получение OperID, OperStatus)

			if err := p.processingOperations(tx, cmd); err != nil {
				return errors.Wrap(err, "failed process")
			}

			currentTx.Status = cmd.nextStatus
			if err := tx.UpdateColumns(currentTx, "updated_at", "status"); err != nil {
				return errors.Wrap(err, "failed update transaction")
			}

			return nil
		})
		if err != nil {
			p.l.Error("failed process transaction", zap.Error(err), zap.Int64("tx_id", cmd.txID), zap.Time("tx_version_at", cmd.updatedAt))
			continue
		}
	}
	return nil
}

func (t *transactionProcessor) Stop() {
	close(t.toProcess)
	t.wg.Wait()
	t.l.Info("Stopped.")
}

// обработка операций в транзакции
func (t *transactionProcessor) processingOperations(tx *reform.TX, msg *processorCommand) error {
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

// AuthInvoice авторизация счета.
// Счет
func (t *transactionProcessor) AuthInvoice(invoiceID int64) error {
	return t.db.InTransaction(func(tx *reform.TX) error {
		authInvoice := &Invoice{}
		if err := tx.SelectOneTo(authInvoice, "WHERE invoice_id = $1", invoiceID); err != nil {
			return errors.Wrap(err, "failed find invoice by ID")
		}

		if !transitionsStatusesOfInvoice.Allowed(authInvoice.Status, AUTH_I) {
			return errors.New("not allowed transition status to AUTH for invoice")
		}

		txs, err := tx.SelectAllFrom((&Transaction{}).View(), "WHERE invoice_id = $1 ORDER BY invoice_id ASC", invoiceID)
		if err != nil {
			return errors.Wrap(err, "failed find transactions by invoice")
		}
		for _, _tx := range txs {
			tx := _tx.(*Transaction)
			if !transactionStatusTransitionChart.Allowed(tx.Status, AUTH_TX) {
				return errors.New("not allowed transition status to AUTH for transaction")
			}
			if err := t.Process(tx.TransactionID, tx.UpdatedAt, tx.Status, AUTH_TX); err != nil {
				return errors.Wrap(err, "failed send to processor of transactions")
			}
		}
		return nil
	})
}

// AcceptInvoice подтверждение счета.
func (t *transactionProcessor) AcceptInvoice(invoiceID int64) error {
	return t.db.InTransaction(func(tx *reform.TX) error {
		authInvoice := &Invoice{}
		if err := tx.SelectOneTo(authInvoice, "WHERE invoice_id = $1", invoiceID); err != nil {
			return errors.Wrap(err, "failed find invoice by ID")
		}

		if !transitionsStatusesOfInvoice.Allowed(authInvoice.Status, ACCEPTED_I) {
			return errors.New("not allowed transition status to ACCEPTED for invoice")
		}

		txs, err := tx.SelectAllFrom((&Transaction{}).View(), "WHERE invoice_id = $1 ORDER BY invoice_id ASC", invoiceID)
		if err != nil {
			return errors.Wrap(err, "failed find transactions by invoice")
		}
		for _, _tx := range txs {
			tx := _tx.(*Transaction)
			if !transactionStatusTransitionChart.Allowed(tx.Status, ACCEPTED_TX) {
				return errors.New("not allowed transition status to ACCEPTED for transaction")
			}
			if err := t.Process(tx.TransactionID, tx.UpdatedAt, tx.Status, ACCEPTED_TX); err != nil {
				return errors.Wrap(err, "failed send to processor of transactions")
			}
		}
		return nil
	})
}

// RejectInvoice отмена счета.
func (t *transactionProcessor) RejectInvoice(invoiceID int64) error {
	return t.db.InTransaction(func(tx *reform.TX) error {
		authInvoice := &Invoice{}
		if err := tx.SelectOneTo(authInvoice, "WHERE invoice_id = $1", invoiceID); err != nil {
			return errors.Wrap(err, "failed find invoice by ID")
		}

		if !transitionsStatusesOfInvoice.Allowed(authInvoice.Status, REJECTED_I) {
			return errors.New("not allowed transition status to REJECTED for invoice")
		}

		txs, err := tx.SelectAllFrom((&Transaction{}).View(), "WHERE invoice_id = $1 ORDER BY invoice_id ASC", invoiceID)
		if err != nil {
			return errors.Wrap(err, "failed find transactions by invoice")
		}
		for _, _tx := range txs {
			tx := _tx.(*Transaction)
			if !transactionStatusTransitionChart.Allowed(tx.Status, REJECTED_TX) {
				return errors.New("not allowed transition status to REJECTED for transaction")
			}
			if err := t.Process(tx.TransactionID, tx.UpdatedAt, tx.Status, REJECTED_TX); err != nil {
				return errors.Wrap(err, "failed send to processor of transactions")
			}
		}
		return nil
	})
}

func (t *transactionProcessor) AcceptTx(txID int64) error {
	return t.db.InTransaction(func(tx *reform.TX) error {
		txObj := &Transaction{TransactionID: txID}
		if err := tx.Reload(txObj); err != nil {
			return errors.Wrap(err, "failed find transaction")
		}
		if !transactionStatusTransitionChart.Allowed(txObj.Status, ACCEPTED_TX) {
			return errors.New("not allowed transition status to ACCEPTED for transaction")
		}
		if err := t.Process(txObj.TransactionID, txObj.UpdatedAt, txObj.Status, ACCEPTED_TX); err != nil {
			return errors.Wrap(err, "failed send to processor of transactions")
		}
		return nil
	})
}

func (t *transactionProcessor) RejectTx(txID int64) error {
	return t.db.InTransaction(func(tx *reform.TX) error {
		txObj := &Transaction{TransactionID: txID}
		if err := tx.Reload(txObj); err != nil {
			return errors.Wrap(err, "failed find transaction")
		}
		if !transactionStatusTransitionChart.Allowed(txObj.Status, REJECTED_TX) {
			return errors.New("not allowed transition status to REJECTED for transaction")
		}
		if err := t.Process(txObj.TransactionID, txObj.UpdatedAt, txObj.Status, REJECTED_TX); err != nil {
			return errors.Wrap(err, "failed send to processor of transactions")
		}
		return nil
	})
}

func (t *transactionProcessor) Process(txID int64, updatedAt time.Time, currentStatus, nextStatus TransactionStatus) error {
	msg := &processorCommand{
		txID:          txID,
		updatedAt:     updatedAt,
		currentStatus: currentStatus,
		nextStatus:    nextStatus,
	}

	select {
	case t.toProcess <- msg:
	default:
		return errors.New("Processor can't keep up.")
	}

	return nil
}
