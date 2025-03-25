package golog

import (
	"fmt"
	"strings"

	"go.uber.org/zap/zapcore"
)

// Level 表示日志级别的枚举类型
type Level int

const (
	DebugLevel   Level = iota // 调试级别
	InfoLevel                 // 常规信息，用于跟踪程序运行状态
	WarnLevel                 // 警告信息，表明潜在问题但不影响运行
	ErrorLevel                // 错误信息，需要立即关注的问题
	FatalLevel                // 致命错误，程序将终止运行
	invalidLevel              // 内部使用，限制枚举范围
)

// 日志级别字符串映射（包含标准格式和常见变体）
var levelStringMap = map[string]Level{
	"DEBUG": DebugLevel,
	"INFO":  InfoLevel,
	"WARN":  WarnLevel,
	"ERROR": ErrorLevel,
	"FATAL": FatalLevel,
}

// String 实现Stringer接口
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return fmt.Sprintf("INVALID(%d)", l)
	}
}

// ZapLevel 转换为zapcore.Level
func (l Level) ZapLevel() zapcore.Level {
	switch l {
	case DebugLevel:
		return zapcore.DebugLevel
	case InfoLevel:
		return zapcore.InfoLevel
	case WarnLevel:
		return zapcore.WarnLevel
	case ErrorLevel:
		return zapcore.ErrorLevel
	case FatalLevel:
		return zapcore.FatalLevel
	default:
		return zapcore.InvalidLevel
	}
}

// Enabled 判断是否允许记录该级别日志
func (l Level) Enabled(minLevel Level) bool {
	return l >= minLevel && l < invalidLevel
}

// ParseLevel 解析日志级别（大小写不敏感）
func ParseLevel(s string) (Level, error) {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	if level, exists := levelStringMap[normalized]; exists {
		return level, nil
	}
	return InfoLevel, fmt.Errorf("invalid log level: %q", s)
}
