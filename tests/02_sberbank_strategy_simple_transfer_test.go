package tests

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/gebv/acca/api"
	"github.com/gebv/acca/engine/strategies/invoices/simple"
	"github.com/gebv/acca/engine/strategies/transactions/sberbank"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func Test02_01SberbankStrategy(t *testing.T) {
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
			Key:        "acc2.1.1",
			CurrencyId: currID,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		accID1 = res.GetAccId()
	})
	t.Run("GetAccountByKey", func(t *testing.T) {
		res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
			CurrId: currID,
			Key:    "acc2.1.1",
		})
		require.NoError(t, err)
		require.NotNil(t, res.GetAccount())
		require.EqualValues(t, accID1, res.GetAccount().GetAccId())
		balance1 = res.GetAccount().GetBalance()
		balanceAccepted1 = res.GetAccount().GetBalanceAccepted()

	})
	t.Run("CreateAccount", func(t *testing.T) {
		res, err := accC.CreateAccount(authCtx, &api.CreateAccountRequest{
			Key:        "acc2.1.2",
			CurrencyId: currID,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		accID2 = res.GetAccId()
	})
	t.Run("GetAccountByKey", func(t *testing.T) {
		res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
			CurrId: currID,
			Key:    "acc2.1.2",
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
			"callback":    "https://ya.ru",
			"description": "",
			"email":       "test@mail.ru",
		}
		meta, err := json.Marshal(&Meta)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&sberbank.Strategy{}).Name().String(),
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

		time.Sleep(15 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc2.1.1",
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
				Key:    "acc2.1.2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})

		var URL string
		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_AUTH_TX, res.GetTx().GetStatus())
			require.NotNil(t, res.GetTx().GetProviderOperStatus())
			require.EqualValues(t, "CREATED", *res.GetTx().GetProviderOperStatus())
			require.NotNil(t, res.GetTx().GetProviderOperUrl())
			URL = *res.GetTx().GetProviderOperUrl()
		})
		sendCardDataInSberbank(t, URL)
		time.Sleep(15 * time.Second)
		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_HOLD_TX, res.GetTx().GetStatus())
			require.NotNil(t, res.GetTx().GetProviderOperStatus())
			require.EqualValues(t, "APPROVED", *res.GetTx().GetProviderOperStatus())
		})
		t.Run("AcceptInvoice", func(t *testing.T) {
			balance1 += 1000
			balance2 += 1000
			balanceAccepted1 += 1000
			balanceAccepted2 += 1000
			_, err := invC.AcceptInvoice(authCtx, &api.AcceptInvoiceRequest{
				InvoiceId: invID,
			})
			require.NoError(t, err)
		})
		time.Sleep(15 * time.Second)
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
				Key:    "acc2.1.1",
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
				Key:    "acc2.1.2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
	})
}

func Test02_02SberbankStrategy(t *testing.T) {
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
			Key:        "acc2.2.1",
			CurrencyId: currID,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		accID1 = res.GetAccId()
	})
	t.Run("GetAccountByKey", func(t *testing.T) {
		res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
			CurrId: currID,
			Key:    "acc2.2.1",
		})
		require.NoError(t, err)
		require.NotNil(t, res.GetAccount())
		require.EqualValues(t, accID1, res.GetAccount().GetAccId())
		balance1 = res.GetAccount().GetBalance()
		balanceAccepted1 = res.GetAccount().GetBalanceAccepted()

	})
	t.Run("CreateAccount", func(t *testing.T) {
		res, err := accC.CreateAccount(authCtx, &api.CreateAccountRequest{
			Key:        "acc2.2.2",
			CurrencyId: currID,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		accID2 = res.GetAccId()
	})
	t.Run("GetAccountByKey", func(t *testing.T) {
		res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
			CurrId: currID,
			Key:    "acc2.2.2",
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
			"callback":    "https://ya.ru",
			"description": "",
			"email":       "test@mail.ru",
		}
		meta, err := json.Marshal(&Meta)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&sberbank.Strategy{}).Name().String(),
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

		time.Sleep(15 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc2.2.1",
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
				Key:    "acc2.2.2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})

		var URL string
		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_AUTH_TX, res.GetTx().GetStatus())
			require.NotNil(t, res.GetTx().GetProviderOperStatus())
			require.EqualValues(t, "CREATED", *res.GetTx().GetProviderOperStatus())
			require.NotNil(t, res.GetTx().GetProviderOperUrl())
			URL = *res.GetTx().GetProviderOperUrl()
		})
		sendCardDataInSberbank(t, URL)
		time.Sleep(15 * time.Second)
		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_HOLD_TX, res.GetTx().GetStatus())
			require.NotNil(t, res.GetTx().GetProviderOperStatus())
			require.EqualValues(t, "APPROVED", *res.GetTx().GetProviderOperStatus())
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
				Key:    "acc2.2.1",
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
				Key:    "acc2.2.2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})

	})
}

