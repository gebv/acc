package tests

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"github.com/gebv/acca/api"
	"github.com/gebv/acca/provider/sberbank"
	"github.com/stretchr/testify/require"
)

func Test04_01SberbankStrategy(t *testing.T) {
	h := NewHelperData()

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc4.1.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc4.1.2", "curr1"))

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
			"sberbank",
			1000,
			&meta,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc4.1.1",
					"acc4.1.2",
					"",
					false,
					api.OperStrategy_RECHARGE_OPS,
					1000,
				),
			},
		))

		t.Run("AuthInvoice", h.AuthInvoice("inv1"))

		h.Sleep(15)

		t.Run("CheckBalances", h.CheckBalances("acc4.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc4.1.2", "curr1"))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", "CREATED", api.TxStatus_AUTH_TX))

		t.Run("SendCardDataInSberbank", h.SendCardDataInSberbank("tx1"))

		h.Sleep(15)

		h.BalanceInc("acc4.1.1", 1000)
		h.AcceptedBalanceInc("acc4.1.1", 1000)
		h.BalanceInc("acc4.1.2", 1000)
		h.AcceptedBalanceInc("acc4.1.2", 1000)

		t.Run("CheckTransaction", h.CheckTransaction("tx1", api.TxStatus_ACCEPTED_TX))

		t.Run("CheckBalances", h.CheckBalances("acc4.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc4.1.2", "curr1"))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", "DEPOSITED", api.TxStatus_ACCEPTED_TX))

	})

	sberProvider := sberbank.NewProvider(
		nil,
		sberbank.Config{
			EntrypointURL: os.Getenv("SBERBANK_ENTRYPOINT_URL"),
			Token:         os.Getenv("SBERBANK_TOKEN"),
			Password:      os.Getenv("SBERBANK_PASSWORD"),
			UserName:      os.Getenv("SBERBANK_USER_NAME"),
		},
		nil,
	)
	var extOrderID string
	res, err := h.invC.GetTransactionByID(h.authCtx, &api.GetTransactionByIDRequest{
		TxId: h.GetTxID("tx1"),
	})
	require.NoError(t, err)
	require.NotNil(t, res.GetTx().GetProviderOperId())
	extOrderID = *res.GetTx().GetProviderOperId()
	t.Run("CheckSberbankStatus", func(t *testing.T) {
		st, err := sberProvider.GetOrderRawStatus(extOrderID)
		require.NoError(t, err)
		require.EqualValues(t, 1000, st.PaymentAmountInfo.ApprovedAmount)
		require.EqualValues(t, 1000, st.PaymentAmountInfo.DepositedAmount)
		require.EqualValues(t, "DEPOSITED", st.PaymentAmountInfo.PaymentState)
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
			"sberbank_refund",
			600,
			&metaTr,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc4.1.1",
					"acc4.1.2",
					"",
					false,
					api.OperStrategy_WITHDRAW_OPS,
					600,
				),
			},
		))

		t.Run("RejectInvoice", h.RejectInvoice("inv2"))

		h.Sleep(15)

		t.Run("CheckTransaction", h.CheckTransaction("tx2", api.TxStatus_REJECTED_TX))

		t.Run("CheckBalances", h.CheckBalances("acc4.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc4.1.2", "curr1"))

	})

	t.Run("CheckSberbankStatus", func(t *testing.T) {
		st, err := sberProvider.GetOrderRawStatus(extOrderID)
		require.NoError(t, err)
		require.EqualValues(t, 1000, st.PaymentAmountInfo.ApprovedAmount)
		require.EqualValues(t, 1000, st.PaymentAmountInfo.DepositedAmount)
		require.EqualValues(t, "DEPOSITED", st.PaymentAmountInfo.PaymentState)
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
			"sberbank_refund",
			600,
			&metaTr,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc4.1.1",
					"acc4.1.2",
					"",
					false,
					api.OperStrategy_WITHDRAW_OPS,
					600,
				),
			},
		))

		h.BalanceDec("acc4.1.1", 600)
		h.AcceptedBalanceDec("acc4.1.1", 600)
		h.BalanceDec("acc4.1.2", 600)
		h.AcceptedBalanceDec("acc4.1.2", 600)

		t.Run("AuthInvoice", h.AuthInvoice("inv2"))

		h.Sleep(15)

		t.Run("CheckTransaction", h.CheckTransaction("tx2", api.TxStatus_ACCEPTED_TX))

		t.Run("CheckBalances", h.CheckBalances("acc4.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc4.1.2", "curr1"))

	})

	t.Run("CheckSberbankStatus", func(t *testing.T) {
		st, err := sberProvider.GetOrderRawStatus(extOrderID)
		require.NoError(t, err)
		require.EqualValues(t, 1000, st.PaymentAmountInfo.ApprovedAmount)
		require.EqualValues(t, 400, st.PaymentAmountInfo.DepositedAmount)
		require.EqualValues(t, "REFUNDED", st.PaymentAmountInfo.PaymentState)
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
			"sberbank_refund",
			600,
			&metaTr,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc4.1.1",
					"acc4.1.2",
					"",
					false,
					api.OperStrategy_WITHDRAW_OPS,
					600,
				),
			},
		))

		t.Run("AuthInvoice", h.AuthInvoice("inv2"))

		h.Sleep(15)

		t.Run("CheckTransaction", h.CheckTransaction("tx2", api.TxStatus_DRAFT_TX))

		t.Run("CheckBalances", h.CheckBalances("acc4.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc4.1.2", "curr1"))

	})

	t.Run("CheckSberbankStatus", func(t *testing.T) {
		st, err := sberProvider.GetOrderRawStatus(extOrderID)
		require.NoError(t, err)
		require.EqualValues(t, 1000, st.PaymentAmountInfo.ApprovedAmount)
		require.EqualValues(t, 400, st.PaymentAmountInfo.DepositedAmount)
		require.EqualValues(t, "REFUNDED", st.PaymentAmountInfo.PaymentState)
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
			"sberbank_refund",
			400,
			&metaTr,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc4.1.1",
					"acc4.1.2",
					"",
					false,
					api.OperStrategy_WITHDRAW_OPS,
					400,
				),
			},
		))

		h.BalanceDec("acc4.1.1", 400)
		h.AcceptedBalanceDec("acc4.1.1", 400)
		h.BalanceDec("acc4.1.2", 400)
		h.AcceptedBalanceDec("acc4.1.2", 400)

		t.Run("AuthInvoice", h.AuthInvoice("inv2"))

		h.Sleep(15)

		t.Run("CheckTransaction", h.CheckTransaction("tx2", api.TxStatus_ACCEPTED_TX))

		t.Run("CheckBalances", h.CheckBalances("acc4.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc4.1.2", "curr1"))

	})

	t.Run("CheckSberbankStatus", func(t *testing.T) {
		st, err := sberProvider.GetOrderRawStatus(extOrderID)
		require.NoError(t, err)
		require.EqualValues(t, 1000, st.PaymentAmountInfo.ApprovedAmount)
		require.EqualValues(t, 0, st.PaymentAmountInfo.DepositedAmount)
		require.EqualValues(t, "REFUNDED", st.PaymentAmountInfo.PaymentState)
	})
}
