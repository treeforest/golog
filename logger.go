package golog

import (
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
}

type coreLogger struct {
	*zap.SugaredLogger
	atomicLevel zap.AtomicLevel
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
	l.SugaredLogger = l.WithOptions(zap.AddCallerSkip(skip))
}

// 全局默认日志器
var defaultLogger Logger

func init() {
	SetDefaultLogger(NewLogger(defaultConfig()))
}

func SetDefaultLogger(logger Logger) {
	// 调整调用栈深度加 1
	logger.AddCallerSkip(1)
	defaultLogger = logger
}

func Debug(args ...interface{}) {
	defaultLogger.Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

func Debugw(msg string, keysAndValues ...interface{}) {
	defaultLogger.Debugw(msg, keysAndValues...)
}

func Info(args ...interface{}) {
	defaultLogger.Info(args...)
}

func Infof(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

func Infow(msg string, keysAndValues ...interface{}) {
	defaultLogger.Infow(msg, keysAndValues...)
}

func Warn(args ...interface{}) {
	defaultLogger.Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	defaultLogger.Warnf(format, args...)
}

func Warnw(msg string, keysAndValues ...interface{}) {
	defaultLogger.Warnw(msg, keysAndValues...)
}

func Error(args ...interface{}) {
	defaultLogger.Error(args...)
}

func Errorf(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

func Errorw(msg string, keysAndValues ...interface{}) {
	defaultLogger.Errorw(msg, keysAndValues...)
}

func Fatal(args ...interface{}) {
	defaultLogger.Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	defaultLogger.Fatalf(format, args...)
}

func Fatalw(msg string, keysAndValues ...interface{}) {
	defaultLogger.Fatalw(msg, keysAndValues...)
}

func SetLevel(lvl Level) {
	defaultLogger.SetLevel(lvl)
}

func GetLevel() Level {
	return defaultLogger.GetLevel()
}

func AddCallerSkip(skip int) {
	defaultLogger.AddCallerSkip(skip)
}

func Sync() error {
	return defaultLogger.Sync()
}
