package shop

import (
	"errors"

	"github.com/gebv/acca"
	reform "gopkg.in/reform.v1"
)

func NewShopPG(
	db *reform.DB,
	ti acca.TransferInspector,
	pi PaymentInspector,
) *ShopPG {
	return &ShopPG{
		db: db,
		ti: ti,
		pi: pi,
	}
}

type ShopPG struct {
	db *reform.DB
	ti acca.TransferInspector
	pi PaymentInspector
}

func (s ShopPG) CreateOrder(orderID string, destinationID int64, _type OrderType, amount int64) (Order, error) {
	o := NewOrder(orderID, destinationID, _type, amount)
	if err := s.db.Insert(o); err != nil {
		return nil, err
	}
	return &orderContainer{o}, errors.New("not implemented")
}

func (s ShopPG) Invoice(orderID string, amount int64) (*acca.Invoice, error) {
	o, err := s.findOrder(orderID)
	if err != nil {
		return nil, err
	}
	i, err := NewInvoice(o, amount)
	if err != nil {
		return nil, err
	}
	if err := s.db.Insert(i); err != nil {
		return nil, err
	}
	return i, nil
}

func (s ShopPG) Pay(invoiceID, sourceID int64) (err error) {
	if err = s.pi.CanPay(invoiceID, sourceID); err != nil {
		return err
	}
	// TODO: check invoiceID
	transfer, tx, err := s.simpleTransfer()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = tx.Rollback()
			return
		}
		err = tx.Commit()
		return
	}()
	_, err = transfer.Hold(sourceID, invoiceID)
	return err
}

func (s ShopPG) simpleTransfer() (acca.Transfer, *reform.TX, error) {
	tx, err := s.db.Begin()
	return &SimpleTransfer{acca.NewTrnasferPG(tx)}, tx, err
}

func (s ShopPG) findOrder(orderID string) (Order, error) {
	return nil, errors.New("not implemented")
}

func (s ShopPG) findInvoice(invoiceID int64) (*acca.Invoice, error) {
	return nil, errors.New("not implemented")
}

func (s ShopPG) findAccount(accID int64) (*acca.Account, error) {
	return nil, errors.New("not implemented")
}
