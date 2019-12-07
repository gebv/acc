package tests

import (
	"testing"
	"time"

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
}

func Test01_02SimpleStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.2.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.2.2", "curr1"))

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
					"acc1.2.1",
					"acc1.2.2",
					"",
					false,
					api.OperStrategy_RECHARGE_OPS,
					1000,
				),
			},
		))

		h.BalanceInc("acc1.2.1", 1000)
		h.AcceptedBalanceInc("acc1.2.1", 1000)
		h.BalanceInc("acc1.2.2", 1000)
		h.AcceptedBalanceInc("acc1.2.2", 1000)

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.2.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.2.2", "curr1"))

	})
}

func Test01_03SimpleStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.3.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.3.2", "curr1"))

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
					"acc1.3.1",
					"acc1.3.2",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		h.BalanceDec("acc1.3.1", 1000)

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.3.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.3.2", "curr1"))

		t.Run("BalanceChanges", h.BalanceChanges("acc1.3.1"))

		h.AcceptedBalanceDec("acc1.3.1", 1000)
		h.BalanceInc("acc1.3.2", 1000)
		h.AcceptedBalanceInc("acc1.3.2", 1000)

		t.Run("AcceptInvoice", h.AcceptInvoice("inv1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.3.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.3.2", "curr1"))

		t.Run("BalanceChanges", h.BalanceChanges("acc1.3.1"))

	})
}

func Test01_04SimpleStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.4.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.4.2", "curr1"))

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
					"acc1.4.1",
					"acc1.4.2",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		t.Run("RejectInvoice", h.RejectInvoice("inv1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_REJECTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.4.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.4.2", "curr1"))

	})
}

func Test01_05SimpleStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.5.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.5.2", "curr1"))

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
					"acc1.5.1",
					"acc1.5.2",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		h.BalanceDec("acc1.5.1", 1000)

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.5.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.5.2", "curr1"))

		h.BalanceInc("acc1.5.1", 1000)

		t.Run("RejectInvoice", h.RejectInvoice("inv1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_REJECTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.5.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.5.2", "curr1"))
	})
}

func Test01_06SimpleStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.6.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.6.2", "curr1"))

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
					"acc1.6.1",
					"acc1.6.2",
					"",
					false,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		h.BalanceDec("acc1.6.1", 1000)
		h.AcceptedBalanceDec("acc1.6.1", 1000)
		h.BalanceInc("acc1.6.2", 1000)
		h.AcceptedBalanceInc("acc1.6.2", 1000)

		t.Run("AuthTx", h.AuthTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.6.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.6.2", "curr1"))

	})
}

func Test01_07SimpleStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.7.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.7.2", "curr1"))

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
					"acc1.7.1",
					"acc1.7.2",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		h.BalanceDec("acc1.7.1", 1000)

		t.Run("AuthTx", h.AuthTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.7.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.7.2", "curr1"))

		h.AcceptedBalanceDec("acc1.7.1", 1000)
		h.BalanceInc("acc1.7.2", 1000)
		h.AcceptedBalanceInc("acc1.7.2", 1000)

		t.Run("AcceptTx", h.AcceptTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.7.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.7.2", "curr1"))

	})
}

func Test01_08SimpleStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.8.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.8.2", "curr1"))

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
					"acc1.8.1",
					"acc1.8.2",
					"",
					false,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		t.Run("RejectTx", h.RejectTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_REJECTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.8.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.8.2", "curr1"))

	})
}

func Test01_09SimpleStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.9.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.9.2", "curr1"))

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
					"acc1.9.1",
					"acc1.9.2",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					1000,
				),
			},
		))

		h.BalanceDec("acc1.9.1", 1000)

		t.Run("AuthTx", h.AuthTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.9.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.9.2", "curr1"))

		h.BalanceInc("acc1.9.1", 1000)

		t.Run("RejectTx", h.RejectTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_REJECTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.9.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.9.2", "curr1"))

	})
}

func Test01_10SimpleStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.10.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.10.2", "curr1"))

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
					"acc1.10.1",
					"acc1.10.2",
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
					"acc1.10.2",
					"acc1.10.1",
					"",
					false,
					api.OperStrategy_SIMPLE_OPS,
					500,
				),
			},
		))

		h.BalanceDec("acc1.10.1", 1000)
		h.AcceptedBalanceDec("acc1.10.1", 1000)
		h.BalanceInc("acc1.10.2", 1000)
		h.AcceptedBalanceInc("acc1.10.2", 1000)

		t.Run("AuthTx", h.AuthTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_DRAFT_I)) // TODO уточнить статус и поправить стратегию

		t.Run("CheckBalances", h.CheckBalances("acc1.10.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.10.2", "curr1"))

		h.BalanceDec("acc1.10.2", 500)
		h.AcceptedBalanceDec("acc1.10.2", 500)
		h.BalanceInc("acc1.10.1", 500)
		h.AcceptedBalanceInc("acc1.10.1", 500)

		t.Run("AuthTx", h.AuthTx("tx2"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.10.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.10.2", "curr1"))
	})
}

