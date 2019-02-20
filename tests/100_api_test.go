package tests

import (
	"context"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gebv/acca/api/acca"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	Ctx  context.Context
	Conn *grpc.ClientConn

	// list accounts
	// key - accountID
	// value - balance
	accounts = map[int64]int64{}
)

// Required case for tests in this file
//
// init gRPC client
// TODO: move to separate package
func Test100_01SetupApi(t *testing.T) {

	t.Run("ConnectToBackend", func(t *testing.T) {
		var err error
		t.Logf("Listen address: %v", accaGrpcAddr)
		Conn, err = grpc.Dial(accaGrpcAddr, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			t.Fatal(err)
		}
	})

}

func Test100_02CreateAndGetCurrency(t *testing.T) {
	c := acca.NewAccountsClient(Conn)

	ctx := metadata.NewOutgoingContext(Ctx, metadata.Pairs("foo", "bar"))
	var md metadata.MD

	t.Run("CreateCurrency", func(t *testing.T) {
		res, err := c.CreateCurrency(ctx, &acca.CreateCurrencyRequest{Key: "from_i.curr", Meta: map[string]string{"foo": "bar"}}, grpc.Trailer(&md))
		require.NoError(t, err)
		assert.NotEmpty(t, res.CurrencyId)
	})

	t.Run("GetCreatedCurrency", func(t *testing.T) {
		res, err := c.GetCurrencies(ctx, &acca.GetCurrenciesRequest{Key: "from_i.curr"}, grpc.Trailer(&md))
		require.NoError(t, err)
		if assert.NotEmpty(t, res) {
			if assert.Len(t, res.Currencies, 1) {
				got := res.Currencies[0]
				assert.Equal(t, "from_i.curr", got.Key)
				assert.Equal(t, map[string]string{"foo": "bar"}, got.Meta)
			}
		}
	})
}

func Test100_02CreateAndGetAccount(t *testing.T) {
	c := acca.NewAccountsClient(Conn)

	ctx := metadata.NewOutgoingContext(Ctx, metadata.Pairs("foo", "bar"))
	var md metadata.MD

	res, err := c.GetCurrencies(ctx, &acca.GetCurrenciesRequest{Key: "from_i.curr"}, grpc.Trailer(&md))
	require.NoError(t, err)
	currID := res.Currencies[0].CurrId
	require.NotEmpty(t, currID)

	var accID int64

	t.Run("CreateAccount", func(t *testing.T) {
		res, err := c.CreateAccount(ctx, &acca.CreateAccountRequest{CurrencyId: currID, Key: "ma.user1.main", Meta: map[string]string{"foo": "bar"}}, grpc.Trailer(&md))
		require.NoError(t, err)
		require.NotEmpty(t, res.AccId)
		accID = res.AccId
	})

	t.Run("GetAccountByID", func(t *testing.T) {
		res, err := c.GetAccountsByIDs(ctx, &acca.GetAccountsByIDsRequest{AccIds: []int64{accID}}, grpc.Trailer(&md))
		require.NoError(t, err)
		if assert.NotEmpty(t, res) {
			if assert.Len(t, res.Accounts, 1) {
				got := res.Accounts[0]
				assert.Equal(t, accID, got.AccId)
				assert.Equal(t, currID, got.CurrId)
				assert.Equal(t, "ma.user1.main", got.Key)
				assert.Equal(t, map[string]string{"foo": "bar"}, got.Meta)
				assert.EqualValues(t, 0, got.GetBalanceAccepted())
			} else {
				t.Fatal("Expected accounts")
			}
		} else {
			t.Fatal("Expected accounts")
		}
	})

	t.Run("GetAccountByKey", func(t *testing.T) {
		res, err := c.GetAccountsByKey(ctx, &acca.GetAccountsByKeyRequest{Key: "ma.user1.main"}, grpc.Trailer(&md))
		require.NoError(t, err)
		if assert.NotEmpty(t, res) {
			if assert.Len(t, res.Accounts, 1) {
				got := res.Accounts[0]
				assert.Equal(t, accID, got.AccId)
				assert.Equal(t, currID, got.CurrId)
				assert.Equal(t, "ma.user1.main", got.Key)
				assert.Equal(t, map[string]string{"foo": "bar"}, got.Meta)
				assert.EqualValues(t, 0, got.GetBalanceAccepted())
			} else {
				t.Fatal("Expected accounts")
			}
		} else {
			t.Fatal("Expected accounts")
		}
	})

	t.Run("GetAccountByUser", func(t *testing.T) {
		res, err := c.GetAccountsByUserID(ctx, &acca.GetAccountsByUserIDRequest{UserId: "user1"}, grpc.Trailer(&md))
		require.NoError(t, err)

		if assert.NotEmpty(t, res) && assert.NotEmpty(t, res.UserAccounts) {
			if assert.Len(t, res.UserAccounts.Balances, 1) {
				got := res.UserAccounts.Balances[0]
				assert.Equal(t, accID, got.AccId)
				assert.Equal(t, "main", got.Type)
				assert.EqualValues(t, 0, got.Balance)
				assert.EqualValues(t, 0, got.BalanceAccepted)
			} else {
				t.Fatal("Expected list balances")
			}
		} else {
			t.Fatal("Expected user accounts")
		}
	})
}

