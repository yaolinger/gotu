package xlog

import (
	"os"

	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	withStdout int = iota
	withoutStdout
)

var (
	needStdout int = withStdout
	gLogger    Logger
)

func init() {
	defaultLogLvl := zapcore.DebugLevel
	gLogger = initLogger(defaultLogLvl)

}

func getEncoder(isProd bool) zapcore.Encoder {
	// 使用ECS兼容的encoder格式
	withColor := !isProd
	config := ecsCompatibleEncoder(withColor)
	config.TimeKey = FieldTimestamp
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	if isProd {
		// prod: 默认使用json格式
		return zapcore.NewJSONEncoder(config)
	} else {
		// dev: 使用带颜色的终端格式
		return zapcore.NewConsoleEncoder(config)
	}
}

// Elastic Common Schema (ECS) 兼容的encoder格式, 便于日志被ELK归档
func ecsCompatibleEncoder(withColor bool) zapcore.EncoderConfig {
	return ecszap.EncoderConfig{
		// 这里只开放了和ECS无关的字段
		EnableName:       true,
		EncodeName:       zapcore.FullNameEncoder,
		EnableStackTrace: true,
		EnableCaller:     true,
		EncodeCaller:     zapcore.ShortCallerEncoder,
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeLevel:      customLevelEncoder(withColor),
		EncodeDuration:   zapcore.StringDurationEncoder,
	}.ToZapCoreEncoderConfig()
}

func defaultOptions() []zap.Option {
	options := []zap.Option{
		// zap.Development(),
		zap.WithCaller(true),
		// DPanic时自动增加Stacktrace
		zap.AddStacktrace(zap.NewAtomicLevelAt(zap.DPanicLevel)),
	}
	return options
}

func initLogger(logLvl zapcore.Level) Logger {
	writerSinker := zapcore.Lock(os.Stdout)
	if needStdout == withoutStdout {
		// 关闭日志标准输出
		writerSinker = zapcore.Lock(zapcore.NewMultiWriteSyncer())
	}
	core := zapcore.NewCore(getEncoder(false), writerSinker, zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= logLvl
	}))
	return newLogger(zap.New(core, defaultOptions()...))
}