func Test01_11SimpleStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.11.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.11.2", "curr1"))

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
					"acc1.11.1",
					"acc1.11.2",
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
					"acc1.11.2",
					"acc1.11.1",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					500,
				),
			},
		))

		h.BalanceDec("acc1.11.1", 1000)
		h.AcceptedBalanceDec("acc1.11.1", 1000)
		h.BalanceInc("acc1.11.2", 1000)
		h.AcceptedBalanceInc("acc1.11.2", 1000)

		t.Run("AuthTx", h.AuthTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_DRAFT_I)) // TODO уточнить статус и поправить стратегию

		t.Run("CheckBalances", h.CheckBalances("acc1.11.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.11.2", "curr1"))

		h.BalanceDec("acc1.11.2", 500)

		t.Run("AuthTx", h.AuthTx("tx2"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_DRAFT_I)) // TODO уточнить статус и поправить стратегию

		time.Sleep(time.Second)

		t.Run("CheckBalances", h.CheckBalances("acc1.11.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.11.2", "curr1"))

		h.BalanceInc("acc1.11.1", 500)
		h.AcceptedBalanceInc("acc1.11.1", 500)
		h.AcceptedBalanceDec("acc1.11.2", 500)

		t.Run("AcceptTx", h.AcceptTx("tx2"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.11.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.11.2", "curr1"))

	})
}

func Test01_12SimpleStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.12.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.12.2", "curr1"))

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
					"acc1.12.1",
					"acc1.12.2",
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
					"acc1.12.2",
					"acc1.12.1",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					500,
				),
			},
		))

		h.BalanceDec("acc1.12.1", 1000)

		t.Run("AuthTx", h.AuthTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_DRAFT_I)) // TODO уточнить статус и поправить стратегию

		t.Run("CheckBalances", h.CheckBalances("acc1.12.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.12.2", "curr1"))

		h.BalanceDec("acc1.12.2", 500)

		t.Run("AuthTx", h.AuthTx("tx2"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.12.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.12.2", "curr1"))

		h.AcceptedBalanceDec("acc1.12.1", 1000)
		h.BalanceInc("acc1.12.2", 1000)
		h.AcceptedBalanceInc("acc1.12.2", 1000)

		t.Run("AcceptTx", h.AcceptTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I)) // TODO уточнить статус и поправить стратегию

		t.Run("CheckBalances", h.CheckBalances("acc1.12.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.12.2", "curr1"))

		h.BalanceInc("acc1.12.1", 500)
		h.AcceptedBalanceInc("acc1.12.1", 500)
		h.AcceptedBalanceDec("acc1.12.2", 500)

		t.Run("AcceptTx", h.AcceptTx("tx2"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.12.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.12.2", "curr1"))

	})
}

func Test01_13SimpleStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.13.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.13.2", "curr1"))

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
					"acc1.13.1",
					"acc1.13.2",
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
					"acc1.13.2",
					"acc1.13.1",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					500,
				),
			},
		))

		h.BalanceDec("acc1.13.1", 1000)

		t.Run("AuthTx", h.AuthTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_DRAFT_I)) // TODO уточнить статус и поправить стратегию

		t.Run("CheckBalances", h.CheckBalances("acc1.13.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.13.2", "curr1"))

		h.BalanceDec("acc1.13.2", 500)

		t.Run("AuthTx", h.AuthTx("tx2"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.13.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.13.2", "curr1"))

		h.BalanceInc("acc1.13.1", 1000)

		t.Run("RejectTx", h.RejectTx("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I)) // TODO уточнить статус и поправить стратегию

		t.Run("CheckBalances", h.CheckBalances("acc1.13.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.13.2", "curr1"))

		h.BalanceInc("acc1.13.2", 500)

		t.Run("RejectTx", h.RejectTx("tx2"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_REJECTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.13.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.13.2", "curr1"))

	})

}

func Test01_14SimpleStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.10.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.10.2", "curr1"))

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
					"acc1.10.1",
					"acc1.10.2",
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
					"acc1.10.2",
					"acc1.10.1",
					"",
					false,
					api.OperStrategy_SIMPLE_OPS,
					500,
				),
			},
		))

		h.BalanceDec("acc1.10.1", 1000)
		h.AcceptedBalanceDec("acc1.10.1", 1000)
		h.BalanceInc("acc1.10.2", 1000)
		h.AcceptedBalanceInc("acc1.10.2", 1000)

		h.BalanceDec("acc1.10.2", 500)
		h.AcceptedBalanceDec("acc1.10.2", 500)
		h.BalanceInc("acc1.10.1", 500)
		h.AcceptedBalanceInc("acc1.10.1", 500)

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_ACCEPTED_TX))

		t.Run("WaitTransaction", h.WaitTransaction("tx2", api.TxStatus_ACCEPTED_TX))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

	})
}

