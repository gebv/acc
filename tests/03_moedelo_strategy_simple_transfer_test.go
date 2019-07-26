package tests

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gebv/acca/api"
	"github.com/gebv/acca/engine/strategies/invoices/simple"
	tmoedelo "github.com/gebv/acca/engine/strategies/transactions/moedelo"
	"github.com/gebv/acca/provider/moedelo"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func Test03_01MoedeloStrategy(t *testing.T) {
	accC := api.NewAccountsClient(Conn)
	invC := api.NewInvoicesClient(Conn)
	authCtx := metadata.NewOutgoingContext(Ctx, metadata.New(map[string]string{}))
	var currID int64
	var accID1 int64
	var balance1 int64
	var balanceAccepted1 int64
	var accID2 int64
	var balance2 int64
	var balanceAccepted2 int64
	var invID int64
	var trID int64

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

	t.Run("CreateCurrency", func(t *testing.T) {
		res, err := accC.CreateCurrency(authCtx, &api.CreateCurrencyRequest{
			Key: "curr1",
		})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		currID = res.GetCurrencyId()
	})
	t.Run("GetCurrency", func(t *testing.T) {
		res, err := accC.GetCurrency(authCtx, &api.GetCurrencyRequest{
			Key: "curr1",
		})
		require.NoError(t, err)
		require.NotNil(t, res.GetCurrency())
		require.EqualValues(t, currID, res.GetCurrency().GetCurrId())
	})
	t.Run("CreateAccount", func(t *testing.T) {
		res, err := accC.CreateAccount(authCtx, &api.CreateAccountRequest{
			Key:        "acc3.1.1",
			CurrencyId: currID,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		accID1 = res.GetAccId()
	})
	t.Run("GetAccountByKey", func(t *testing.T) {
		res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
			CurrId: currID,
			Key:    "acc3.1.1",
		})
		require.NoError(t, err)
		require.NotNil(t, res.GetAccount())
		require.EqualValues(t, accID1, res.GetAccount().GetAccId())
		balance1 = res.GetAccount().GetBalance()
		balanceAccepted1 = res.GetAccount().GetBalanceAccepted()

	})
	t.Run("CreateAccount", func(t *testing.T) {
		res, err := accC.CreateAccount(authCtx, &api.CreateAccountRequest{
			Key:        "acc3.1.2",
			CurrencyId: currID,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		accID2 = res.GetAccId()
	})
	t.Run("GetAccountByKey", func(t *testing.T) {
		res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
			CurrId: currID,
			Key:    "acc3.1.2",
		})
		require.NoError(t, err)
		require.NotNil(t, res.GetAccount())
		require.EqualValues(t, accID2, res.GetAccount().GetAccId())
		balance2 = res.GetAccount().GetBalance()
		balanceAccepted2 = res.GetAccount().GetBalanceAccepted()

	})

	t.Run("RechargeFromInvoice", func(t *testing.T) {

		t.Run("NewInvoice", func(t *testing.T) {
			res, err := invC.NewInvoice(authCtx, &api.NewInvoiceRequest{
				Key:      "inv1",
				Strategy: (&simple.Strategy{}).Name().String(),
				Amount:   1000,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetInvoiceId())
			invID = res.GetInvoiceId()
		})

		t.Run("GetInvoiceByID", func(t *testing.T) {
			res, err := invC.GetInvoiceByID(authCtx, &api.GetInvoiceByIDRequest{
				InvoiceId: invID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetInvoice())
			require.EqualValues(t, api.InvoiceStatus_DRAFT_I, res.GetInvoice().GetStatus())
		})

		Meta := map[string]string{
			"title":         "Сервис в тесте.",
			"kontragent_id": strconv.FormatInt(*kontragentID, 10),
			"count":         "1",
			"price":         "10.00",
			"unit":          "шт",
		}
		meta, err := json.Marshal(&Meta)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tmoedelo.Strategy{}).Name().String(),
				Amount:    1000,
				Meta:      &meta,
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID1,
						DstAccId:  accID2,
						Strategy:  api.OperStrategy_RECHARGE_OPS,
						Amount:    1000,
						Meta:      nil,
						Hold:      true,
						HoldAccId: nil,
					},
				},
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTxId())
			trID = res.GetTxId()
		})

		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_DRAFT_TX, res.GetTx().GetStatus())
		})

		t.Run("AuthInvoice", func(t *testing.T) {
			_, err := invC.AuthInvoice(authCtx, &api.AuthInvoiceRequest{
				InvoiceId: invID,
			})
			require.NoError(t, err)
		})

		time.Sleep(45 * time.Second) // listener updated every 30 second

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc3.1.1",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID1, res.GetAccount().GetAccId())
			require.EqualValues(t, balance1, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted1, res.GetAccount().GetBalanceAccepted())

		})

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc3.1.2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})

		var URL string
		var billID int64
		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_AUTH_TX, res.GetTx().GetStatus())
			require.NotNil(t, res.GetTx().GetProviderOperStatus())
			require.EqualValues(t, moedelo.NotPaid.String(), *res.GetTx().GetProviderOperStatus())
			require.NotNil(t, res.GetTx().GetProviderOperUrl())
			URL = *res.GetTx().GetProviderOperUrl()
			require.NotNil(t, res.GetTx().GetProviderOperId())
			billID, err = strconv.ParseInt(*res.GetTx().GetProviderOperId(), 10, 64)
			require.NoError(t, err)
		})

		t.Run("UpdateBill", func(t *testing.T) {
			t.Log("MOE DELO URL: ", URL)
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
		})
		time.Sleep(45 * time.Second) // listener updated every 30 second

		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_HOLD_TX, res.GetTx().GetStatus())
			require.NotNil(t, res.GetTx().GetProviderOperStatus())
			require.EqualValues(t, moedelo.PartiallyPaid.String(), *res.GetTx().GetProviderOperStatus())
		})

		t.Run("GetAccountByKey", func(t *testing.T) {
			bill, err := md.GetBill(billID)
			require.NoError(t, err)
			require.NotNil(t, bill)
			require.EqualValues(t, moedelo.PartiallyPaid, bill.Status)
		})

		t.Run("UpdateBill", func(t *testing.T) {
			balance1 += 1000
			balance2 += 1000
			balanceAccepted1 += 1000
			balanceAccepted2 += 1000
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
		})
		time.Sleep(45 * time.Second) // listener updated every 30 second

		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_ACCEPTED_TX, res.GetTx().GetStatus())
		})

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc3.1.1",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID1, res.GetAccount().GetAccId())
			require.EqualValues(t, balance1, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted1, res.GetAccount().GetBalanceAccepted())

		})

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc3.1.2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})

		t.Run("GetAccountByKey", func(t *testing.T) {
			bill, err := md.GetBill(billID)
			require.NoError(t, err)
			require.NotNil(t, bill)
			require.EqualValues(t, moedelo.Paid, bill.Status)
		})

	})
}

