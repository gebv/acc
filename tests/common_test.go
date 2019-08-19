package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	Conn        *grpc.ClientConn
	Ctx         context.Context
	AccessToken string
)

const (
	accessTokenMDKey = "access-token"
)

func assertGRPCError(t testing.TB, expected *status.Status, actual error) {
	t.Helper()

	s, ok := status.FromError(actual)
	if !assert.True(t, ok, "expected gRPC Status, got %T:\n%s", actual, actual) {
		return
	}
	assert.Equal(t, expected.Code(), s.Code(), "gRPC status codes are not equal")
	assert.Equal(t, expected.Message(), s.Message(), "gRPC status messages are not equal")
}