func Test01_15SimpleStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.11.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.11.2", "curr1"))

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
					"acc1.11.1",
					"acc1.11.2",
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
					"acc1.11.2",
					"acc1.11.1",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					500,
				),
			},
		))

		h.BalanceDec("acc1.11.1", 1000)
		h.AcceptedBalanceDec("acc1.11.1", 1000)
		h.BalanceInc("acc1.11.2", 1000)
		h.AcceptedBalanceInc("acc1.11.2", 1000)

		h.BalanceDec("acc1.11.2", 500)

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_ACCEPTED_TX)) // TODO  уточнить статус и поправить стратегию

		t.Run("WaitTransaction", h.WaitTransaction("tx2", api.TxStatus_HOLD_TX))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_AUTH_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.11.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.11.2", "curr1"))

		h.BalanceInc("acc1.11.1", 500)
		h.AcceptedBalanceInc("acc1.11.1", 500)
		h.AcceptedBalanceDec("acc1.11.2", 500)

		t.Run("AcceptTx", h.AcceptTx("tx2"))

		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_ACCEPTED_TX))

		t.Run("WaitTransaction", h.WaitTransaction("tx2", api.TxStatus_ACCEPTED_TX))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.11.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.11.2", "curr1"))

	})
}

func Test01_16SimpleStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.12.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.12.2", "curr1"))

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
					"acc1.12.1",
					"acc1.12.2",
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
					"acc1.12.2",
					"acc1.12.1",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					500,
				),
			},
		))

		h.BalanceDec("acc1.12.1", 1000)
		h.BalanceDec("acc1.12.2", 500)

		t.Run("AuthTx", h.AuthInvoice("inv1"))

		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_HOLD_TX))

		t.Run("WaitTransaction", h.WaitTransaction("tx2", api.TxStatus_HOLD_TX))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.12.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.12.2", "curr1"))

		h.AcceptedBalanceDec("acc1.12.1", 1000)
		h.BalanceInc("acc1.12.2", 1000)
		h.AcceptedBalanceInc("acc1.12.2", 1000)

		t.Run("AcceptTx", h.AcceptTx("tx1"))

		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_ACCEPTED_TX))

		t.Run("WaitTransaction", h.WaitTransaction("tx2", api.TxStatus_HOLD_TX))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.12.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.12.2", "curr1"))

		h.BalanceInc("acc1.12.1", 500)
		h.AcceptedBalanceInc("acc1.12.1", 500)
		h.AcceptedBalanceDec("acc1.12.2", 500)

		t.Run("AcceptTx", h.AcceptTx("tx2"))

		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_ACCEPTED_TX))

		t.Run("WaitTransaction", h.WaitTransaction("tx2", api.TxStatus_ACCEPTED_TX))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.12.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.12.2", "curr1"))

	})
}

func Test01_17SimpleStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.13.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc1.13.2", "curr1"))

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
					"acc1.13.1",
					"acc1.13.2",
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
					"acc1.13.2",
					"acc1.13.1",
					"",
					true,
					api.OperStrategy_SIMPLE_OPS,
					500,
				),
			},
		))

		h.BalanceDec("acc1.13.1", 1000)
		h.BalanceDec("acc1.13.2", 500)

		t.Run("AuthTx", h.AuthInvoice("inv1"))

		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_HOLD_TX))

		t.Run("WaitTransaction", h.WaitTransaction("tx2", api.TxStatus_HOLD_TX))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.13.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.13.2", "curr1"))

		h.BalanceInc("acc1.13.1", 1000)

		t.Run("RejectTx", h.RejectTx("tx1"))

		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_REJECTED_TX))

		t.Run("WaitTransaction", h.WaitTransaction("tx2", api.TxStatus_HOLD_TX))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.13.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.13.2", "curr1"))

		h.BalanceInc("acc1.13.2", 500)

		t.Run("RejectTx", h.RejectTx("tx2"))

		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_REJECTED_TX))

		t.Run("WaitTransaction", h.WaitTransaction("tx2", api.TxStatus_REJECTED_TX))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_REJECTED_I))

		t.Run("CheckBalances", h.CheckBalances("acc1.13.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc1.13.2", "curr1"))

	})

}
