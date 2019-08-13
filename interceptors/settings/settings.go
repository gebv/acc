package settings

import (
	"context"
	"runtime/pprof"

	"github.com/gebv/acca/httputils"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/gebv/acca/services"
)

// Unary интерцептор с настройками
//
// Устанавливает в контекст
// - настройки текущие
// - request info (from package httputils.RequestInfo)
// - экзмемляр логгера, в нем задан request_id
func Unary(appVersion string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		ctx, reqMeta := httputils.SetRequestInfo(ctx, appVersion)

		l := zap.L().Named(info.FullMethod).With(
			zap.String("request_id", reqMeta.RequestID),
			zap.String("device_id", reqMeta.DeviceID),
			zap.String("session_id", reqMeta.SessionID),
			zap.String("backend_version", reqMeta.AppVersion),
		)

		if e := grpc.SetTrailer(
			ctx,
			metadata.Pairs(
				"request-id", reqMeta.RequestID,
				"device-id", reqMeta.DeviceID,
				"session-id", reqMeta.SessionID,
				"backend-version", reqMeta.AppVersion),
		); e != nil {
			l.Warn("Failed to send request-id trailer.", zap.Error(e))
		}

		// if s.Dev {
		// 	l = l.WithOptions(settings.LoggerWithLevel(zapcore.DebugLevel))
		// }
		ctx = services.SetLogger(ctx, l)

		// add pprof labels for more useful profiles
		defer pprof.SetGoroutineLabels(ctx)
		ctx = pprof.WithLabels(ctx, pprof.Labels("method", info.FullMethod))
		pprof.SetGoroutineLabels(ctx)

		return handler(ctx, req)
	}
}

func Stream(appVersion string) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

		ctx := ss.Context()

		ctx, reqMeta := httputils.SetRequestInfo(ctx, appVersion)

		l := zap.L().Named(info.FullMethod).With(
			zap.String("request_id", reqMeta.RequestID),
			zap.String("device_id", reqMeta.DeviceID),
			zap.String("session_id", reqMeta.SessionID),
			zap.String("backend_version", reqMeta.AppVersion),
		)

		// // if s.Dev {
		// // 	l = l.WithOptions(settings.LoggerWithLevel(zapcore.DebugLevel))
		// // }
		ctx = services.SetLogger(ctx, l)

		// add pprof labels for more useful profiles
		defer pprof.SetGoroutineLabels(ctx)
		ctx = pprof.WithLabels(ctx, pprof.Labels("method", info.FullMethod))
		pprof.SetGoroutineLabels(ctx)

		wrapped := middleware.WrapServerStream(ss)
		wrapped.WrappedContext = ctx
		return handler(srv, wrapped)
	}
}
