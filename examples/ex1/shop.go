package ex1

import (
	"errors"

	"github.com/gebv/acca"
	reform "gopkg.in/reform.v1"
)

// var _ acca.Seller = (*Shop)(nil)
var _ acca.Payment = (*Shop)(nil)

type Shop struct {
	db *reform.DB
}

func (s *Shop) NewOrder() {}

// func (s *Shop) Invoice()                            {}
func (s *Shop) Pay(invoiceID, sourceID int64) error {
	return errors.New("not implemented")
}

func (s *Shop) trnasfer() acca.Transfer {
	return NewTrnasfer(s.db)
}
