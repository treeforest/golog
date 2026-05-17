package golog

import (
	"fmt"

	"go.uber.org/zap/zapcore"
)

// color ANSI 终端颜色码
type color int

const (
	colorBlack color = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
	colorCyan
	colorWhite
)

// showColor 为文本添加 ANSI 颜色
func showColor(c color, msg string) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", int(c), msg)
}

// showColorBold 为文本添加 ANSI 粗体颜色
func showColorBold(c color, msg string) string {
	return fmt.Sprintf("\033[%d;1m%s\033[0m", int(c), msg)
}

// levelColor 根据日志级别返回对应颜色
func levelColor(level zapcore.Level) color {
	switch level {
	case zapcore.DebugLevel:
		return colorCyan
	case zapcore.InfoLevel:
		return colorGreen
	case zapcore.WarnLevel:
		return colorYellow
	case zapcore.ErrorLevel:
		return colorRed
	case zapcore.FatalLevel, zapcore.PanicLevel, zapcore.DPanicLevel:
		return colorRed
	default:
		return colorWhite
	}
}

// colorLevelEncoder 为控制台输出级别字段着色（INFO 绿 / WARN 黄 / ERROR 红等）
func colorLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	text := level.CapitalString()
	if level >= zapcore.FatalLevel {
		enc.AppendString(showColorBold(levelColor(level), text))
		return
	}
	enc.AppendString(showColor(levelColor(level), text))
}
