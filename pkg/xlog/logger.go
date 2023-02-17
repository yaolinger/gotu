package xlog

import "go.uber.org/zap"

type Logger interface {
	// TODO: 后续不再提供Sugar()
	Sugar() *zap.SugaredLogger

	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field) // 严重错误 一定次数报警

	// TODO FATAL EMERGENT DPANIC

	// 返回原始zap.Logger
	Raw() *zap.Logger
}

type xlogger struct {
	*zap.Logger
}

func newLogger(l *zap.Logger) Logger {
	return &xlogger{Logger: l}
}

func (log *xlogger) Raw() *zap.Logger { return log.Logger }
