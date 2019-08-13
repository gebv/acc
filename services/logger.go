package services

import (
	"context"

	"go.uber.org/zap"
)

// GetLogger returns logger from the context.
func GetLogger(ctx context.Context) *zap.Logger {
	return ctx.Value(loggerKey).(*zap.Logger)
}

// SetLogger returns a new context with set (or re-set) logger.
func SetLogger(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}
