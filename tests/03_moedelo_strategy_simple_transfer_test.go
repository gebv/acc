package tests

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gebv/acca/api"
	"github.com/gebv/acca/provider/moedelo"
	"github.com/stretchr/testify/require"
)

func Test03_01MoedeloStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc3.1.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc3.1.2", "curr1"))

	md := moedelo.NewProvider(
		nil,
		moedelo.Config{
			EntrypointURL: os.Getenv("MOEDELO_ENTRYPOINT_URL"),
			Token:         os.Getenv("MOEDELO_TOKEN"),
		},
		nil,
	)
	kontragentID, err := md.CreateKontragent("7720239606", "Иванов Иван Иванович")
	require.NoError(t, err)
	require.NotEmpty(t, kontragentID)

	t.Run("RechargeFromInvoice", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		Meta := map[string]string{
			"title":         "Сервис в тесте.",
			"kontragent_id": strconv.FormatInt(*kontragentID, 10),
			"count":         "1",
			"price":         "10.00",
			"unit":          "шт",
		}
		meta, err := json.Marshal(&Meta)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"moedelo",
			1000,
			&meta,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc3.1.1",
					"acc3.1.2",
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

		t.Run("CheckBalances", h.CheckBalances("acc3.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc3.1.2", "curr1"))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", moedelo.NotPaid.String(), api.TxStatus_AUTH_TX))

		var billID int64
		t.Run("UpdateBill", func(t *testing.T) {
			billID, err = strconv.ParseInt(h.GetTxProviderID("tx1"), 10, 64)
			require.NoError(t, err)
			status := moedelo.PartiallyPaid
			err = md.UpdateBill(billID, *kontragentID, time.Now(), []moedelo.SalesDocumentItemModel{
				{
					Name:    Meta["title"],
					Count:   1,
					Unit:    Meta["unit"],
					Type:    moedelo.Service,
					Price:   10.00,
					NdsType: moedelo.Nds0,
				},
			},
				&status,
			)
			require.NoError(t, err)
		})

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", moedelo.PartiallyPaid.String(), api.TxStatus_HOLD_TX))

		t.Run("GetAccountByKey", func(t *testing.T) {
			bill, err := md.GetBill(billID)
			require.NoError(t, err)
			require.NotNil(t, bill)
			require.EqualValues(t, moedelo.PartiallyPaid, bill.Status)
		})

		h.BalanceInc("acc3.1.1", 1000)
		h.AcceptedBalanceInc("acc3.1.1", 1000)
		h.BalanceInc("acc3.1.2", 1000)
		h.AcceptedBalanceInc("acc3.1.2", 1000)

		t.Run("UpdateBill", func(t *testing.T) {
			status := moedelo.Paid
			err = md.UpdateBill(billID, *kontragentID, time.Now(), []moedelo.SalesDocumentItemModel{
				{
					Name:    Meta["title"],
					Count:   1,
					Unit:    Meta["unit"],
					Type:    moedelo.Service,
					Price:   10.00,
					NdsType: moedelo.Nds0,
				},
			},
				&status,
			)
			require.NoError(t, err)
		})

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckTransaction", h.CheckTransaction("tx1", api.TxStatus_ACCEPTED_TX))

		t.Run("CheckBalances", h.CheckBalances("acc3.1.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc3.1.2", "curr1"))

		t.Run("GetAccountByKey", func(t *testing.T) {
			bill, err := md.GetBill(billID)
			require.NoError(t, err)
			require.NotNil(t, bill)
			require.EqualValues(t, moedelo.Paid, bill.Status)
		})

	})
}

