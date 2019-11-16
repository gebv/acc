package tests

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go"

	"github.com/gebv/acca/api"
)

func Test06_01StripeStrategy(t *testing.T) {
	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc6.1.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc6.1.2", "curr1"))

	t.Run("RechargeFromInvoice", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		Meta := map[string]string{
			"description": "",
			"email":       "test@mail.ru",
		}
		meta, err := json.Marshal(&Meta)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"stripe",
			1001,
			&meta,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc6.1.1",
					"acc6.1.2",
					"",
					true,
					api.OperStrategy_RECHARGE_OPS,
					1001,
				),
			},
		))

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		// Так как авторизация транзакции проходит через 2 запроса.
		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_AUTH_TX))
		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_AUTH_TX))

		t.Run("CheckBalances", h.CheckBalances("acc6.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc6.1.2", "curr1"))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", string(stripe.PaymentIntentStatusRequiresPaymentMethod), api.TxStatus_AUTH_TX))

		t.Run("SendCardDataInStripe", h.SendCardDataInStripe("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", string(stripe.PaymentIntentStatusRequiresCapture), api.TxStatus_HOLD_TX))

		h.BalanceInc("acc6.1.1", 1001)
		h.AcceptedBalanceInc("acc6.1.1", 1001)
		h.BalanceInc("acc6.1.2", 1001)
		h.AcceptedBalanceInc("acc6.1.2", 1001)

		t.Run("AcceptInvoice", h.AcceptInvoice("inv1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckTransaction", h.CheckTransaction("tx1", api.TxStatus_ACCEPTED_TX))

		t.Run("CheckBalances", h.CheckBalances("acc6.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc6.1.2", "curr1"))

	})
}

func Test06_02StripeStrategy(t *testing.T) {
	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc6.2.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc6.2.2", "curr1"))

	t.Run("RechargeFromInvoice", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		Meta := map[string]string{
			"description": "",
			"email":       "test@mail.ru",
		}
		meta, err := json.Marshal(&Meta)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"stripe",
			1000,
			&meta,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc6.2.1",
					"acc6.2.2",
					"",
					true,
					api.OperStrategy_RECHARGE_OPS,
					1000,
				),
			},
		))

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		// Так как авторизация транзакции проходит через 2 запроса.
		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_AUTH_TX))
		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_AUTH_TX))

		t.Run("CheckBalances", h.CheckBalances("acc6.2.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc6.2.2", "curr1"))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", string(stripe.PaymentIntentStatusRequiresPaymentMethod), api.TxStatus_AUTH_TX))

		t.Run("SendCardDataInStripe", h.SendCardDataInStripe("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", string(stripe.PaymentIntentStatusRequiresCapture), api.TxStatus_HOLD_TX))

		t.Run("CheckBalances", h.CheckBalances("acc6.2.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc6.2.2", "curr1"))

		t.Run("RejectInvoice", h.RejectInvoice("inv1"))

		// Так как отмена транзакции проходит через 2 запроса.
		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_REJECTED_TX))
		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_REJECTED_TX))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", string(stripe.PaymentIntentStatusCanceled), api.TxStatus_REJECTED_TX))

		t.Run("CheckBalances", h.CheckBalances("acc6.2.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc6.2.2", "curr1"))

	})
}

func Test06_03StripeStrategy(t *testing.T) {
	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc6.3.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc6.3.2", "curr1"))

	t.Run("RechargeFromInvoice", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		Meta := map[string]string{
			"description": "",
			"email":       "test@mail.ru",
		}
		meta, err := json.Marshal(&Meta)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"stripe",
			1000,
			&meta,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc6.3.1",
					"acc6.3.2",
					"",
					false,
					api.OperStrategy_RECHARGE_OPS,
					1000,
				),
				h.CreateOperation(
					"acc6.3.1",
					"acc6.3.2",
					"",
					false,
					api.OperStrategy_WITHDRAW_OPS,
					100,
				),
			},
		))

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		// Так как авторизация транзакции проходит через 2 запроса.
		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_AUTH_TX))
		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_AUTH_TX))

		t.Run("CheckBalances", h.CheckBalances("acc6.3.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc6.3.2", "curr1"))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", string(stripe.PaymentIntentStatusRequiresPaymentMethod), api.TxStatus_AUTH_TX))

		h.BalanceInc("acc6.3.1", 1000)
		h.BalanceDec("acc6.3.1", 100)
		h.BalanceInc("acc6.3.2", 1000)
		h.BalanceDec("acc6.3.2", 100)
		h.AcceptedBalanceInc("acc6.3.1", 1000)
		h.AcceptedBalanceDec("acc6.3.1", 100)
		h.AcceptedBalanceInc("acc6.3.2", 1000)
		h.AcceptedBalanceDec("acc6.3.2", 100)

		t.Run("SendCardDataInStripe", h.SendCardDataInStripe("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", string(stripe.PaymentIntentStatusSucceeded), api.TxStatus_ACCEPTED_TX))

		t.Run("CheckBalances", h.CheckBalances("acc6.3.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc6.3.2", "curr1"))

	})
}

