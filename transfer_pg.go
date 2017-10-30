package acca

import (
	"errors"
	"log"
	"sync/atomic"
	"time"

	reform "gopkg.in/reform.v1"
)

var ErrTransferClosed = errors.New("transfer closed")

var _ Transfer = (*TransferPG)(nil)

func NewTrnasferPG(tx *reform.TX) *TransferPG {
	return &TransferPG{tx, 0}
}

type TransferPG struct {
	tx   *reform.TX
	once uint32
}

// Accept подтверждает транзакцию
// Успешно закрывается операция.
func (c *TransferPG) Accept(txID int64) (err error) {
	if c.once > 0 {
		return ErrTransferClosed
	}
	defer atomic.AddUint32(&c.once, 1)

	tx, err := c.findTransaction(c.tx, txID)
	if err != nil {
		log.Println("ERR: find transaction", txID, err)
		return err
	}

	if tx.Status != Authorization {
		err = errors.New("transaction has closed")
		return
	}

	i, err := c.findInvoice(c.tx, tx.InvoiceID)
	if err != nil {
		log.Println("ERR: find invoice", tx.InvoiceID, err)
		return err
	}
	if i.Paid {
		err = ErrInvoiceHasBeenPaid
		log.Println("ERR: invoice has been paid", i.InvoiceID)
		return
	}

	dst, err := c.findAccount(c.tx, i.DestinationID)
	if err != nil {
		log.Println("ERR: find destination account of invoice", i.DestinationID, err)
		return err
	}

	ch, err := c.accept(c.tx, tx, i, dst)
	if err != nil {
		log.Println("ERR: accept", err)
		return
	}

	log.Printf("INFO: success accept change_id=%d\n", ch.ChangeID)

	return nil
}

// Reject отклоняет транзакцию.
// Откатывается вся операция.
func (c *TransferPG) Reject(txID int64) (err error) {
	if c.once > 0 {
		return ErrTransferClosed
	}
	defer atomic.AddUint32(&c.once, 1)

	tx, err := c.findTransaction(c.tx, txID)
	if err != nil {
		log.Println("ERR: find transaction", txID, err)
		return err
	}

	if tx.Status != Authorization {
		err = errors.New("transaction has closed")
		return
	}

	i, err := c.findInvoice(c.tx, tx.InvoiceID)
	if err != nil {
		log.Println("ERR: find invoice", tx.InvoiceID, err)
		return err
	}
	if i.Paid {
		err = ErrInvoiceHasBeenPaid
		log.Println("ERR: invoice has been paid", i.InvoiceID)
		return
	}

	src, _ := c.findAccount(c.tx, i.SourceIDOrZero())
	if err != nil {
		log.Println("ERR: find source account of invoice", i.SourceID, err)
		return err
	}

	ch, err := c.reject(c.tx, tx, i, src)
	if err != nil {
		log.Println("ERR: reject", err)
		return
	}

	log.Printf("INFO: success reject change_id=%d\n", ch.ChangeID)

	return nil
}

// Hold замораживаются средства
// Средства становятся доступны адресату после подвтерждения транзакции
// В противном случае средства возвращаются
func (c *TransferPG) Hold(sourceID, invoiceID int64) (txID int64, err error) {
	if c.once > 0 {
		return 0, ErrTransferClosed
	}
	defer atomic.AddUint32(&c.once, 1)

	i, err := c.findInvoice(c.tx, invoiceID)
	if err != nil {
		log.Println("ERR: find invoice account", sourceID, err)
		return
	}
	if i.Paid {
		err = ErrInvoiceHasBeenPaid
		log.Println("ERR: invoice has been paid", i.InvoiceID)
		return
	}

	i.SetSourceID(sourceID)
	if err = c.tx.UpdateColumns(i, "source_id"); err != nil {
		log.Println("ERR: update invocie - set source_id", err)
		return
	}

	s, _ := c.findAccount(c.tx, sourceID)
	if err != nil {
		log.Println("ERR: find source account", sourceID, err)
		return
	}
	// d, _ := c.findAccount(tx, i.DestinationID) // TODO: проверка возможности перевода средств с SourceID -> DestinationID

	ch, holdTx, err := c.hold(c.tx, i, s)
	if err != nil {
		return 0, err
	}

	log.Printf("INFO: success hold change_id=%d tx_id=%d\n", ch.ChangeID, holdTx.TransactionID)

	return holdTx.TransactionID, nil
}

