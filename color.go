package golog

import (
	"fmt"

	"go.uber.org/zap/zapcore"
)

// color ANSI 终端前景色编码定义。
type color int

const (
	colorBlack   color = iota + 30 // 黑色
	colorRed                       // 红色
	colorGreen                     // 绿色
	colorYellow                    // 黄色
	colorBlue                      // 蓝色
	colorMagenta                   // 洋红/紫色
	colorCyan                      // 青色/蓝绿
	colorWhite                     // 白色
)

// showColor 将文本包装为标准 ANSI 颜色转义字符。
func showColor(c color, msg string) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", int(c), msg)
}

// showColorBold 将文本包装为带粗体（Bold）属性的 ANSI 颜色转义字符。
func showColorBold(c color, msg string) string {
	return fmt.Sprintf("\033[%d;1m%s\033[0m", int(c), msg)
}

// levelColor 根据日志级别返回对应的 ANSI 终端颜色。
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

// colorLevelEncoder 为终端控制台输出提供级别字段的高亮着色编码（INFO 绿色 / WARN 黄色 / ERROR 红色 / FATAL 红色粗体）。
// 注意：该编码器仅作用于终端控制台输出，不会应用于文件输出流，以保持文件纯文本纯净。
func colorLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	text := level.CapitalString()
	if level >= zapcore.FatalLevel {
		enc.AppendString(showColorBold(levelColor(level), text))
		return
	}
	enc.AppendString(showColor(levelColor(level), text))
}