func Test06_04StripeStrategy(t *testing.T) {
	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc6.4.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc6.4.2", "curr1"))

	t.Run("RechargeFromInvoice", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		Meta := map[string]string{
			"customer_id": "cus_G5hZebFbQ77Grp",
			"pm_id":       "pm_1FbHHfBz5RLqjsMcdqPUfpaE", // Если не указать,
			// то при подтверждении нужно указывать данные карты или платежный метод добавленных ранее к пользователю.
			"description": "",
			"email":       "test@mail.ru",
		}
		meta, err := json.Marshal(&Meta)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"stripe",
			1001,
			&meta,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc6.4.1",
					"acc6.4.2",
					"",
					true,
					api.OperStrategy_RECHARGE_OPS,
					1001,
				),
			},
		))

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		// Так как авторизация транзакции проходит через 2 запроса.
		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_AUTH_TX))
		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_AUTH_TX))

		t.Run("CheckBalances", h.CheckBalances("acc6.4.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc6.4.2", "curr1"))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", string(stripe.PaymentIntentStatusRequiresConfirmation), api.TxStatus_AUTH_TX))

		t.Run("SendConfirmPaymentInStripe", h.SendConfirmPaymentInStripe("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", string(stripe.PaymentIntentStatusRequiresCapture), api.TxStatus_HOLD_TX))

		h.BalanceInc("acc6.4.1", 1001)
		h.AcceptedBalanceInc("acc6.4.1", 1001)
		h.BalanceInc("acc6.4.2", 1001)
		h.AcceptedBalanceInc("acc6.4.2", 1001)

		t.Run("AcceptInvoice", h.AcceptInvoice("inv1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckTransaction", h.CheckTransaction("tx1", api.TxStatus_ACCEPTED_TX))

		t.Run("CheckBalances", h.CheckBalances("acc6.4.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc6.4.2", "curr1"))

	})
}

func Test06_05StripeStrategy(t *testing.T) {
	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc6.5.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc6.5.2", "curr1"))

	t.Run("RechargeFromInvoice", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		Meta := map[string]string{
			"customer_id": "cus_G5hZebFbQ77Grp",
			"description": "",
			"email":       "test@mail.ru",
		}
		meta, err := json.Marshal(&Meta)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"stripe",
			1001,
			&meta,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc6.5.1",
					"acc6.5.2",
					"",
					true,
					api.OperStrategy_RECHARGE_OPS,
					1001,
				),
			},
		))

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		// Так как авторизация транзакции проходит через 2 запроса.
		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_AUTH_TX))
		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_AUTH_TX))

		t.Run("CheckBalances", h.CheckBalances("acc6.5.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc6.5.2", "curr1"))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", string(stripe.PaymentIntentStatusRequiresPaymentMethod), api.TxStatus_AUTH_TX))

		t.Run("SendConfirmWithPaymentMethodInStripe", h.SendConfirmWithPaymentMethodInStripe("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", string(stripe.PaymentIntentStatusRequiresCapture), api.TxStatus_HOLD_TX))

		h.BalanceInc("acc6.5.1", 1001)
		h.AcceptedBalanceInc("acc6.5.1", 1001)
		h.BalanceInc("acc6.5.2", 1001)
		h.AcceptedBalanceInc("acc6.5.2", 1001)

		t.Run("AcceptInvoice", h.AcceptInvoice("inv1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckTransaction", h.CheckTransaction("tx1", api.TxStatus_ACCEPTED_TX))

		t.Run("CheckBalances", h.CheckBalances("acc6.5.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc6.5.2", "curr1"))

	})
}
