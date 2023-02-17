package xlog

import (
	"fmt"

	"go.uber.org/zap/zapcore"
)

func encodeLevel(l zapcore.Level) (string, termColor) {
	// 文本和颜色大部分从zap中沿用而来
	switch l {
	case zapcore.DebugLevel:
		return "DEBUG", colorMagenta
	case zapcore.InfoLevel:
		return "INFO", colorBlue
	case zapcore.WarnLevel:
		return "WARN", colorYellow
	case zapcore.ErrorLevel:
		return "ERROR", colorRed
	default:
		return fmt.Sprintf("LEVEL(%d)", l), colorRed
	}
}

// 自定义LevelEncoder
func customLevelEncoder(withColor bool) func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	return func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		lvlName, color := encodeLevel(l)
		if withColor {
			lvlName = color.Add(lvlName)
		}
		enc.AppendString(lvlName)
	}
}
