package logger

import (
	"context"

	"go.uber.org/zap"
)

type ctxKeyLogger struct{}

// ToContext 将 Logger 放入 context，供 render 等包在请求链路中取用。
func ToContext(ctx context.Context, l *zap.Logger) context.Context {
	if l == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKeyLogger{}, l)
}

// FromContext 取出 Logger；不存在时返回 Nop。
func FromContext(ctx context.Context) *zap.Logger {
	if v := ctx.Value(ctxKeyLogger{}); v != nil {
		if log, ok := v.(*zap.Logger); ok {
			return log
		}
	}
	return zap.NewNop()
}
