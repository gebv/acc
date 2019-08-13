package auth

import (
	"context"
	"strings"

	"github.com/gebv/acca/api"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware"

	"github.com/gebv/acca/httputils"
	"github.com/gebv/acca/services/auditor"
	validator "github.com/mwitkow/go-proto-validators"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type Interceptor interface {
	InterceptSession(ctx context.Context, methodName string, ip string) (context.Context, error)
}

// Unary интерцептор с логикой авторизации сессии.
func Unary(i Interceptor, auditor *auditor.HttpAuditor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		// значение задает интерцептор settings
		reqMeta := httputils.GetRequestInfo(ctx)

		var userID *int64
		var err error
		ctx, err = i.InterceptSession(ctx, info.FullMethod, reqMeta.RealIP)
		if err != nil {
			auditor.Log(
				ctx,
				userID,
				reqMeta.DeviceID,
				reqMeta.SessionID,
				reqMeta.UserAgent,
				reqMeta.RealIP,
				reqMeta.FirstProxyIP(),
				reqMeta.RequestID,
				info.FullMethod,
				req,
			)
			auditor.Log(
				ctx,
				userID,
				reqMeta.DeviceID,
				reqMeta.SessionID,
				reqMeta.UserAgent,
				reqMeta.RealIP,
				reqMeta.FirstProxyIP(),
				reqMeta.RequestID,
				info.FullMethod+"/invalid_session",
				err,
			)
			return nil, err
		}

		auditor.Log(
			ctx,
			userID,
			reqMeta.DeviceID,
			reqMeta.SessionID,
			reqMeta.UserAgent,
			reqMeta.RealIP,
			reqMeta.FirstProxyIP(),
			reqMeta.RequestID,
			info.FullMethod,
			req,
		)

		// валидация если задана
		if v, ok := req.(validator.Validator); ok {
			if err := v.Validate(); err != nil {

				auditor.Log(
					ctx,
					userID,
					reqMeta.DeviceID,
					reqMeta.SessionID,
					reqMeta.UserAgent,
					reqMeta.RealIP,
					reqMeta.FirstProxyIP(),
					reqMeta.RequestID,
					info.FullMethod+"/invalid_argument",
					err,
				)

				return nil, api.MakeError(codes.InvalidArgument, "Validation failed.")
			}
		}

		resp, err := handler(ctx, req)

		// NOTE: в случае ошибки в респонсе как правило (В таблице users_http_request) в поле body будет null.
		// В этом случае надо по request_id найти в логах ошибку типа WARN
		// Например:
		//
		// WARN	/api.Receipts/AddReceipt	recover/recover.go:33	Done with gRPC error.	{"request_id": "test-1544204264087962000", "duration": "3.177234ms", "error": "rpc error: code = Internal desc = Error message"}
		auditor.Log(
			ctx,
			userID,
			reqMeta.DeviceID,
			reqMeta.SessionID,
			reqMeta.UserAgent,
			reqMeta.RealIP,
			reqMeta.FirstProxyIP(),
			reqMeta.RequestID,
			info.FullMethod+"/response",
			resp,
		)

		return resp, err
	}
}

func Stream(i Interceptor) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// do not authenticate other APIs (for example, gRPC reflection APIs)
		if !strings.HasPrefix(info.FullMethod, "/api") {
			return handler(srv, ss)
		}

		// значение зада интерцептор settings
		reqMeta := httputils.GetRequestInfo(ss.Context())

		ctx, err := i.InterceptSession(ss.Context(), info.FullMethod, reqMeta.RealIP)
		if err != nil {
			return err
		}

		wrapped := middleware.WrapServerStream(ss)
		wrapped.WrappedContext = ctx
		return handler(srv, wrapped)
	}
}
