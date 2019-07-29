package tests

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gebv/acca/api"
	"github.com/stretchr/testify/require"
)

func Test02_01SberbankStrategy(t *testing.T) {
	h := NewHelperData()

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc2.1.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc2.1.2", "curr1"))

	t.Run("RechargeFromInvoice", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple"))

		Meta := map[string]string{
			"callback":    "https://ya.ru",
			"description": "",
			"email":       "test@mail.ru",
		}
		meta, err := json.Marshal(&Meta)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"sberbank",
			1000,
			&meta,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc2.1.1",
					"acc2.1.2",
					"",
					true,
					api.OperStrategy_RECHARGE_OPS,
					1000,
				),
			},
		))

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		h.Sleep(15)

		t.Run("CheckBalances", h.CheckBalances("acc2.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc2.1.2", "curr1"))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", "CREATED", api.TxStatus_AUTH_TX))

		t.Run("SendCardDataInSberbank", h.SendCardDataInSberbank("tx1"))

		time.Sleep(15 * time.Second)

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", "APPROVED", api.TxStatus_HOLD_TX))

		h.BalanceInc("acc2.1.1", 1000)
		h.AcceptedBalanceInc("acc2.1.1", 1000)
		h.BalanceInc("acc2.1.2", 1000)
		h.AcceptedBalanceInc("acc2.1.2", 1000)

		t.Run("AcceptInvoice", h.AcceptInvoice("inv1"))

		h.Sleep(15)

		t.Run("CheckTransaction", h.CheckTransaction("tx1", api.TxStatus_ACCEPTED_TX))

		t.Run("CheckBalances", h.CheckBalances("acc2.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc2.1.2", "curr1"))

	})
}

func Test02_02SberbankStrategy(t *testing.T) {
	h := NewHelperData()

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc2.2.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc2.2.2", "curr1"))

	t.Run("RechargeFromInvoice", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple"))

		Meta := map[string]string{
			"callback":    "https://ya.ru",
			"description": "",
			"email":       "test@mail.ru",
		}
		meta, err := json.Marshal(&Meta)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"sberbank",
			1000,
			&meta,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc2.2.1",
					"acc2.2.2",
					"",
					true,
					api.OperStrategy_RECHARGE_OPS,
					1000,
				),
			},
		))

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		time.Sleep(15 * time.Second)

		t.Run("CheckBalances", h.CheckBalances("acc2.2.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc2.2.2", "curr1"))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", "CREATED", api.TxStatus_AUTH_TX))

		t.Run("SendCardDataInSberbank", h.SendCardDataInSberbank("tx1"))

		time.Sleep(15 * time.Second)

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", "APPROVED", api.TxStatus_HOLD_TX))

		t.Run("CheckBalances", h.CheckBalances("acc2.2.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc2.2.2", "curr1"))

		t.Run("RejectInvoice", h.RejectInvoice("inv1"))

		time.Sleep(10 * time.Second)

		t.Run("CheckTransaction", h.CheckTransaction("tx1", api.TxStatus_REJECTED_TX))

		t.Run("CheckBalances", h.CheckBalances("acc2.2.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc2.2.2", "curr1"))

	})
}

func Test02_03SberbankStrategy(t *testing.T) {
	h := NewHelperData()

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc2.3.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc2.3.2", "curr1"))

	t.Run("RechargeFromInvoice", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple"))

		Meta := map[string]string{
			"callback":    "https://ya.ru",
			"description": "",
			"email":       "test@mail.ru",
		}
		meta, err := json.Marshal(&Meta)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"sberbank",
			1000,
			&meta,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc2.3.1",
					"acc2.3.2",
					"",
					false,
					api.OperStrategy_RECHARGE_OPS,
					1000,
				),
				h.CreateOperation(
					"acc2.3.1",
					"acc2.3.2",
					"",
					false,
					api.OperStrategy_WITHDRAW_OPS,
					100,
				),
			},
		))

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		time.Sleep(15 * time.Second)

		t.Run("CheckBalances", h.CheckBalances("acc2.3.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc2.3.2", "curr1"))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", "CREATED", api.TxStatus_AUTH_TX))

		h.BalanceInc("acc2.3.1", 1000)
		h.BalanceDec("acc2.3.1", 100)
		h.BalanceInc("acc2.3.2", 1000)
		h.BalanceDec("acc2.3.2", 100)
		h.AcceptedBalanceInc("acc2.3.1", 1000)
		h.AcceptedBalanceDec("acc2.3.1", 100)
		h.AcceptedBalanceInc("acc2.3.2", 1000)
		h.AcceptedBalanceDec("acc2.3.2", 100)

		t.Run("SendCardDataInSberbank", h.SendCardDataInSberbank("tx1"))

		time.Sleep(15 * time.Second)

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", "DEPOSITED", api.TxStatus_ACCEPTED_TX))

		t.Run("CheckBalances", h.CheckBalances("acc2.3.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc2.3.2", "curr1"))

	})
}
