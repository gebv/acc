package tests

import (
	"testing"

	"github.com/gebv/acca/api"
)

func Test05_01GetUpdates(t *testing.T) {
	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc5.1.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc5.1.2", "curr1"))

	t.Run("ChangeFromInvoice", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"simple",
			1000,
			nil,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc5.1.1",
					"acc5.1.2",
					"",
					false,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		h.BalanceDec("acc5.1.1", 1000)
		h.AcceptedBalanceDec("acc5.1.1", 1000)
		h.BalanceInc("acc5.1.2", 1000)
		h.AcceptedBalanceInc("acc5.1.2", 1000)

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CompareUpdates", h.CompareUpdates([]*api.Update{
			{
				Type: &api.Update_UpdatedInvoice{
					UpdatedInvoice: &api.UpdatedInvoice{
						InvoiceId: h.GetInvoiceID("inv1"),
						Status:    api.InvoiceStatus_AUTH_I,
					},
				},
			},
			{
				Type: &api.Update_UpdatedTransaction{
					UpdatedTransaction: &api.UpdatedTransaction{
						TransactionId: h.GetTxID("tx1"),
						Status:        api.TxStatus_AUTH_TX,
					},
				},
			},
			{
				Type: &api.Update_UpdatedInvoice{
					UpdatedInvoice: &api.UpdatedInvoice{
						InvoiceId: h.GetInvoiceID("inv1"),
						Status:    api.InvoiceStatus_ACCEPTED_I,
					},
				},
			},
		}))

		t.Run("CheckBalances", h.CheckBalances("acc5.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc5.1.2", "curr1"))

	})

}
