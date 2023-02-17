package xlog

import (
	"context"

	"go.uber.org/zap/zapcore"
)

type loggerKeyType int

const (
	loggerKey       loggerKeyType = iota
	customLoggerKey               // 自定义子logger
)

// 生成一个新的子logger，绑定到新的context中
func NewContext(ctx context.Context, fields ...zapcore.Field) context.Context {
	return context.WithValue(ctx, loggerKey, newLogger(Get(ctx).Raw().With(fields...)))
}

// 从srcContext中取出logger, 绑定到destCtx中
func FromContext(srcCtx, destCtx context.Context, fields ...zapcore.Field) context.Context {
	srcLogger := Get(srcCtx).Raw()
	return context.WithValue(destCtx, loggerKey, newLogger(srcLogger.With(fields...)))
}

// context获取logger
func Get(ctx context.Context) Logger {
	if ctx == nil {
		return gLogger
	}
	if ctxLogger, ok := ctx.Value(loggerKey).(Logger); ok {
		return ctxLogger
	}
	return gLogger
}
