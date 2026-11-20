package golog

import (
	"fmt"
	"strings"

	"go.uber.org/zap/zapcore"
)

// Level 表示日志严重程度的枚举级别类型。
type Level int

// 日志级别常量定义，级别数值随严重性递增。
const (
	DebugLevel   Level = iota // 调试级别：用于输出细粒度的排错与诊断信息
	InfoLevel                 // 信息级别：用于常规运行时状态跟踪（推荐生产默认）
	WarnLevel                 // 警告级别：表示发生非预期情况或潜在风险，但程序仍可正常继续运行
	ErrorLevel                // 错误级别：表示发生运行错误，需关注但无需终止当前进程
	FatalLevel                // 致命级别：表示系统发生不可恢复的严重故障，记录后将立即终止进程（os.Exit(1)）
	invalidLevel              // 内部保留，用于界定合法级别的上边界
)

// levelStringMap 维护支持的日志级别字符串到 Level 枚举的映射表。
var levelStringMap = map[string]Level{
	"DEBUG": DebugLevel,
	"INFO":  InfoLevel,
	"WARN":  WarnLevel,
	"ERROR": ErrorLevel,
	"FATAL": FatalLevel,
}

// String 实现 fmt.Stringer 接口，返回该级别的标准大写字符串表示。
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

// ZapLevel 将当前 Level 转换为对应的 zapcore.Level 枚举。
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

// Enabled 检查当前日志级别是否大于或等于指定的最低输出门槛级别 minLevel。
func (l Level) Enabled(minLevel Level) bool {
	return l >= minLevel && l < invalidLevel
}

// ParseLevel 将字符串形式的日志级别（大小写不敏感，支持前后空格）解析为 Level 枚举。
// 若输入不属于合法级别，返回 InfoLevel 及对应错误。
func ParseLevel(s string) (Level, error) {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	if level, exists := levelStringMap[normalized]; exists {
		return level, nil
	}
	return InfoLevel, fmt.Errorf("invalid log level: %q", s)
}
