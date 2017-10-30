package acca

import "testing"
import "github.com/stretchr/testify/assert"

func TestCashier_closedTreansfer(t *testing.T) {
	_t := &TransferPG{nil, 1}
	_, err := _t.Hold(1, 1)
	assert.Error(t, err, ErrOrderClosed)
}

func TestCashier_successAccept(t *testing.T) {
	resetFixtures()
	tx, _ := db.Begin()
	txID, err := NewTrnasferPG(tx).Hold(2, 1)
	assert.NoError(t, err, "hold")
	assert.NoError(t, tx.Commit(), "hold commit")
	if assert.NotNil(t, txID, "tx ID") {
		tx, _ := db.Begin()
		err = NewTrnasferPG(tx).Accept(txID)
		assert.NoError(t, err, "accept")
		assert.NoError(t, tx.Commit(), "accept commit")
	}

	// TODO: more checks
	d := dumpFromInvoice(1)
	assert.True(t, d.i.Paid, "invoice - paid=true")
	assert.EqualValues(t, 0, d.FindAccount(d.i.SourceIDOrZero()).Balance, "check balance of acccount=%d", d.i.SourceID)
	assert.EqualValues(t, 1100, d.FindAccount(d.i.DestinationID).Balance)
}

func TestCashier_successReject(t *testing.T) {
	resetFixtures()
	tx, _ := db.Begin()
	txID, err := NewTrnasferPG(tx).Hold(2, 1)
	assert.NoError(t, err, "hold")
	assert.NoError(t, tx.Commit(), "hold commit")
	if assert.NotNil(t, txID, "tx ID") {
		tx, _ := db.Begin()
		err = NewTrnasferPG(tx).Reject(txID)
		assert.NoError(t, err, "reject")
		assert.NoError(t, tx.Commit(), "reject commit")
	}

	// check

	d := dumpFromInvoice(1)
	assert.False(t, d.i.Paid, "invoice - paid=false")
	assert.EqualValues(t, 100, d.FindAccount(d.i.SourceIDOrZero()).Balance)
	assert.EqualValues(t, 1000, d.FindAccount(d.i.DestinationID).Balance)
}
