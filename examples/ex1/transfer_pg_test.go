package ex1

import "testing"
import "github.com/stretchr/testify/assert"
import "github.com/gebv/acca"

func TestCashier_closedTreansfer(t *testing.T) {
	_t := &Transfer{nil, 1}
	_, err := _t.Hold(1, 1)
	assert.Error(t, err, acca.ErrOrderClosed)
}

func TestCashier_successAccept(t *testing.T) {
	resetFixtures()

	txID, err := NewTrnasfer(db).Hold(2, 1)
	assert.NoError(t, err, "hold")
	if assert.NotNil(t, txID, "tx ID") {
		err = NewTrnasfer(db).Accept(txID)
		assert.NoError(t, err, "accept")
	}

	// TODO: more checks
	d := dumpFromInvoice(1)
	assert.True(t, d.i.Paid, "invoice - paid=true")
	assert.EqualValues(t, 0, d.FindAccount(d.i.SourceIDOrZero()).Balance, "check balance of acccount=%d", d.i.SourceID)
	assert.EqualValues(t, 1100, d.FindAccount(d.i.DestinationID).Balance)
}

func TestCashier_successReject(t *testing.T) {
	resetFixtures()

	txID, err := NewTrnasfer(db).Hold(2, 1)
	assert.NoError(t, err, "hold")
	if assert.NotNil(t, txID, "tx ID") {
		err = NewTrnasfer(db).Reject(txID)
		assert.NoError(t, err, "reject")
	}

	// check

	d := dumpFromInvoice(1)
	assert.False(t, d.i.Paid, "invoice - paid=false")
	assert.EqualValues(t, 100, d.FindAccount(d.i.SourceIDOrZero()).Balance)
	assert.EqualValues(t, 1000, d.FindAccount(d.i.DestinationID).Balance)
}
