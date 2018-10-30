package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

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

func Test100_02CreateAccount(t *testing.T) {
	c := acca.NewAccountsClient(Conn)

	ctx := metadata.NewOutgoingContext(Ctx, metadata.Pairs("foo", "bar"))
	var md metadata.MD

	res, err := c.CreateCurrency(ctx, &acca.CreateCurrencyRequest{Key: "from_i.curr"}, grpc.Trailer(&md))
	assert.NoError(t, err)
	assert.NotEmpty(t, res.CurrencyId)
}
