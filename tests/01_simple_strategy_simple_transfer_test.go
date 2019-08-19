package tests

import (
	"testing"

	"github.com/gebv/acca/api"
)

func Test01_01SimpleStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.1.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.1.2", "curr1"))

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
					"acc1.1.1",
					"acc1.1.2",
					"",
					false,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		h.BalanceDec("acc1.1.1", 1000)
		h.AcceptedBalanceDec("acc1.1.1", 1000)
		h.BalanceInc("acc1.1.2", 1000)
		h.AcceptedBalanceInc("acc1.1.2", 1000)

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

	})

	t.Run("ChangeFromInvoiceRechargeOperation", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"simple",
			1000,
			nil,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc1.1.1",
					"acc1.1.2",
					"",
					false,
					api.OperStrategy_RECHARGE_OPS,
					1000,
				),
			},
		))

		h.BalanceInc("acc1.1.1", 1000)
		h.AcceptedBalanceInc("acc1.1.1", 1000)
		h.BalanceInc("acc1.1.2", 1000)
		h.AcceptedBalanceInc("acc1.1.2", 1000)

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

	})

	t.Run("InternalWithHold", func(t *testing.T) {
		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"simple",
			1000,
			nil,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc1.1.1",
					"acc1.1.2",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		h.BalanceDec("acc1.1.1", 1000)

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

		t.Run("BalanceChanges", h.BalanceChanges("acc1.1.1"))

		h.AcceptedBalanceDec("acc1.1.1", 1000)
		h.BalanceInc("acc1.1.2", 1000)
		h.AcceptedBalanceInc("acc1.1.2", 1000)

		t.Run("AcceptInvoice", h.AcceptInvoice("inv1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

		t.Run("BalanceChanges", h.BalanceChanges("acc1.1.1"))

	})

	t.Run("InternalReject", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"simple",
			1000,
			nil,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc1.1.1",
					"acc1.1.2",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		t.Run("RejectInvoice", h.RejectInvoice("inv1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_REJECTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

	})

	t.Run("InternalRejectWithHold", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"simple",
			1000,
			nil,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc1.1.1",
					"acc1.1.2",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		h.BalanceDec("acc1.1.1", 1000)

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

		h.BalanceInc("acc1.1.1", 1000)

		t.Run("RejectInvoice", h.RejectInvoice("inv1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_REJECTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))
	})

	t.Run("ChangeFromTransactions", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"simple",
			1000,
			nil,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc1.1.1",
					"acc1.1.2",
					"",
					false,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		h.BalanceDec("acc1.1.1", 1000)
		h.AcceptedBalanceDec("acc1.1.1", 1000)
		h.BalanceInc("acc1.1.2", 1000)
		h.AcceptedBalanceInc("acc1.1.2", 1000)

		t.Run("AuthTx", h.AuthTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

	})

	t.Run("InternalWithHoldFromTransactions", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"simple",
			1000,
			nil,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc1.1.1",
					"acc1.1.2",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		h.BalanceDec("acc1.1.1", 1000)

		t.Run("AuthTx", h.AuthTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

		h.AcceptedBalanceDec("acc1.1.1", 1000)
		h.BalanceInc("acc1.1.2", 1000)
		h.AcceptedBalanceInc("acc1.1.2", 1000)

		t.Run("AcceptTx", h.AcceptTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

	})

	t.Run("InternalRejectFromTransaction", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"simple",
			1000,
			nil,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc1.1.1",
					"acc1.1.2",
					"",
					false,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		t.Run("RejectTx", h.RejectTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_REJECTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

	})

	t.Run("InternalRejectWithHoldFromTransaction", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"simple",
			1000,
			nil,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc1.1.1",
					"acc1.1.2",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		h.BalanceDec("acc1.1.1", 1000)

		t.Run("AuthTx", h.AuthTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

		h.BalanceInc("acc1.1.1", 1000)

		t.Run("RejectTx", h.RejectTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_REJECTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

	})

	t.Run("ChangeFromTransactions", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"simple",
			1000,
			nil,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc1.1.1",
					"acc1.1.2",
					"",
					false,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx2",
			"simple",
			500,
			nil,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc1.1.2",
					"acc1.1.1",
					"",
					false,
					api.OperStrategy_SIMPLE_OPS,
					500,
				),
			},
		))

		h.BalanceDec("acc1.1.1", 1000)
		h.AcceptedBalanceDec("acc1.1.1", 1000)
		h.BalanceInc("acc1.1.2", 1000)
		h.AcceptedBalanceInc("acc1.1.2", 1000)

		t.Run("AuthTx", h.AuthTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

		h.BalanceDec("acc1.1.2", 500)
		h.AcceptedBalanceDec("acc1.1.2", 500)
		h.BalanceInc("acc1.1.1", 500)
		h.AcceptedBalanceInc("acc1.1.1", 500)

		t.Run("AuthTx", h.AuthTx("tx2"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))
	})

	t.Run("ChangeWithHoldFromTransactions", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"simple",
			1000,
			nil,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc1.1.1",
					"acc1.1.2",
					"",
					false,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx2",
			"simple",
			500,
			nil,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc1.1.2",
					"acc1.1.1",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					500,
				),
			},
		))

		h.BalanceDec("acc1.1.1", 1000)
		h.AcceptedBalanceDec("acc1.1.1", 1000)
		h.BalanceInc("acc1.1.2", 1000)
		h.AcceptedBalanceInc("acc1.1.2", 1000)

		t.Run("AuthTx", h.AuthTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

		h.BalanceDec("acc1.1.2", 500)

		t.Run("AuthTx", h.AuthTx("tx2"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

		h.BalanceInc("acc1.1.1", 500)
		h.AcceptedBalanceInc("acc1.1.1", 500)
		h.AcceptedBalanceDec("acc1.1.2", 500)

		t.Run("AcceptTx", h.AcceptTx("tx2"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

	})

	t.Run("ChangeWithHoldFromTransactions", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"simple",
			1000,
			nil,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc1.1.1",
					"acc1.1.2",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx2",
			"simple",
			500,
			nil,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc1.1.2",
					"acc1.1.1",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					500,
				),
			},
		))

		h.BalanceDec("acc1.1.1", 1000)

		t.Run("AuthTx", h.AuthTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

		h.BalanceDec("acc1.1.2", 500)

		t.Run("AuthTx", h.AuthTx("tx2"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

		h.AcceptedBalanceDec("acc1.1.1", 1000)
		h.BalanceInc("acc1.1.2", 1000)
		h.AcceptedBalanceInc("acc1.1.2", 1000)

		t.Run("AcceptTx", h.AcceptTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

		h.BalanceInc("acc1.1.1", 500)
		h.AcceptedBalanceInc("acc1.1.1", 500)
		h.AcceptedBalanceDec("acc1.1.2", 500)

		t.Run("AcceptTx", h.AcceptTx("tx2"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

	})

	t.Run("RejectWithHoldFromTransactions", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"simple",
			1000,
			nil,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc1.1.1",
					"acc1.1.2",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx2",
			"simple",
			500,
			nil,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc1.1.2",
					"acc1.1.1",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					500,
				),
			},
		))

		h.BalanceDec("acc1.1.1", 1000)

		t.Run("AuthTx", h.AuthTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

		h.BalanceDec("acc1.1.2", 500)

		t.Run("AuthTx", h.AuthTx("tx2"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

		h.BalanceInc("acc1.1.1", 1000)

		t.Run("RejectTx", h.RejectTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_REJECTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

		h.BalanceInc("acc1.1.2", 500)

		t.Run("RejectTx", h.RejectTx("tx2"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_REJECTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.1.2", "curr1"))

	})

}