func Test03_02MoedeloStrategy(t *testing.T) {

	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc3.2.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc3.2.2", "curr1"))

	md := moedelo.NewProvider(
		nil,
		moedelo.Config{
			EntrypointURL: os.Getenv("MOEDELO_ENTRYPOINT_URL"),
			Token:         os.Getenv("MOEDELO_TOKEN"),
		},
		nil,
	)
	kontragentID, err := md.CreateKontragent("7720239606", "Иванов Иван Иванович")
	require.NoError(t, err)
	require.NotEmpty(t, kontragentID)

	t.Run("RechargeFromInvoice", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		Meta := map[string]string{
			"title":         "Сервис в тесте.",
			"kontragent_id": strconv.FormatInt(*kontragentID, 10),
			"count":         "1",
			"price":         "1200",
			"unit":          "шт",
		}
		meta, err := json.Marshal(&Meta)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"moedelo",
			1000,
			&meta,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc3.2.1",
					"acc3.2.2",
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

		t.Run("CheckBalances", h.CheckBalances("acc3.2.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc3.2.2", "curr1"))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", moedelo.NotPaid.String(), api.TxStatus_AUTH_TX))

		var billID int64
		t.Run("UpdateBill", func(t *testing.T) {
			billID, err = strconv.ParseInt(h.GetTxProviderID("tx1"), 10, 64)
			require.NoError(t, err)
			status := moedelo.PartiallyPaid
			err = md.UpdateBill(billID, *kontragentID, time.Now(), []moedelo.SalesDocumentItemModel{
				{
					Name:    Meta["title"],
					Count:   1,
					Unit:    Meta["unit"],
					Type:    moedelo.Service,
					Price:   10.00,
					NdsType: moedelo.Nds0,
				},
			},
				&status,
			)
			require.NoError(t, err)
		})

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_WAIT_I))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", moedelo.PartiallyPaid.String(), api.TxStatus_HOLD_TX))

		t.Run("RejectInvoice", h.RejectInvoice("inv1"))

		// Так как отмена транзакции проходит через 2 запроса.
		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_REJECTED_TX))
		t.Run("WaitTransaction", h.WaitTransaction("tx1", api.TxStatus_REJECTED_TX))

		t.Run("CheckTransaction", h.CheckTransaction("tx1", api.TxStatus_REJECTED_TX))

		t.Run("CheckBalances", h.CheckBalances("acc3.2.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc3.2.2", "curr1"))
	})
}

func Test03_03MoedeloStrategy(t *testing.T) {
	h := NewHelperData(t)

	t.Run("CreateCurrency", h.CreateCurrency("curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc3.3.1", "curr1"))

	t.Run("CreateAccount", h.CreateAccount("acc3.3.2", "curr1"))

	md := moedelo.NewProvider(
		nil,
		moedelo.Config{
			EntrypointURL: os.Getenv("MOEDELO_ENTRYPOINT_URL"),
			Token:         os.Getenv("MOEDELO_TOKEN"),
		},
		nil,
	)
	kontragentID, err := md.CreateKontragent("7720239606", "Иванов Иван Иванович")
	require.NoError(t, err)
	require.NotEmpty(t, kontragentID)

	t.Run("RechargeFromInvoice", func(t *testing.T) {

		t.Run("NewInvoice", h.NewInvoice("inv1", "simple", nil))

		Meta := map[string]string{
			"title":         "Сервис в тесте.",
			"kontragent_id": strconv.FormatInt(*kontragentID, 10),
			"count":         "1",
			"price":         "10.00",
			"unit":          "шт",
		}
		meta, err := json.Marshal(&Meta)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", h.AddTransactionToInvoice(
			"inv1",
			"tx1",
			"moedelo",
			1000,
			&meta,
			[]*api.AddTransactionToInvoiceRequest_Oper{
				h.CreateOperation(
					"acc3.3.1",
					"acc3.3.2",
					"",
					false,
					api.OperStrategy_RECHARGE_OPS,
					1000,
				),
				h.CreateOperation(
					"acc3.3.1",
					"acc3.3.2",
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

		t.Run("CheckBalances", h.CheckBalances("acc3.3.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc3.3.2", "curr1"))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", moedelo.NotPaid.String(), api.TxStatus_AUTH_TX))

		h.BalanceInc("acc3.3.1", 1000)
		h.BalanceDec("acc3.3.1", 100)
		h.BalanceInc("acc3.3.2", 1000)
		h.BalanceDec("acc3.3.2", 100)
		h.AcceptedBalanceInc("acc3.3.1", 1000)
		h.AcceptedBalanceDec("acc3.3.1", 100)
		h.AcceptedBalanceInc("acc3.3.2", 1000)
		h.AcceptedBalanceDec("acc3.3.2", 100)

		var billID int64
		t.Run("UpdateBill", func(t *testing.T) {
			billID, err = strconv.ParseInt(h.GetTxProviderID("tx1"), 10, 64)
			require.NoError(t, err)
			status := moedelo.Paid
			err = md.UpdateBill(billID, *kontragentID, time.Now(), []moedelo.SalesDocumentItemModel{
				{
					Name:    Meta["title"],
					Count:   1,
					Unit:    Meta["unit"],
					Type:    moedelo.Service,
					Price:   10.00,
					NdsType: moedelo.Nds0,
				},
			},
				&status,
			)
			require.NoError(t, err)
		})

		t.Run("WaitInvoice", h.WaitInvoice("inv1", api.InvoiceStatus_ACCEPTED_I))

		t.Run("CheckTransactionWithProvider", h.CheckTransactionWithProvider("tx1", moedelo.Paid.String(), api.TxStatus_ACCEPTED_TX))

		t.Run("CheckBalances", h.CheckBalances("acc3.3.1", "curr1"))

		t.Run("CheckBalances", h.CheckBalances("acc3.3.2", "curr1"))

	})
}