func Test02_03SberbankStrategy(t *testing.T) {
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
			Key:        "acc2.3.1",
			CurrencyId: currID,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		accID1 = res.GetAccId()
	})
	t.Run("GetAccountByKey", func(t *testing.T) {
		res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
			CurrId: currID,
			Key:    "acc2.3.1",
		})
		require.NoError(t, err)
		require.NotNil(t, res.GetAccount())
		require.EqualValues(t, accID1, res.GetAccount().GetAccId())
		balance1 = res.GetAccount().GetBalance()
		balanceAccepted1 = res.GetAccount().GetBalanceAccepted()

	})
	t.Run("CreateAccount", func(t *testing.T) {
		res, err := accC.CreateAccount(authCtx, &api.CreateAccountRequest{
			Key:        "acc2.3.2",
			CurrencyId: currID,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		accID2 = res.GetAccId()
	})
	t.Run("GetAccountByKey", func(t *testing.T) {
		res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
			CurrId: currID,
			Key:    "acc2.3.2",
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
			"callback":    "https://ya.ru",
			"description": "",
			"email":       "test@mail.ru",
		}
		meta, err := json.Marshal(&Meta)
		require.NoError(t, err)

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&sberbank.Strategy{}).Name().String(),
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

		time.Sleep(15 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc2.3.1",
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
				Key:    "acc2.3.2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})

		var URL string
		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_AUTH_TX, res.GetTx().GetStatus())
			require.NotNil(t, res.GetTx().GetProviderOperStatus())
			require.EqualValues(t, "CREATED", *res.GetTx().GetProviderOperStatus())
			require.NotNil(t, res.GetTx().GetProviderOperUrl())
			URL = *res.GetTx().GetProviderOperUrl()
		})
		balance1 += 1000
		balance1 -= 100
		balance2 += 1000
		balance2 -= 100
		balanceAccepted1 += 1000
		balanceAccepted1 -= 100
		balanceAccepted2 += 1000
		balanceAccepted2 -= 100
		sendCardDataInSberbank(t, URL)
		time.Sleep(10 * time.Second)
		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_ACCEPTED_TX, res.GetTx().GetStatus())
			require.NotNil(t, res.GetTx().GetProviderOperStatus())
			require.EqualValues(t, "DEPOSITED", *res.GetTx().GetProviderOperStatus())
		})
		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc2.3.1",
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
				Key:    "acc2.3.2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
	})
}

func sendCardDataInSberbank(t *testing.T, URL string) {
	// https://3dsec.sberbank.ru/payment/merchants/sbersafe/payment_ru.html?mdOrder=ebc0d85c-42e9-7593-96af-650104b2e43b
	t.Run("SendCardDataInSberbank", func(t *testing.T) {
		if len(URL) < 37 {
			t.Fatal("URL: ", URL)
		}
		c := http.Client{
			Transport: http.DefaultTransport,
			Timeout:   10 * time.Second,
		}
		resp, err := c.Post(
			"https://3dsec.sberbank.ru/payment/rest/processform.do?MDORDER="+
				URL[len(URL)-36:]+
				"&$PAN=5555555555555599&$CVC=123&MM=12&YYYY=2019&language=ru&TEXT=CARDHOLDER+NAME",
			"",
			nil,
		)
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		errCode := &struct {
			ErrorCode int64  `json:"errorCode"`
			Redirect  string `json:"redirect"`
		}{}
		err = json.Unmarshal(body, errCode)
		require.NoError(t, err)
		require.EqualValues(t, 0, errCode.ErrorCode)
		getSberbankWebhook(t, errCode.Redirect)
	})
}

func getSberbankWebhook(t *testing.T, URL string) {
	t.Run("GetSberbankWebhook", func(t *testing.T) {
		c := http.Client{
			Transport: http.DefaultTransport,
			Timeout:   10 * time.Second,
		}
		t.Log("URL: ", URL)
		resp, err := c.Get("http://localhost:10003/webhook/sberbank" + URL[len("localhost"):])
		require.NoError(t, err)
		defer resp.Body.Close()
	})
}
