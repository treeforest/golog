package golog

import (
	"sync"

	"go.uber.org/zap"
)

// Logger 接口定义
type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Debugw(msg string, keysAndValues ...interface{})

	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Infow(msg string, keysAndValues ...interface{})

	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Warnw(msg string, keysAndValues ...interface{})

	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Errorw(msg string, keysAndValues ...interface{})

	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Fatalw(msg string, keysAndValues ...interface{})

	AddCallerSkip(skip int)
	SetLevel(lvl Level)
	GetLevel() Level
	Sync() error

	// Zap 返回底层强类型 zap.Logger，适用于热路径
	Zap() *zap.Logger
	// Clone 创建独立副本（内建 AddCallerSkip(1)），用于设为全局默认而不修改原实例
	Clone() Logger
}

// coreLogger 基于 zap 的 Logger 实现
type coreLogger struct {
	*zap.SugaredLogger
	zapLogger   *zap.Logger     // 底层 zap.Logger，供 Zap() 与选项变更使用
	atomicLevel zap.AtomicLevel // 原子级别，支持运行时 SetLevel
	rotWriter   *rotatingWriter // 文件轮转写入器，仅根实例持有；Sync 时关闭
}

var _ Logger = (*coreLogger)(nil)

func (l *coreLogger) SetLevel(lvl Level) {
	l.atomicLevel.SetLevel(lvl.ZapLevel())
}

func (l *coreLogger) GetLevel() Level {
	lvl, _ := ParseLevel(l.atomicLevel.Level().String())
	return lvl
}

func (l *coreLogger) AddCallerSkip(skip int) {
	l.zapLogger = l.zapLogger.WithOptions(zap.AddCallerSkip(skip))
	l.SugaredLogger = l.zapLogger.Sugar()
}

// Zap 返回底层 *zap.Logger
func (l *coreLogger) Zap() *zap.Logger {
	return l.zapLogger
}

// Clone 克隆 Logger 并增加一层 caller skip，不修改原实例
func (l *coreLogger) Clone() Logger {
	zl := l.zapLogger.WithOptions(zap.AddCallerSkip(1))
	return &coreLogger{
		SugaredLogger: zl.Sugar(),
		zapLogger:     zl,
		atomicLevel:   l.atomicLevel,
	}
}

func (l *coreLogger) Sync() error {
	if l.rotWriter != nil {
		l.rotWriter.close()
	}
	return l.zapLogger.Sync()
}

var (
	defaultMu     sync.RWMutex // 保护 defaultLogger 并发读写
	defaultLogger Logger
	defaultOnce   sync.Once // 首次调用全局方法时懒加载默认 Logger
)

// getDefault 获取全局默认 Logger（懒加载 + 读锁）
func getDefault() Logger {
	defaultOnce.Do(func() {
		defaultMu.Lock()
		defer defaultMu.Unlock()
		// SetDefaultLogger 可能已在首次调用前设置；勿用 defaultConfig 覆盖
		if defaultLogger != nil {
			return
		}
		defaultLogger = NewLogger(defaultConfig()).Clone()
	})
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultLogger
}

// SetDefaultLogger 设置全局默认 Logger（内部 Clone，不修改传入实例）
func SetDefaultLogger(logger Logger) {
	cloned := logger.Clone()
	defaultMu.Lock()
	defaultLogger = cloned
	defaultMu.Unlock()
}

func Debug(args ...interface{}) {
	getDefault().Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	getDefault().Debugf(format, args...)
}

func Debugw(msg string, keysAndValues ...interface{}) {
	getDefault().Debugw(msg, keysAndValues...)
}

func Info(args ...interface{}) {
	getDefault().Info(args...)
}

func Infof(format string, args ...interface{}) {
	getDefault().Infof(format, args...)
}

func Infow(msg string, keysAndValues ...interface{}) {
	getDefault().Infow(msg, keysAndValues...)
}

func Warn(args ...interface{}) {
	getDefault().Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	getDefault().Warnf(format, args...)
}

func Warnw(msg string, keysAndValues ...interface{}) {
	getDefault().Warnw(msg, keysAndValues...)
}

func Error(args ...interface{}) {
	getDefault().Error(args...)
}

func Errorf(format string, args ...interface{}) {
	getDefault().Errorf(format, args...)
}

func Errorw(msg string, keysAndValues ...interface{}) {
	getDefault().Errorw(msg, keysAndValues...)
}

func Fatal(args ...interface{}) {
	getDefault().Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	getDefault().Fatalf(format, args...)
}

func Fatalw(msg string, keysAndValues ...interface{}) {
	getDefault().Fatalw(msg, keysAndValues...)
}

func SetLevel(lvl Level) {
	getDefault().SetLevel(lvl)
}

func GetLevel() Level {
	return getDefault().GetLevel()
}

func AddCallerSkip(skip int) {
	getDefault().AddCallerSkip(skip)
}

func Sync() error {
	return getDefault().Sync()
}
