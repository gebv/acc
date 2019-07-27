package provider

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

type Store struct {
	DB *reform.DB
}

const (
	prefixOrderId = "acca"
)

func (s *Store) NewOrder(ordID string, providerName Provider, rawOrderStatus string) error {
	return s.DB.Insert(&InvoiceTransactionsExtOrders{
		OrderNumber:       formatOrderID(providerName, ordID),
		PaymentSystemName: providerName,
		RawOrderStatus:    rawOrderStatus,
	})
}

func (s *Store) GetByOrderID(ordID string, providerName Provider) (*InvoiceTransactionsExtOrders, error) {
	so := &InvoiceTransactionsExtOrders{OrderNumber: formatOrderID(providerName, ordID)}
	err := s.DB.Reload(so)
	if err != nil {
		if err == reform.ErrNoRows {
			return nil, err
		}
		return nil, errors.Wrap(err, "Failed get invoice transactions ext orders")
	}
	return so, nil
}

func (s *Store) SetStatus(ordID string, providerName Provider, newStatus string) error {
	o := &InvoiceTransactionsExtOrders{OrderNumber: formatOrderID(providerName, ordID)}
	err := s.DB.Reload(o)
	if err != nil {
		return err
	}
	o.RawOrderStatus = newStatus
	return s.DB.Save(o)
}

func (s *Store) NewOrderWithExtUpdate(ordID string, providerName Provider, rawOrderStatus string, extUpdatedAt time.Time) error {
	return s.DB.Insert(&InvoiceTransactionsExtOrders{
		OrderNumber:       formatOrderID(providerName, ordID),
		PaymentSystemName: providerName,
		RawOrderStatus:    rawOrderStatus,
		ExtUpdatedAt:      extUpdatedAt,
	})
}

func (s *Store) SetStatusWithExtUpdate(ordID string, providerName Provider, newStatus string, tm time.Time) error {
	o := &InvoiceTransactionsExtOrders{OrderNumber: formatOrderID(providerName, ordID)}
	err := s.DB.Reload(o)
	if err != nil {
		return err
	}
	o.RawOrderStatus = newStatus
	o.ExtUpdatedAt = tm
	return s.DB.Save(o)
}

func (s *Store) GetLastTimeFromPaymentSystemName(name Provider) (*time.Time, error) {
	tm := time.Date(2019, 01, 01, 00, 00, 00, 00, time.UTC)
	var so InvoiceTransactionsExtOrders
	err := s.DB.SelectOneTo(&so, "WHERE payment_system_name = $1 ORDER BY ext_updated_at DESC LIMIT 1", name)
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
	PaymentSystemName Provider  `reform:"payment_system_name"`
	RawOrderStatus    string    `reform:"raw_order_status"`
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

func formatOrderID(p Provider, extOrderID string) string {
	return prefixOrderId + fmt.Sprintf("-%s-%s", p, extOrderID)
}