func Test03_02MoedeloStrategy(t *testing.T) {
	accC := api.NewAccountsClient(Conn)
	invC := api.NewInvoicesClient(Conn)
	authCtx := metadata.NewOutgoingContext(Ctx, metadata.New(map[string]string{}))
	var currID int64
	var accID1 int64
	var balance1 int64
	var balanceAccepted1 int64
	var accID2 int64
	var balance2 int64
	var balanceAccepted2 int64
	var invID int64
	var trID int64

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

	t.Run("CreateCurrency", func(t *testing.T) {
		res, err := accC.CreateCurrency(authCtx, &api.CreateCurrencyRequest{
			Key: "curr1",
		})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		currID = res.GetCurrencyId()
	})
	t.Run("GetCurrency", func(t *testing.T) {
		res, err := accC.GetCurrency(authCtx, &api.GetCurrencyRequest{
			Key: "curr1",
		})
		require.NoError(t, err)
		require.NotNil(t, res.GetCurrency())
		require.EqualValues(t, currID, res.GetCurrency().GetCurrId())
	})
	t.Run("CreateAccount", func(t *testing.T) {
		res, err := accC.CreateAccount(authCtx, &api.CreateAccountRequest{
			Key:        "acc3.2.1",
			CurrencyId: currID,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		accID1 = res.GetAccId()
	})
	t.Run("GetAccountByKey", func(t *testing.T) {
		res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
			CurrId: currID,
			Key:    "acc3.2.1",
		})
		require.NoError(t, err)
		require.NotNil(t, res.GetAccount())
		require.EqualValues(t, accID1, res.GetAccount().GetAccId())
		balance1 = res.GetAccount().GetBalance()
		balanceAccepted1 = res.GetAccount().GetBalanceAccepted()

	})
	t.Run("CreateAccount", func(t *testing.T) {
		res, err := accC.CreateAccount(authCtx, &api.CreateAccountRequest{
			Key:        "acc3.2.2",
			CurrencyId: currID,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		accID2 = res.GetAccId()
	})
	t.Run("GetAccountByKey", func(t *testing.T) {
		res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
			CurrId: currID,
			Key:    "acc3.2.2",
		})
		require.NoError(t, err)
		require.NotNil(t, res.GetAccount())
		require.EqualValues(t, accID2, res.GetAccount().GetAccId())
		balance2 = res.GetAccount().GetBalance()
		balanceAccepted2 = res.GetAccount().GetBalanceAccepted()

	})

	t.Run("RechargeFromInvoice", func(t *testing.T) {

		t.Run("NewInvoice", func(t *testing.T) {
			res, err := invC.NewInvoice(authCtx, &api.NewInvoiceRequest{
				Key:      "inv1",
				Strategy: (&simple.Strategy{}).Name().String(),
				Amount:   1000,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetInvoiceId())
			invID = res.GetInvoiceId()
		})

		t.Run("GetInvoiceByID", func(t *testing.T) {
			res, err := invC.GetInvoiceByID(authCtx, &api.GetInvoiceByIDRequest{
				InvoiceId: invID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetInvoice())
			require.EqualValues(t, api.InvoiceStatus_DRAFT_I, res.GetInvoice().GetStatus())
		})
		Meta := map[string]string{
			"title":         "Сервис в тесте.",
			"kontragent_id": strconv.FormatInt(*kontragentID, 10),
			"count":         "1",
			"price":         "1200",
			"unit":          "шт",
		}
		meta, err := json.Marshal(&Meta)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tmoedelo.Strategy{}).Name().String(),
				Amount:    1000,
				Meta:      &meta,
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID1,
						DstAccId:  accID2,
						Strategy:  api.OperStrategy_RECHARGE_OPS,
						Amount:    1000,
						Meta:      nil,
						Hold:      true,
						HoldAccId: nil,
					},
				},
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTxId())
			trID = res.GetTxId()
		})

		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_DRAFT_TX, res.GetTx().GetStatus())
		})

		t.Run("AuthInvoice", func(t *testing.T) {
			_, err := invC.AuthInvoice(authCtx, &api.AuthInvoiceRequest{
				InvoiceId: invID,
			})
			require.NoError(t, err)
		})

		time.Sleep(45 * time.Second) // listener updated every 30 second

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc3.2.1",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID1, res.GetAccount().GetAccId())
			require.EqualValues(t, balance1, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted1, res.GetAccount().GetBalanceAccepted())

		})

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc3.2.2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})

		var URL string
		var billID int64
		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_AUTH_TX, res.GetTx().GetStatus())
			require.NotNil(t, res.GetTx().GetProviderOperStatus())
			require.EqualValues(t, moedelo.NotPaid.String(), *res.GetTx().GetProviderOperStatus())
			require.NotNil(t, res.GetTx().GetProviderOperUrl())
			URL = *res.GetTx().GetProviderOperUrl()
			require.NotNil(t, res.GetTx().GetProviderOperId())
			billID, err = strconv.ParseInt(*res.GetTx().GetProviderOperId(), 10, 64)
			require.NoError(t, err)
		})

		t.Run("UpdateBill", func(t *testing.T) {
			t.Log("MOE DELO URL: ", URL)
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
		})
		time.Sleep(45 * time.Second) // listener updated every 30 second

		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_HOLD_TX, res.GetTx().GetStatus())
			require.NotNil(t, res.GetTx().GetProviderOperStatus())
			require.EqualValues(t, moedelo.PartiallyPaid.String(), *res.GetTx().GetProviderOperStatus())
		})
		t.Run("RejectInvoice", func(t *testing.T) {
			_, err := invC.RejectInvoice(authCtx, &api.RejectInvoiceRequest{
				InvoiceId: invID,
			})
			require.NoError(t, err)
		})
		time.Sleep(10 * time.Second)
		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_REJECTED_TX, res.GetTx().GetStatus())
		})
		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc3.2.1",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID1, res.GetAccount().GetAccId())
			require.EqualValues(t, balance1, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted1, res.GetAccount().GetBalanceAccepted())

		})

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc3.2.2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
	})
}

