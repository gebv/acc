package for_testing

import (
	"log"
	"sync"
	"time"

	"github.com/gebv/acca/engine"
)

var NoDbFromTest = NewNoDbFromTest()

type NoDbFromTestSTR struct {
	rw  sync.RWMutex
	inv map[int64]*engine.Invoice
	tr  map[int64]*engine.Transaction
}

func NewNoDbFromTest() *NoDbFromTestSTR {
	trKey1 := "1"
	trKey2 := "2"
	return &NoDbFromTestSTR{
		inv: map[int64]*engine.Invoice{
			1: {
				InvoiceID: 1,
				Key:       "simple", // TODO определить формат ключа для инвойса.
				Status:    engine.DRAFT_I,
				Strategy:  "invoice_simple_strategy",
				Meta:      nil,
				Payload:   nil,
				UpdatedAt: time.Time{},
				CreatedAt: time.Time{},
			},
			2: {
				InvoiceID: 2,
				Key:       "recharge", // TODO определить формат ключа для инвойса.
				Status:    engine.DRAFT_I,
				Strategy:  "invoice_recharge_strategy",
				Meta:      nil,
				Payload:   nil,
				UpdatedAt: time.Time{},
				CreatedAt: time.Time{},
			},
		},
		tr: map[int64]*engine.Transaction{
			1: {
				TransactionID:      1,
				InvoiceID:          1,
				Key:                &trKey1,
				Strategy:           "transaction_simple_strategy",
				Provider:           engine.INTERNAL,
				ProviderOperID:     nil,
				ProviderOperStatus: nil,
				Meta:               nil,
				Status:             engine.DRAFT_TX,
				UpdatedAt:          time.Time{},
				CreatedAt:          time.Time{},
			},
			2: {
				TransactionID:      2,
				InvoiceID:          1,
				Key:                &trKey2,
				Strategy:           "transaction_simple_strategy",
				Provider:           engine.INTERNAL,
				ProviderOperID:     nil,
				ProviderOperStatus: nil,
				Meta:               nil,
				Status:             engine.DRAFT_TX,
				UpdatedAt:          time.Time{},
				CreatedAt:          time.Time{},
			},
			3: {
				TransactionID:      3,
				InvoiceID:          2,
				Key:                &trKey1,
				Strategy:           "transaction_sberbank_strategy",
				Provider:           engine.SBERBANK,
				ProviderOperID:     nil,
				ProviderOperStatus: nil,
				Meta:               nil,
				Status:             engine.DRAFT_TX,
				UpdatedAt:          time.Time{},
				CreatedAt:          time.Time{},
			},
			4: {
				TransactionID:      4,
				InvoiceID:          2,
				Key:                &trKey2,
				Strategy:           "transaction_simple_strategy",
				Provider:           engine.INTERNAL,
				ProviderOperID:     nil,
				ProviderOperStatus: nil,
				Meta:               nil,
				Status:             engine.DRAFT_TX,
				UpdatedAt:          time.Time{},
				CreatedAt:          time.Time{},
			},
		},
	}
}

func (db *NoDbFromTestSTR) GetInv(invID int64) *engine.Invoice {
	db.rw.RLock()
	defer db.rw.RUnlock()
	inv, ok := NoDbFromTest.inv[invID]
	if !ok {
		log.Println("Invoice not found id: ", invID)
		return nil
	}
	res := *inv
	return &res
}

func (db *NoDbFromTestSTR) SaveInv(inv *engine.Invoice) {
	if inv == nil {
		return
	}
	db.rw.Lock()
	defer db.rw.Unlock()
	NoDbFromTest.inv[inv.InvoiceID] = inv
	return
}

func (db *NoDbFromTestSTR) ListTr(invID int64) []*engine.Transaction {
	db.rw.RLock()
	defer db.rw.RUnlock()
	res := make([]*engine.Transaction, 0, 2)
	for _, v := range db.tr {
		tr := *v
		if invID == v.InvoiceID {
			res = append(res, &tr)
		}
	}
	return res
}

func (db *NoDbFromTestSTR) GetTr(trID int64) *engine.Transaction {
	db.rw.RLock()
	defer db.rw.RUnlock()
	tr, ok := NoDbFromTest.tr[trID]
	if !ok {
		log.Println("Transaction not found id: ", trID)
		return nil
	}
	res := *tr
	return &res
}

func (db *NoDbFromTestSTR) SaveTr(tr *engine.Transaction) {
	if tr == nil {
		return
	}
	db.rw.Lock()
	defer db.rw.Unlock()
	NoDbFromTest.tr[tr.TransactionID] = tr
	return
}

var SimRequestToSberbank simRequest

type simRequest struct {
	m    sync.RWMutex
	fail bool
}

func (s *simRequest) SetFailRequest(fail bool) {
	s.m.Lock()
	defer s.m.Unlock()
	s.fail = fail
}

func (s *simRequest) RequestToSberbank() bool {
	s.m.RLock()
	defer s.m.RUnlock()
	time.Sleep(time.Second)
	if s.fail {
		time.Sleep(time.Second)
	}
	return s.fail
}
