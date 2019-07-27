package tests

import (
	"testing"
	"time"

	"github.com/gebv/acca/api"
	isimple "github.com/gebv/acca/engine/strategies/invoices/simple"
	tsimple "github.com/gebv/acca/engine/strategies/transactions/simple"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func Test01_01SimpleStrategy(t *testing.T) {
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
	var trID2 int64

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
			Key:        "acc1",
			CurrencyId: currID,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		accID1 = res.GetAccId()
	})
	t.Run("GetAccountByKey", func(t *testing.T) {
		res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
			CurrId: currID,
			Key:    "acc1",
		})
		require.NoError(t, err)
		require.NotNil(t, res.GetAccount())
		require.EqualValues(t, accID1, res.GetAccount().GetAccId())
		balance1 = res.GetAccount().GetBalance()
		balanceAccepted1 = res.GetAccount().GetBalanceAccepted()

	})
	t.Run("CreateAccount", func(t *testing.T) {
		res, err := accC.CreateAccount(authCtx, &api.CreateAccountRequest{
			Key:        "acc2",
			CurrencyId: currID,
		})
		require.NoError(t, err)
		require.NotEmpty(t, res)
		accID2 = res.GetAccId()
	})
	t.Run("GetAccountByKey", func(t *testing.T) {
		res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
			CurrId: currID,
			Key:    "acc2",
		})
		require.NoError(t, err)
		require.NotNil(t, res.GetAccount())
		require.EqualValues(t, accID2, res.GetAccount().GetAccId())
		balance2 = res.GetAccount().GetBalance()
		balanceAccepted2 = res.GetAccount().GetBalanceAccepted()

	})

	t.Run("ChangeFromInvoice", func(t *testing.T) {

		t.Run("NewInvoice", func(t *testing.T) {
			res, err := invC.NewInvoice(authCtx, &api.NewInvoiceRequest{
				Key:      "inv1",
				Strategy: (&isimple.Strategy{}).Name().String(),
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

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			balance1 -= 1000
			balanceAccepted1 -= 1000
			balance2 += 1000
			balanceAccepted2 += 1000
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tsimple.Strategy{}).Name().String(),
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID1,
						DstAccId:  accID2,
						Strategy:  api.OperStrategy_SIMPLE_OPS,
						Amount:    1000,
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

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
	})

	t.Run("InternalWithHold", func(t *testing.T) {

		t.Run("NewInvoice", func(t *testing.T) {
			res, err := invC.NewInvoice(authCtx, &api.NewInvoiceRequest{
				Key:      "inv1",
				Strategy: (&isimple.Strategy{}).Name().String(),
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

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tsimple.Strategy{}).Name().String(),
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID1,
						DstAccId:  accID2,
						Strategy:  api.OperStrategy_SIMPLE_OPS,
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
			balance1 -= 1000
			_, err := invC.AuthInvoice(authCtx, &api.AuthInvoiceRequest{
				InvoiceId: invID,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
		t.Run("BalanceChanges", func(t *testing.T) {
			res, err := accC.BalanceChanges(authCtx, &api.BalanceChangesRequest{
				Offset: 0,
				Limit:  2,
				AccId:  &accID1,
			})
			require.NoError(t, err)
			require.Len(t, res.GetBalanceChanges(), 2)
			require.EqualValues(t, accID1, res.GetBalanceChanges()[0].GetAccId())
			require.EqualValues(t, balance1, res.GetBalanceChanges()[0].GetBalance())
			require.EqualValues(t, balanceAccepted1, res.GetBalanceChanges()[0].GetBalanceAccepted())
		})
		t.Run("AcceptInvoice", func(t *testing.T) {
			balanceAccepted1 -= 1000
			balance2 += 1000
			balanceAccepted2 += 1000
			_, err := invC.AcceptInvoice(authCtx, &api.AcceptInvoiceRequest{
				InvoiceId: invID,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
		t.Run("BalanceChanges", func(t *testing.T) {
			res, err := accC.BalanceChanges(authCtx, &api.BalanceChangesRequest{
				Offset: 0,
				Limit:  2,
				AccId:  &accID1,
			})
			require.NoError(t, err)
			require.Len(t, res.GetBalanceChanges(), 2)
			require.EqualValues(t, accID1, res.GetBalanceChanges()[1].GetAccId())
			require.NotEqual(t, balance1, res.GetBalanceChanges()[1].GetBalance())
			require.NotEqual(t, balanceAccepted1, res.GetBalanceChanges()[1].GetBalanceAccepted())
			require.EqualValues(t, accID1, res.GetBalanceChanges()[0].GetActualAccount().GetAccId())
			require.EqualValues(t, balance1, res.GetBalanceChanges()[0].GetActualAccount().GetBalance())
			require.EqualValues(t, balanceAccepted1, res.GetBalanceChanges()[0].GetActualAccount().GetBalanceAccepted())
		})
	})

	t.Run("InternalReject", func(t *testing.T) {

		t.Run("NewInvoice", func(t *testing.T) {
			res, err := invC.NewInvoice(authCtx, &api.NewInvoiceRequest{
				Key:      "inv1",
				Strategy: (&isimple.Strategy{}).Name().String(),
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

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tsimple.Strategy{}).Name().String(),
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID1,
						DstAccId:  accID2,
						Strategy:  api.OperStrategy_SIMPLE_OPS,
						Amount:    1000,
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
			_, err := invC.RejectInvoice(authCtx, &api.RejectInvoiceRequest{
				InvoiceId: invID,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
	})

	t.Run("InternalRejectWithHold", func(t *testing.T) {

		t.Run("NewInvoice", func(t *testing.T) {
			res, err := invC.NewInvoice(authCtx, &api.NewInvoiceRequest{
				Key:      "inv1",
				Strategy: (&isimple.Strategy{}).Name().String(),
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

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tsimple.Strategy{}).Name().String(),
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID1,
						DstAccId:  accID2,
						Strategy:  api.OperStrategy_SIMPLE_OPS,
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
			balance1 -= 1000
			_, err := invC.AuthInvoice(authCtx, &api.AuthInvoiceRequest{
				InvoiceId: invID,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
		t.Run("AcceptInvoice", func(t *testing.T) {
			balance1 += 1000
			_, err := invC.RejectInvoice(authCtx, &api.RejectInvoiceRequest{
				InvoiceId: invID,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
	})

	t.Run("ChangeFromTransactions", func(t *testing.T) {

		t.Run("NewInvoice", func(t *testing.T) {
			res, err := invC.NewInvoice(authCtx, &api.NewInvoiceRequest{
				Key:      "inv1",
				Strategy: (&isimple.Strategy{}).Name().String(),
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

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			balance1 -= 1000
			balanceAccepted1 -= 1000
			balance2 += 1000
			balanceAccepted2 += 1000
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tsimple.Strategy{}).Name().String(),
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID1,
						DstAccId:  accID2,
						Strategy:  api.OperStrategy_SIMPLE_OPS,
						Amount:    1000,
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

		t.Run("AuthTx", func(t *testing.T) {
			_, err := invC.AuthTx(authCtx, &api.AuthTxRequest{
				TxId: trID,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
	})

	t.Run("InternalWithHoldFromTransactions", func(t *testing.T) {

		t.Run("NewInvoice", func(t *testing.T) {
			res, err := invC.NewInvoice(authCtx, &api.NewInvoiceRequest{
				Key:      "inv1",
				Strategy: (&isimple.Strategy{}).Name().String(),
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

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tsimple.Strategy{}).Name().String(),
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID1,
						DstAccId:  accID2,
						Strategy:  api.OperStrategy_SIMPLE_OPS,
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

		t.Run("AuthTx", func(t *testing.T) {
			balance1 -= 1000
			_, err := invC.AuthTx(authCtx, &api.AuthTxRequest{
				TxId: trID,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
		t.Run("AcceptInvoice", func(t *testing.T) {
			balanceAccepted1 -= 1000
			balance2 += 1000
			balanceAccepted2 += 1000
			_, err := invC.AcceptTx(authCtx, &api.AcceptTxRequest{
				TxId: trID,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
	})

	t.Run("InternalRejectFromTransaction", func(t *testing.T) {

		t.Run("NewInvoice", func(t *testing.T) {
			res, err := invC.NewInvoice(authCtx, &api.NewInvoiceRequest{
				Key:      "inv1",
				Strategy: (&isimple.Strategy{}).Name().String(),
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

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tsimple.Strategy{}).Name().String(),
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID1,
						DstAccId:  accID2,
						Strategy:  api.OperStrategy_SIMPLE_OPS,
						Amount:    1000,
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

		t.Run("RejectTx", func(t *testing.T) {
			_, err := invC.RejectTx(authCtx, &api.RejectTxRequest{
				TxId: trID,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
	})

	t.Run("InternalRejectWithHoldFromTransaction", func(t *testing.T) {

		t.Run("NewInvoice", func(t *testing.T) {
			res, err := invC.NewInvoice(authCtx, &api.NewInvoiceRequest{
				Key:      "inv1",
				Strategy: (&isimple.Strategy{}).Name().String(),
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

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tsimple.Strategy{}).Name().String(),
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID1,
						DstAccId:  accID2,
						Strategy:  api.OperStrategy_SIMPLE_OPS,
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

		t.Run("AuthTx", func(t *testing.T) {
			balance1 -= 1000
			_, err := invC.AuthTx(authCtx, &api.AuthTxRequest{
				TxId: trID,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
		t.Run("AcceptInvoice", func(t *testing.T) {
			balance1 += 1000
			_, err := invC.RejectTx(authCtx, &api.RejectTxRequest{
				TxId: trID,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
	})

	t.Run("ChangeFromTransactions", func(t *testing.T) {

		t.Run("NewInvoice", func(t *testing.T) {
			res, err := invC.NewInvoice(authCtx, &api.NewInvoiceRequest{
				Key:      "inv1",
				Strategy: (&isimple.Strategy{}).Name().String(),
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

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			key := "1"
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tsimple.Strategy{}).Name().String(),
				Key:       &key,
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID1,
						DstAccId:  accID2,
						Strategy:  api.OperStrategy_SIMPLE_OPS,
						Amount:    1000,
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

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			key := "2"
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tsimple.Strategy{}).Name().String(),
				Key:       &key,
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID2,
						DstAccId:  accID1,
						Strategy:  api.OperStrategy_SIMPLE_OPS,
						Amount:    500,
						Meta:      nil,
						Hold:      false,
						HoldAccId: nil,
					},
				},
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTxId())
			trID2 = res.GetTxId()
		})

		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID2,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_DRAFT_TX, res.GetTx().GetStatus())
		})

		t.Run("AuthTx", func(t *testing.T) {
			balance1 -= 1000
			balanceAccepted1 -= 1000
			balance2 += 1000
			balanceAccepted2 += 1000
			_, err := invC.AuthTx(authCtx, &api.AuthTxRequest{
				TxId: trID,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})

		t.Run("AuthTx", func(t *testing.T) {
			balance1 += 500
			balanceAccepted1 += 500
			balance2 -= 500
			balanceAccepted2 -= 500
			_, err := invC.AuthTx(authCtx, &api.AuthTxRequest{
				TxId: trID2,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
	})

	t.Run("ChangeWithHoldFromTransactions", func(t *testing.T) {

		t.Run("NewInvoice", func(t *testing.T) {
			res, err := invC.NewInvoice(authCtx, &api.NewInvoiceRequest{
				Key:      "inv1",
				Strategy: (&isimple.Strategy{}).Name().String(),
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

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			key := "1"
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tsimple.Strategy{}).Name().String(),
				Key:       &key,
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID1,
						DstAccId:  accID2,
						Strategy:  api.OperStrategy_SIMPLE_OPS,
						Amount:    1000,
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

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			key := "2"
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tsimple.Strategy{}).Name().String(),
				Key:       &key,
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID2,
						DstAccId:  accID1,
						Strategy:  api.OperStrategy_SIMPLE_OPS,
						Amount:    500,
						Meta:      nil,
						Hold:      true,
						HoldAccId: nil,
					},
				},
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTxId())
			trID2 = res.GetTxId()
		})

		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID2,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_DRAFT_TX, res.GetTx().GetStatus())
		})

		t.Run("AuthTx", func(t *testing.T) {
			balance1 -= 1000
			balanceAccepted1 -= 1000
			balance2 += 1000
			balanceAccepted2 += 1000
			_, err := invC.AuthTx(authCtx, &api.AuthTxRequest{
				TxId: trID,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})

		t.Run("AuthTx", func(t *testing.T) {
			balance2 -= 500
			_, err := invC.AuthTx(authCtx, &api.AuthTxRequest{
				TxId: trID2,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
		t.Run("AcceptTx", func(t *testing.T) {
			balance1 += 500
			balanceAccepted1 += 500
			balanceAccepted2 -= 500
			_, err := invC.AcceptTx(authCtx, &api.AcceptTxRequest{
				TxId: trID2,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
	})

	t.Run("ChangeWithHoldFromTransactions", func(t *testing.T) {

		t.Run("NewInvoice", func(t *testing.T) {
			res, err := invC.NewInvoice(authCtx, &api.NewInvoiceRequest{
				Key:      "inv1",
				Strategy: (&isimple.Strategy{}).Name().String(),
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

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			key := "1"
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tsimple.Strategy{}).Name().String(),
				Key:       &key,
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID1,
						DstAccId:  accID2,
						Strategy:  api.OperStrategy_SIMPLE_OPS,
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

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			key := "2"
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tsimple.Strategy{}).Name().String(),
				Key:       &key,
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID2,
						DstAccId:  accID1,
						Strategy:  api.OperStrategy_SIMPLE_OPS,
						Amount:    500,
						Meta:      nil,
						Hold:      true,
						HoldAccId: nil,
					},
				},
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTxId())
			trID2 = res.GetTxId()
		})

		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID2,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_DRAFT_TX, res.GetTx().GetStatus())
		})

		t.Run("AuthTx", func(t *testing.T) {
			balance1 -= 1000
			_, err := invC.AuthTx(authCtx, &api.AuthTxRequest{
				TxId: trID,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})

		t.Run("AuthTx", func(t *testing.T) {
			balance2 -= 500
			_, err := invC.AuthTx(authCtx, &api.AuthTxRequest{
				TxId: trID2,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
		t.Run("AcceptTx", func(t *testing.T) {
			balance2 += 1000
			balanceAccepted1 -= 1000
			balanceAccepted2 += 1000
			_, err := invC.AcceptTx(authCtx, &api.AcceptTxRequest{
				TxId: trID,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
		t.Run("AcceptTx", func(t *testing.T) {
			balance1 += 500
			balanceAccepted1 += 500
			balanceAccepted2 -= 500
			_, err := invC.AcceptTx(authCtx, &api.AcceptTxRequest{
				TxId: trID2,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
	})

	t.Run("RejectWithHoldFromTransactions", func(t *testing.T) {

		t.Run("NewInvoice", func(t *testing.T) {
			res, err := invC.NewInvoice(authCtx, &api.NewInvoiceRequest{
				Key:      "inv1",
				Strategy: (&isimple.Strategy{}).Name().String(),
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

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			key := "1"
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tsimple.Strategy{}).Name().String(),
				Key:       &key,
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID1,
						DstAccId:  accID2,
						Strategy:  api.OperStrategy_SIMPLE_OPS,
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

		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			key := "2"
			res, err := invC.AddTransactionToInvoice(authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId: invID,
				Strategy:  (&tsimple.Strategy{}).Name().String(),
				Key:       &key,
				Operations: []*api.AddTransactionToInvoiceRequest_Oper{
					{
						SrcAccId:  accID2,
						DstAccId:  accID1,
						Strategy:  api.OperStrategy_SIMPLE_OPS,
						Amount:    500,
						Meta:      nil,
						Hold:      true,
						HoldAccId: nil,
					},
				},
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTxId())
			trID2 = res.GetTxId()
		})

		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := invC.GetTransactionByID(authCtx, &api.GetTransactionByIDRequest{
				TxId: trID2,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTx())
			require.EqualValues(t, api.TxStatus_DRAFT_TX, res.GetTx().GetStatus())
		})

		t.Run("AuthTx", func(t *testing.T) {
			balance1 -= 1000
			_, err := invC.AuthTx(authCtx, &api.AuthTxRequest{
				TxId: trID,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})

		t.Run("AuthTx", func(t *testing.T) {
			balance2 -= 500
			_, err := invC.AuthTx(authCtx, &api.AuthTxRequest{
				TxId: trID2,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
		t.Run("RejectTx", func(t *testing.T) {
			balance1 += 1000
			_, err := invC.RejectTx(authCtx, &api.RejectTxRequest{
				TxId: trID,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
		t.Run("RejectTx", func(t *testing.T) {
			balance2 += 500
			_, err := invC.RejectTx(authCtx, &api.RejectTxRequest{
				TxId: trID2,
			})
			require.NoError(t, err)
		})

		time.Sleep(3 * time.Second)

		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := accC.GetAccountByKey(authCtx, &api.GetAccountByKeyRequest{
				CurrId: currID,
				Key:    "acc1",
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
				Key:    "acc2",
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, accID2, res.GetAccount().GetAccId())
			require.EqualValues(t, balance2, res.GetAccount().GetBalance())
			require.EqualValues(t, balanceAccepted2, res.GetAccount().GetBalanceAccepted())

		})
	})

}
