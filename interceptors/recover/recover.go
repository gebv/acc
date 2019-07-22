package recover

import (
	"context"
	"time"

	"github.com/gebv/acca/services"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (res interface{}, err error) {
		start := time.Now()
		defer func() {
			dur := zap.Duration("duration", time.Since(start))
			if rec := recover(); rec != nil {
				res = nil
				err = status.New(codes.Internal, "Internal server error.").Err()
				services.GetLogger(ctx).DPanic("Unhandled panic.", zap.Any("panic", rec), dur)
				return
			}

			if err == nil {
				services.GetLogger(ctx).Info("Done.", dur)
				return
			}

			if _, ok := status.FromError(err); ok {
				services.GetLogger(ctx).Warn("Done with gRPC error.", dur, zap.Error(err))
			} else {
				services.GetLogger(ctx).Error("Done with unknown error.", dur, zap.Error(err))
			}
		}()

		res, err = handler(ctx, req)
		return
	}
}

func Stream() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		ctx := ss.Context()
		start := time.Now()
		defer func() {
			dur := zap.Duration("duration", time.Since(start))

			if rec := recover(); rec != nil {
				services.GetLogger(ctx).DPanic("Unhandled panic.", zap.Any("panic", rec))
				err = status.New(codes.Internal, "Internal server error.").Err()
				return
			}

			if err == nil {
				services.GetLogger(ctx).Info("Done.", dur)
				return
			}

			if _, ok := status.FromError(err); ok {
				services.GetLogger(ctx).Warn("Done with gRPC error.", dur, zap.Error(err))
			} else {
				services.GetLogger(ctx).Error("Done with unknown error.", dur, zap.Error(err))
			}
		}()

		err = handler(srv, ss)
		return
	}
}