func Test100_03CreateTransfer(t *testing.T) {
	var _, acc1ID, acc2ID int64
	t.Run("Init", func(t *testing.T) {
		_, acc1ID = createAccount(t, "from_i.curr", "ma.user1.main")
		_, acc2ID = createAccount(t, "from_i.curr", "ma.user2.main")
		loadAccountBalances(t, "ma")
	})

	require.NotEmpty(t, acc1ID)
	require.NotEmpty(t, acc2ID)
	mux := sync.RWMutex{}
	recivedUpdates := []*acca.Update{}

	t.Run("GetUpdates", func(t *testing.T) {
		c := acca.NewTransferClient(Conn)
		ctx := metadata.NewOutgoingContext(Ctx, metadata.Pairs("foo", "bar"))

		res, err := c.GetUpdates(ctx, &acca.GetUpdatesRequest{})
		require.NoError(t, err)
		go func() {
			for {
				msg, err := res.Recv()
				if err != nil {
					log.Println(err)
					return
				}
				mux.Lock()
				recivedUpdates = append(recivedUpdates, msg)
				mux.Unlock()
			}
		}()

	})

	t.Run("Transfer", func(t *testing.T) {
		c := acca.NewTransferClient(Conn)
		ctx := metadata.NewOutgoingContext(Ctx, metadata.Pairs("foo", "bar"))

		res, err := c.NewTransfer(ctx, &acca.NewTransferRequest{
			Reason: "testing",
			Meta:   map[string]string{"foo": "bar"},
			Opers: []*acca.TxOper{
				{
					SrcAccId: acc1ID,
					DstAccId: acc2ID,
					Type:     Recharge,
					Amount:   20,
					Reason:   "reacharge",
					Meta:     map[string]string{"foo": "bar"},
				},
				{
					SrcAccId: acc2ID,
					DstAccId: acc1ID,
					Type:     Internal,
					Amount:   20,
					Reason:   "internal",
					Meta:     map[string]string{"foo": "bar"},
				},
			},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, res.TxId)
	})

	t.Run("Apply", func(t *testing.T) {
		c := acca.NewTransferClient(Conn)
		ctx := metadata.NewOutgoingContext(Ctx, metadata.Pairs("foo", "bar"))

		res, err := c.HandleRequests(ctx, &acca.HandleRequestsRequest{Limit: 1})
		require.NoError(t, err)

		require.EqualValues(t, 1, res.NumOk)
		require.EqualValues(t, 0, res.NumErr)
	})

	t.Run("CheckBalances", func(t *testing.T) {
		c := acca.NewAccountsClient(Conn)
		ctx := metadata.NewOutgoingContext(Ctx, metadata.Pairs("foo", "bar"))

		res, err := c.GetAccountsByIDs(ctx, &acca.GetAccountsByIDsRequest{AccIds: []int64{acc1ID, acc2ID}})
		require.NoError(t, err)
		require.Len(t, res.Accounts, 2)
		loadAccountBalances(t, "ma")
		require.EqualValues(t, 40, accounts[acc1ID])
		require.EqualValues(t, 0, accounts[acc2ID])
		require.EqualValues(t, 40, res.GetAccounts()[0].BalanceAccepted)
		require.EqualValues(t, 0, res.GetAccounts()[1].BalanceAccepted)
	})

	time.Sleep(time.Millisecond * 10)

	t.Run("CheckSubscribeEvents", func(t *testing.T) {
		mux.RLock()
		require.Len(t, recivedUpdates, 4)
		mux.RUnlock()

		// last status tx accepted
		// first status tx draft

		require.EqualValues(t, "draft", recivedUpdates[0].Type.(*acca.Update_TxStatus).TxStatus.NewStatus)
		require.EqualValues(t, "accepted", recivedUpdates[len(recivedUpdates)-1].Type.(*acca.Update_TxStatus).TxStatus.NewStatus)
	})

	t.Run("LoadTxByEvents", func(t *testing.T) {
		mux.RLock()
		txID := recivedUpdates[0].Type.(*acca.Update_TxStatus).TxStatus.TxId
		mux.RUnlock()

		c := acca.NewTransferClient(Conn)
		ctx := metadata.NewOutgoingContext(Ctx, metadata.Pairs("foo", "bar"))

		res, err := c.GetTxByID(ctx, &acca.GetTxByIDRequest{TxId: txID, WithOpers: true})
		require.NoError(t, err)
		require.Equal(t, txID, res.Tx.TxId)
		require.Len(t, res.Opers, 2)
	})

}

func loadAccountBalances(t *testing.T, key string) {
	c := acca.NewAccountsClient(Conn)
	ctx := metadata.NewOutgoingContext(Ctx, metadata.Pairs("foo", "bar"))

	res, err := c.GetAccountsByKey(ctx, &acca.GetAccountsByKeyRequest{Key: key})
	require.NoError(t, err)
	require.NotEmpty(t, res.Accounts)
	for _, acc := range res.Accounts {
		accounts[acc.AccId] = acc.Balance
	}
}

func createAccount(t *testing.T, curr, key string) (currID, accID int64) {
	c := acca.NewAccountsClient(Conn)
	ctx := metadata.NewOutgoingContext(Ctx, metadata.Pairs("foo", "bar"))

	{
		c.CreateCurrency(ctx, &acca.CreateCurrencyRequest{Key: key, Meta: map[string]string{"foo": "bar"}})
	}

	{
		res, err := c.GetCurrencies(ctx, &acca.GetCurrenciesRequest{Key: key})
		require.NoError(t, err)
		require.NotEmpty(t, res.Currencies)
		currID = res.Currencies[0].CurrId
		require.NotEmpty(t, currID)
	}

	{
		res, err := c.CreateAccount(ctx, &acca.CreateAccountRequest{CurrencyId: currID, Key: key, Meta: map[string]string{"foo": "bar"}})
		require.NoError(t, err)
		require.NotEmpty(t, res.AccId)
		accID = res.AccId
	}

	return
}

func Test100_04RecentActivity(t *testing.T) {
	c := acca.NewTransferClient(Conn)
	ctx := metadata.NewOutgoingContext(Ctx, metadata.Pairs("foo", "bar"))
	ids := make([]int64, 0, 4)
	res, err := c.RecentActivity(ctx, &acca.RecentActivityRequest{LastId: 0, Limit: 2})
	require.NoError(t, err)
	require.Len(t, res.GetList(), 2)
	for _, v := range res.GetList() {
		t.Log("RecentActivity: ", v)
		ids = append(ids, v.Id)
	}

	lastID := res.List[len(res.List)-1].Id

	res, err = c.RecentActivity(ctx, &acca.RecentActivityRequest{LastId: lastID, Limit: 2})
	require.NoError(t, err)
	require.Len(t, res.GetList(), 2)
	for _, v := range res.GetList() {
		t.Log("RecentActivity: ", v)
		ids = append(ids, v.Id)
	}

	var lID int64
	for i, id := range ids {
		if i == 0 {
			lID = id
			continue
		}
		require.True(t, lID > id)
		lID = id
	}

	// TODO: more tests
}

func Test100_05JournalActivity(t *testing.T) {
	c := acca.NewTransferClient(Conn)
	ctx := metadata.NewOutgoingContext(Ctx, metadata.Pairs("foo", "bar"))
	ids := make([]int64, 0, 4)
	res, err := c.JournalActivity(ctx, &acca.JournalActivityRequest{LastId: 0, Limit: 2})
	require.NoError(t, err)
	require.Len(t, res.GetList(), 2)
	for _, v := range res.GetList() {
		t.Log("JournalActivity: ", v)
		ids = append(ids, v.Id)
	}

	lastID := res.GetList()[len(res.GetList())-1].Id

	res, err = c.JournalActivity(ctx, &acca.JournalActivityRequest{LastId: lastID, Limit: 2})
	require.NoError(t, err)
	require.Len(t, res.GetList(), 2)
	for _, v := range res.GetList() {
		t.Log("JournalActivity: ", v)
		ids = append(ids, v.Id)
	}

	var lID int64
	for i, id := range ids {
		if i == 0 {
			lID = id
			continue
		}
		require.True(t, lID < id)
		lID = id
	}

	// TODO: more tests
}
