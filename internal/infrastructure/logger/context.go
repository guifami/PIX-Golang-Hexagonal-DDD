package logger

import (
	"context"

	"go.uber.org/zap"
)

type contextKey string

const loggerKey contextKey = "logger"

func WithContext(ctx context.Context, log *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, log)
}

func FromContext(ctx context.Context) *zap.Logger {
	if log, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		return log
	}
	return Log
}