func (s *TransferPG) hold(
	tx *reform.TX,
	i *Invoice,
	src *Account, // source
) (hold *BalanceChanges, holdTx *Transaction, err error) {
	holdTx = &Transaction{
		InvoiceID:   i.InvoiceID,
		Amount:      i.Amount,
		Source:      src.AccountID,
		Destination: i.DestinationID,
		Status:      Authorization,
		CreatedAt:   time.Now(),
	}

	if err = tx.Insert(holdTx); err != nil {
		log.Println("ERR: new tx", err)
		return nil, nil, err
	}

	src.UpdatedAt = time.Now()
	src.Balance -= i.Amount

	if src.Balance < 0 {
		log.Println("ERR: не достаточно средств на счете", src.AccountID)
		return nil, nil, ErrInsufficientFunds
	}

	if err = tx.UpdateColumns(src, "balance", "updated_at"); err != nil {
		log.Println("ERR: update account", src.AccountID, err)
		return nil, nil, err
	}
	hold = &BalanceChanges{
		AccountID:     src.AccountID,
		TransactionID: holdTx.TransactionID,
		Type:          Hold,
		Amount:        -i.Amount,
		Balance:       src.Balance,
		CreatedAt:     time.Now(),
	}
	if err = tx.Insert(hold); err != nil {
		log.Println("ERR: change balance", src.AccountID, err)
		return nil, nil, err
	}

	return hold, holdTx, nil
}

func (s *TransferPG) accept(
	dbtx *reform.TX,
	tx *Transaction,
	i *Invoice,
	dst *Account, // destination
) (change *BalanceChanges, err error) {
	tx.Status = Accepted
	tx.ClosedAt = time.Now()

	if err = dbtx.UpdateColumns(tx, "status", "closed_at"); err != nil {
		log.Println("ERR: closed tx", err)
		return nil, err
	}

	dst.Balance += i.Amount
	dst.UpdatedAt = time.Now()

	if err = dbtx.UpdateColumns(dst, "balance", "updated_at"); err != nil {
		log.Println("ERR: update balance of account", err)
		return nil, err
	}

	change = &BalanceChanges{
		AccountID:     dst.AccountID,
		TransactionID: tx.TransactionID,
		Type:          Complete,
		Amount:        i.Amount,
		Balance:       dst.Balance,
		CreatedAt:     time.Now(),
	}
	if err = dbtx.Insert(change); err != nil {
		log.Println("ERR: change balance", dst.AccountID, err)
		return nil, err
	}

	i.Paid = true
	if err := dbtx.UpdateColumns(i, "paid"); err != nil {
		log.Println("ERR: update invoice - set paid=true", i.InvoiceID, err)
		return nil, err
	}

	return
}

func (s *TransferPG) reject(
	dbtx *reform.TX,
	tx *Transaction,
	i *Invoice,
	src *Account, // destination
) (change *BalanceChanges, err error) {
	tx.Status = Rejected
	tx.ClosedAt = time.Now()

	if err = dbtx.UpdateColumns(tx, "status", "closed_at"); err != nil {
		log.Println("ERR: closed tx", err)
		return nil, err
	}

	src.Balance += i.Amount
	src.UpdatedAt = time.Now()

	if err = dbtx.UpdateColumns(src, "balance", "updated_at"); err != nil {
		log.Println("ERR: update balance of account", err)
		return nil, err
	}

	change = &BalanceChanges{
		AccountID:     src.AccountID,
		TransactionID: tx.TransactionID,
		Type:          Refund,
		Amount:        i.Amount,
		Balance:       src.Balance,
		CreatedAt:     time.Now(),
	}
	if err = dbtx.Insert(change); err != nil {
		log.Println("ERR: change balance", src.AccountID, err)
		return nil, err
	}

	return
}

func (s *TransferPG) findInvoice(tx *reform.TX, objID int64) (obj *Invoice, err error) {
	obj = &Invoice{}
	if err = tx.FindByPrimaryKeyTo(obj, objID); err != nil {
		log.Println("ERR: find invoice by ID", objID, err)
		return nil, err
	}
	return obj, nil
}

func (s *TransferPG) findAccount(tx *reform.TX, objID int64) (obj *Account, err error) {
	obj = &Account{}
	if err = tx.FindByPrimaryKeyTo(obj, objID); err != nil {
		log.Println("ERR: find account by ID", objID, err)
		return nil, err
	}
	return obj, nil
}

func (s *TransferPG) findTransaction(tx *reform.TX, objID int64) (obj *Transaction, err error) {
	obj = &Transaction{}
	if err = tx.FindByPrimaryKeyTo(obj, objID); err != nil {
		log.Println("ERR: find transaction by ID", objID, err)
		return nil, err
	}
	return obj, nil
}
