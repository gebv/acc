package provider

import (
	"time"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

type Store struct {
	DB *reform.DB
}

func (s *Store) NewOrder(ordID string, paymentSystemName string, rawOrderStatus string) error {
	return s.DB.Insert(&InvoiceTransactionsExtOrders{
		OrderNumber:       ordID,
		PaymentSystemName: paymentSystemName, // CARD_SBERBANK,
		RawOrderStatus:    rawOrderStatus,    // CREATED,
	})
}

func (s *Store) GetByOrderID(ordID string) (*InvoiceTransactionsExtOrders, error) {
	so := &InvoiceTransactionsExtOrders{OrderNumber: ordID}
	err := s.DB.Reload(so)
	if err != nil {
		if err == reform.ErrNoRows {
			return nil, err
		}
		return nil, errors.Wrap(err, "Failed get invoice transactions ext orders")
	}
	return so, nil
}

func (s *Store) SetStatus(sOrdID string, newStatus string) error {
	o := &InvoiceTransactionsExtOrders{OrderNumber: sOrdID}
	err := s.DB.Reload(o)
	if err != nil {
		return err
	}
	o.RawOrderStatus = newStatus
	return s.DB.Save(o)
}

func (s *Store) NewOrderWithExtUpdate(ordID string, paymentSystemName string, rawOrderStatus string, extUpdatedAt time.Time) error {
	return s.DB.Insert(&InvoiceTransactionsExtOrders{
		OrderNumber:       ordID,
		PaymentSystemName: paymentSystemName, // CARD_SBERBANK,
		RawOrderStatus:    rawOrderStatus,    // CREATED,
		ExtUpdatedAt:      extUpdatedAt,
	})
}

func (s *Store) SetStatusWithExtUpdate(sOrdID string, newStatus string, tm time.Time) error {
	o := &InvoiceTransactionsExtOrders{OrderNumber: sOrdID}
	err := s.DB.Reload(o)
	if err != nil {
		return err
	}
	o.RawOrderStatus = newStatus
	o.ExtUpdatedAt = tm
	return s.DB.Save(o)
}

func (s *Store) GetLastTimeFromNoncash() (*time.Time, error) {
	tm := time.Date(2019, 01, 01, 00, 00, 00, 00, time.UTC)
	var so InvoiceTransactionsExtOrders
	err := s.DB.SelectOneTo(&so, "WHERE payment_system_name = 'noncash' ORDER BY ext_updated_at DESC LIMIT 1")
	if err != nil {
		if err == reform.ErrNoRows {
			return &tm, nil
		}
		return nil, errors.Wrap(err, "Failed get invoice transactions ext orders")
	}
	return &so.ExtUpdatedAt, nil
}

//go:generate reform

//reform:acca.invoice_transactions_ext_orders
type InvoiceTransactionsExtOrders struct {
	OrderNumber       string    `reform:"order_number,pk"`
	PaymentSystemName string    `reform:"payment_system_name"`
	RawOrderStatus    string    `reform:"raw_order_status"`
	OrderStatus       string    `reform:"order_status"`
	CreatedAt         time.Time `reform:"created_at"`
	UpdatedAt         time.Time `reform:"updated_at"`
	ExtUpdatedAt      time.Time `reform:"ext_updated_at"`
}

func (o *InvoiceTransactionsExtOrders) BeforeInsert() error {
	o.UpdatedAt = time.Now()
	o.CreatedAt = time.Now()
	return nil
}

func (o *InvoiceTransactionsExtOrders) BeforeUpdate() error {
	o.UpdatedAt = time.Now()
	return nil
}
