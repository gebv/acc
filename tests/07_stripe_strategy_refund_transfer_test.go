package tests

import (
	"encoding/json"
	"strconv"
	"testing"

	pkgStripe "github.com/stripe/stripe-go"

	"github.com/gebv/acca/api"
	"github.com/gebv/acca/provider/stripe"

	"github.com/stretchr/testify/require"
)

func Test07_01StripeStrategy(t *testing.T) {
	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc7.1.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc7.1.2", "curr1"))

	t.Run("RechargeFromInvoice", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

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
			"stripe",
			1000,
			&meta,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc7.1.1",
					"acc7.1.2",
					"",
					false,
					api.OperStrategy_RECHARGE_OPS,
					1000,
				),
			},
		))

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		// Так как авторизация транзакции проходит через 2 запроса.
		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_AUTH_TX))
		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_AUTH_TX))

		t.Run("CheckBalances", h.CheckBalances("acc7.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc7.1.2", "curr1"))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", string(pkgStripe.PaymentIntentStatusRequiresPaymentMethod), api.TxStatus_AUTH_TX))

		t.Run("SendCardDataInStripe", h.SendCardDataInStripe("tx1"))

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		h.BalanceInc("acc7.1.1", 1000)
		h.AcceptedBalanceInc("acc7.1.1", 1000)
		h.BalanceInc("acc7.1.2", 1000)
		h.AcceptedBalanceInc("acc7.1.2", 1000)

		t.Run("CheckBalances", h.CheckBalances("acc7.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc7.1.2", "curr1"))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", string(pkgStripe.PaymentIntentStatusSucceeded), api.TxStatus_ACCEPTED_TX))

	})

	stripeProvider := stripe.NewProvider(
		nil,
		nil,
	)
	var extOrderID string
	res, err := h.invC.GetTransactionByIDs(h.authCtx, &api.GetTransactionByIDsRequest{
		TxIds: []int64{h.GetTxID("tx1")},
	})
	require.NoError(t, err)
	require.NotEmpty(t, res.GetTransactions())
	require.NotNil(t, res.GetTransactions()[0].GetProviderOperId())
	extOrderID = *res.GetTransactions()[0].GetProviderOperId()
	t.Run("CheckStripeStatus", func(t *testing.T) {
		st, err := stripeProvider.GetPaymentIntent(extOrderID)
		require.NoError(t, err)
		require.EqualValues(t, 1000, st.Amount)
		require.EqualValues(t, 1000, st.AmountReceived)
		require.EqualValues(t, pkgStripe.PaymentIntentStatusSucceeded, st.Status)
	})

	t.Run("RefundFromInvoiceRejected", func(t *testing.T) {

		MetaInv := map[string]string{
			"invoice_id": strconv.FormatInt(h.GetInvoiceID("inv1"), 10),
		}
		metaInv, err := json.Marshal(&MetaInv)
		require.NoError(t, err)

		t.Run("NewInvoice", h.NewInvoice("inv2", "refund", &metaInv))

		MetaTr := map[string]string{
			"tx_id": strconv.FormatInt(h.GetTxID("tx1"), 10),
		}
		metaTr, err := json.Marshal(&MetaTr)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv2",
			"tx2",
			"stripe_refund",
			600,
			&metaTr,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc7.1.1",
					"acc7.1.2",
					"",
					false,
					api.OperStrategy_WITHDRAW_OPS,
					600,
				),
			},
		))

		t.Run("RejectInvoice", h.RejectInvoice("inv2"))

		t.Run("WaitInvoice", h.WaitInvoice("inv2", api.InvoiceStatus_REJECTED_I))

		t.Run("CheckTransaction", h.CheckTransaction("tx2", api.TxStatus_REJECTED_TX))

		t.Run("CheckBalances", h.CheckBalances("acc7.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc7.1.2", "curr1"))

	})

	t.Run("CheckStripeStatus", func(t *testing.T) {
		st, err := stripeProvider.GetPaymentIntent(extOrderID)
		require.NoError(t, err)
		require.EqualValues(t, 1000, st.Amount)
		require.EqualValues(t, 1000, st.AmountReceived)
		require.EqualValues(t, pkgStripe.PaymentIntentStatusSucceeded, st.Status)
	})

	t.Run("RefundFromInvoice", func(t *testing.T) {

		MetaInv := map[string]string{
			"invoice_id": strconv.FormatInt(h.GetInvoiceID("inv1"), 10),
		}
		metaInv, err := json.Marshal(&MetaInv)
		require.NoError(t, err)

		t.Run("NewInvoice", h.NewInvoice("inv2", "refund", &metaInv))

		MetaTr := map[string]string{
			"tx_id": strconv.FormatInt(h.GetTxID("tx1"), 10),
		}
		metaTr, err := json.Marshal(&MetaTr)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv2",
			"tx2",
			"stripe_refund",
			600,
			&metaTr,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc7.1.1",
					"acc7.1.2",
					"",
					false,
					api.OperStrategy_WITHDRAW_OPS,
					600,
				),
			},
		))

		h.BalanceDec("acc7.1.1", 600)
		h.AcceptedBalanceDec("acc7.1.1", 600)
		h.BalanceDec("acc7.1.2", 600)
		h.AcceptedBalanceDec("acc7.1.2", 600)

		t.Run("AuthInvoice", h.AuthInvoice("inv2"))

		t.Run("WaitInvoice", h.WaitInvoice("inv2", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckTransaction", h.CheckTransaction("tx2", api.TxStatus_ACCEPTED_TX))

		t.Run("CheckBalances", h.CheckBalances("acc7.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc7.1.2", "curr1"))

	})

	t.Run("CheckStripeStatus", func(t *testing.T) {
		st, err := stripeProvider.GetPaymentIntent(extOrderID)
		require.NoError(t, err)
		require.EqualValues(t, 1000, st.Amount)
		require.EqualValues(t, 1000, st.AmountReceived)
		require.NotNil(t, st.Charges)
		require.Len(t, st.Charges.Data, 1)
		require.NotNil(t, st.Charges.Data[0])
		require.EqualValues(t, 1000, st.Charges.Data[0].Amount)
		require.EqualValues(t, 600, st.Charges.Data[0].AmountRefunded)
		require.EqualValues(t, pkgStripe.PaymentIntentStatusSucceeded, st.Status)
		require.False(t, st.Charges.Data[0].Refunded)
		require.Len(t, st.Charges.Data[0].Refunds.Data, 1)
		require.EqualValues(t, 600, st.Charges.Data[0].Refunds.Data[0].Amount)
		require.EqualValues(t, pkgStripe.RefundStatusSucceeded, st.Charges.Data[0].Refunds.Data[0].Status)
	})

	t.Run("RefundFromInvoice", func(t *testing.T) {

		MetaInv := map[string]string{
			"invoice_id": strconv.FormatInt(h.GetInvoiceID("inv1"), 10),
		}
		metaInv, err := json.Marshal(&MetaInv)
		require.NoError(t, err)

		t.Run("NewInvoice", h.NewInvoice("inv2", "refund", &metaInv))

		MetaTr := map[string]string{
			"tx_id": strconv.FormatInt(h.GetTxID("tx1"), 10),
		}
		metaTr, err := json.Marshal(&MetaTr)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv2",
			"tx2",
			"stripe_refund",
			600,
			&metaTr,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc7.1.1",
					"acc7.1.2",
					"",
					false,
					api.OperStrategy_WITHDRAW_OPS,
					600,
				),
			},
		))

		t.Run("AuthInvoice", h.AuthInvoice("inv2"))

		t.Run("WaitInvoice", h.WaitInvoice("inv2", api.InvoiceStatus_AUTH_I))

		t.Run("CheckTransaction", h.CheckTransaction("tx2", api.TxStatus_DRAFT_TX))

		t.Run("CheckBalances", h.CheckBalances("acc7.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc7.1.2", "curr1"))

	})

	t.Run("CheckStripeStatus", func(t *testing.T) {
		st, err := stripeProvider.GetPaymentIntent(extOrderID)
		require.NoError(t, err)
		require.EqualValues(t, 1000, st.Amount)
		require.EqualValues(t, 1000, st.AmountReceived)
		require.NotNil(t, st.Charges)
		require.Len(t, st.Charges.Data, 1)
		require.NotNil(t, st.Charges.Data[0])
		require.EqualValues(t, 1000, st.Charges.Data[0].Amount)
		require.EqualValues(t, 600, st.Charges.Data[0].AmountRefunded)
		require.EqualValues(t, pkgStripe.PaymentIntentStatusSucceeded, st.Status)
		require.False(t, st.Charges.Data[0].Refunded)
		require.Len(t, st.Charges.Data[0].Refunds.Data, 1)
		require.EqualValues(t, 600, st.Charges.Data[0].Refunds.Data[0].Amount)
		require.EqualValues(t, pkgStripe.RefundStatusSucceeded, st.Charges.Data[0].Refunds.Data[0].Status)
	})

	t.Run("RefundFromInvoice", func(t *testing.T) {

		MetaInv := map[string]string{
			"invoice_id": strconv.FormatInt(h.GetInvoiceID("inv1"), 10),
		}
		metaInv, err := json.Marshal(&MetaInv)
		require.NoError(t, err)

		t.Run("NewInvoice", h.NewInvoice("inv2", "refund", &metaInv))

		MetaTr := map[string]string{
			"tx_id": strconv.FormatInt(h.GetTxID("tx1"), 10),
		}
		metaTr, err := json.Marshal(&MetaTr)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv2",
			"tx2",
			"stripe_refund",
			400,
			&metaTr,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc7.1.1",
					"acc7.1.2",
					"",
					false,
					api.OperStrategy_WITHDRAW_OPS,
					400,
				),
			},
		))

		h.BalanceDec("acc7.1.1", 400)
		h.AcceptedBalanceDec("acc7.1.1", 400)
		h.BalanceDec("acc7.1.2", 400)
		h.AcceptedBalanceDec("acc7.1.2", 400)

		t.Run("AuthInvoice", h.AuthInvoice("inv2"))

		t.Run("WaitInvoice", h.WaitInvoice("inv2", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckTransaction", h.CheckTransaction("tx2", api.TxStatus_ACCEPTED_TX))

		t.Run("CheckBalances", h.CheckBalances("acc7.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc7.1.2", "curr1"))

	})

	t.Run("CheckStripeStatus", func(t *testing.T) {
		st, err := stripeProvider.GetPaymentIntent(extOrderID)
		require.NoError(t, err)
		require.EqualValues(t, 1000, st.Amount)
		require.EqualValues(t, 1000, st.AmountReceived)
		require.NotNil(t, st.Charges)
		require.Len(t, st.Charges.Data, 1)
		require.NotNil(t, st.Charges.Data[0])
		require.EqualValues(t, 1000, st.Charges.Data[0].Amount)
		require.EqualValues(t, 1000, st.Charges.Data[0].AmountRefunded)
		require.EqualValues(t, pkgStripe.PaymentIntentStatusSucceeded, st.Status)
		require.True(t, st.Charges.Data[0].Refunded)
		require.Len(t, st.Charges.Data[0].Refunds.Data, 2)
		require.EqualValues(t, 400, st.Charges.Data[0].Refunds.Data[0].Amount)
		require.EqualValues(t, pkgStripe.RefundStatusSucceeded, st.Charges.Data[0].Refunds.Data[0].Status)
		require.EqualValues(t, 600, st.Charges.Data[0].Refunds.Data[1].Amount)
		require.EqualValues(t, pkgStripe.RefundStatusSucceeded, st.Charges.Data[0].Refunds.Data[1].Status)
	})
}
