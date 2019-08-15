package services

import (
	"context"

	"github.com/gebv/acca/api"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"gopkg.in/reform.v1"
)

type ctxKey int

const (
	loggerKey ctxKey = iota
	clientCtxKey
)

const (
	AccessTokenMDKey = "access-token"
)

// InterceptSession обработчик контекста (запроса) на предмет сессии
// На основе метода (справочник) определяем требования к сессии
// В результате работы обработчика всегда в контекст кладется сессия
func (s *Service) InterceptSession(ctx context.Context, methodName string, ip string) (context.Context, error) {
	// получить device-id из метадаты gRPC запроса
	var accessToken string
	md, _ := metadata.FromIncomingContext(ctx)

	if md != nil {
		vs := md.Get(AccessTokenMDKey)
		switch len(vs) {
		case 0:
		case 1:
			accessToken = vs[0]
		default:
			GetLogger(ctx).Warn("Got several access token.", zap.Int("count", len(vs)))
			return nil, api.MakeError(codes.Unauthenticated, "No access token.")
		}
	}

	if accessToken == "" {
		return nil, api.MakeError(codes.PermissionDenied, "Access token is required.")
	}

	var client Client
	err := s.db.FindOneTo(&client, "access_token", accessToken)
	switch err {
	case nil:
	case reform.ErrNoRows:
		return nil, api.MakeError(codes.PermissionDenied, "Not found access_token.")
	default:
		GetLogger(ctx).Warn("Failed find access_token.", zap.Error(err))
		return nil, api.MakeError(codes.PermissionDenied, "Failed find access_token.")
	}

	ctx = context.WithValue(ctx, clientCtxKey, &client)

	return ctx, nil
}