func Test03_03MoedeloStrategy(t *testing.T) {
	accC := api.NewAccountsClient(Conn)
	invC := api.NewInvoicesClient(Conn)
	authCtx := metadata.NewOutgoingContext(Ctx, metadata.New(map[string]string{}))
	var currID int64
	var accID1 int64
	var balance1 int64
	var balanceAccepted1 int64
	var accID2 int64
	var balance2 int64
	var balanceAccepted2 int64
	var invID int64
	var trID int64

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

	t.Run("CreateCurrency", func(t *testing.T) {
		res, err := accC.CreateCurrency(authCtx, &api.CreateCurrencyRequest{
			Key: "curr1",
		})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		currID = res.GetCurrencyId()
	})
	t.Run("GetCurrency", func(t *testing.T) {
		res, err := accC.GetCurrency(authCtx, &api.GetCurrencyRequest{
			Key: "curr1",
		})
		require.NoError(t, err)
		require.NotNil(t, res.GetCurrency())
		require.EqualValues(t, currID, res.GetCurrency().GetCurrId())
	})
	t.Run("CreateAccount", func(t *testing.T) {
		res, err := accC.CreateAccount(authCtx, &api.CreateAccountRequest{
			Key:        "acc3.3.1",
			CurrencyId: currID,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		accID1 = res.GetAccId()
	})
	t.Run("GetAccountByKey", func(t *testing.T) {
		res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
			CurrId: currID,
			Key:    "acc3.3.1",
		})
		require.NoError(t, err)
		require.NotNil(t, res.GetAccount())
		require.EqualValues(t, accID1, res.GetAccount().GetAccId())
		balance1 = res.GetAccount().GetBalance()
		balanceAccepted1 = res.GetAccount().GetBalanceAccepted()

	})
	t.Run("CreateAccount", func(t *testing.T) {
		res, err := accC.CreateAccount(authCtx, &api.CreateAccountRequest{
			Key:        "acc3.3.2",
			CurrencyId: currID,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		accID2 = res.GetAccId()
	})
	t.Run("GetAccountByKey", func(t *testing.T) {
		res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
			CurrId: currID,
			Key:    "acc3.3.2",
		})
		require.NoError(t, err)
		require.NotNil(t, res.GetAccount())
		require.EqualValues(t, accID2, res.GetAccount().GetAccId())
		balance2 = res.GetAccount().GetBalance()
		balanceAccepted2 = res.GetAccount().GetBalanceAccepted()

	})

	t.Run("RechargeFromInvoice", func(t *testing.T) {

		t.Run("NewInvoice", func(t *testing.T) {
			res, err := invC.NewInvoice(authCtx, &api.NewInvoiceRequest{
				Key:      "inv1",
				Strategy: (&simple.Strategy{}).Name().String(),
				Amount:   1000,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetInvoiceId())
			invID = res.GetInvoiceId()
		})

		t.Run("GetInvoiceByID", func(t *testing.T) {
			res, err := invC.GetInvoiceByID(authCtx, &api.GetInvoiceByIDRequest{
				InvoiceId: invID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetInvoice())
			require.EqualValues(t, api.InvoiceStatus_DRAFT_I, res.GetInvoice().GetStatus())
		})

		Meta := map[string]string{
			"title":         "Сервис в тесте.",
			"kontragent_id": strconv.FormatInt(*kontragentID, 10),
			"count":         "1",
			"price":         "10.00",
			"unit":          "шт",
		}
		meta, err := json.Marshal(&Meta)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tmoedelo.Strategy{}).Name().String(),
				Amount:    1000,
				Meta:      &meta,
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID1,
						DstAccId:  accID2,
						Strategy:  api.OperStrategy_RECHARGE_OPS,
						Amount:    1000,
						Meta:      nil,
						Hold:      false,
						HoldAccId: nil,
					},
					{
						SrcAccId:  accID1,
						DstAccId:  accID2,
						Strategy:  api.OperStrategy_WITHDRAW_OPS,
						Amount:    100,
						Meta:      nil,
						Hold:      false,
						HoldAccId: nil,
					},
				},
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTxId())
			trID = res.GetTxId()
		})

		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_DRAFT_TX, res.GetTx().GetStatus())
		})

		t.Run("AuthInvoice", func(t *testing.T) {
			_, err := invC.AuthInvoice(authCtx, &api.AuthInvoiceRequest{
				InvoiceId: invID,
			})
			require.NoError(t, err)
		})

		time.Sleep(45 * time.Second) // listener updated every 30 second

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc3.3.1",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID1, res.GetAccount().GetAccId())
			require.EqualValues(t, balance1, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted1, res.GetAccount().GetBalanceAccepted())

		})

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc3.3.2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})

		var URL string
		var billID int64
		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_AUTH_TX, res.GetTx().GetStatus())
			require.NotNil(t, res.GetTx().GetProviderOperStatus())
			require.EqualValues(t, moedelo.NotPaid.String(), *res.GetTx().GetProviderOperStatus())
			require.NotNil(t, res.GetTx().GetProviderOperUrl())
			URL = *res.GetTx().GetProviderOperUrl()
			require.NotNil(t, res.GetTx().GetProviderOperId())
			billID, err = strconv.ParseInt(*res.GetTx().GetProviderOperId(), 10, 64)
			require.NoError(t, err)
		})

		balance1 += 1000
		balance1 -= 100
		balance2 += 1000
		balance2 -= 100
		balanceAccepted1 += 1000
		balanceAccepted1 -= 100
		balanceAccepted2 += 1000
		balanceAccepted2 -= 100

		t.Log("MOE DELO URL: ", URL)

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
		})
		time.Sleep(45 * time.Second) // listener updated every 30 second

		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_ACCEPTED_TX, res.GetTx().GetStatus())
			require.NotNil(t, res.GetTx().GetProviderOperStatus())
			require.EqualValues(t, moedelo.Paid.String(), *res.GetTx().GetProviderOperStatus())
		})
		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc3.3.1",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID1, res.GetAccount().GetAccId())
			require.EqualValues(t, balance1, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted1, res.GetAccount().GetBalanceAccepted())

		})

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc3.3.2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
	})
}
