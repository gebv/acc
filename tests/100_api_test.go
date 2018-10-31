package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gebv/acca/api/acca"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	Ctx  context.Context
	Conn *grpc.ClientConn
)

func Test100_01SetupApi(t *testing.T) {
	Ctx, _ = context.WithCancel(context.Background())

	var err error
	Conn, err = grpc.Dial(accaGrpcAddr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		t.Fatal(err)
	}
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
		res, err := c.CreateAccount(ctx, &acca.CreateAccountRequest{CurrencyId: currID, Key: "ns.user1.main", Meta: map[string]string{"foo": "bar"}}, grpc.Trailer(&md))
		require.NoError(t, err)
		require.NotEmpty(t, res.AccId)
		accID = res.AccId
	})

	t.Run("GetAccount", func(t *testing.T) {
		res, err := c.GetAccountsByIDs(ctx, &acca.GetAccountsByIDsRequest{AccIds: []int64{accID}}, grpc.Trailer(&md))
		require.NoError(t, err)
		if assert.NotEmpty(t, res) {
			if assert.Len(t, res.Accounts, 1) {
				got := res.Accounts[0]
				assert.Equal(t, accID, got.AccId)
				assert.Equal(t, currID, got.CurrId)
				assert.Equal(t, "ns.user1.main", got.Key)
				assert.Equal(t, map[string]string{"foo": "bar"}, got.Meta)
			}
		}
	})
}
