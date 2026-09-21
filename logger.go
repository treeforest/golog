package golog

import (
	"fmt"
	"os"
	"sync"

	"go.uber.org/zap"
)

// Logger 接口定义
type Logger interface {
	Debug(args ...any)
	Debugf(format string, args ...any)
	Debugw(msg string, keysAndValues ...any)

	Info(args ...any)
	Infof(format string, args ...any)
	Infow(msg string, keysAndValues ...any)

	Warn(args ...any)
	Warnf(format string, args ...any)
	Warnw(msg string, keysAndValues ...any)

	Error(args ...any)
	Errorf(format string, args ...any)
	Errorw(msg string, keysAndValues ...any)

	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Fatalw(msg string, keysAndValues ...any)

	AddCallerSkip(skip int)
	SetLevel(lvl Level)
	GetLevel() Level
	// Sync 刷盘，不关闭文件写入器
	Sync() error
	// Close 刷盘并释放文件写入器所有权（进程退出时调用）
	Close() error

	// Zap 返回底层强类型 zap.Logger，适用于热路径
	Zap() *zap.Logger
	// Clone 创建独立副本（内建 AddCallerSkip(1)），共享 rotWriter 但不取得所有权
	Clone() Logger
}

// coreLogger 基于 zap 的 Logger 实现
type coreLogger struct {
	*zap.SugaredLogger
	zapLogger   *zap.Logger     // 底层 zap.Logger，供 Zap() 与选项变更使用
	atomicLevel zap.AtomicLevel // 原子级别，支持运行时 SetLevel
	rotWriter   *rotatingWriter // 文件轮转写入器（可与 Clone/Context 派生共享）
	ownsWriter  bool            // 是否持有 rotWriter 引用（Close 时 release）
	mu          sync.Mutex      // 保护 AddCallerSkip 时的指针替换
	closeOnce   sync.Once
}

var _ Logger = (*coreLogger)(nil)

func (l *coreLogger) SetLevel(lvl Level) {
	l.atomicLevel.SetLevel(lvl.ZapLevel())
}

func (l *coreLogger) GetLevel() Level {
	lvl, err := ParseLevel(l.atomicLevel.Level().String())
	if err != nil {
		return InfoLevel
	}
	return lvl
}

func (l *coreLogger) AddCallerSkip(skip int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.zapLogger = l.zapLogger.WithOptions(zap.AddCallerSkip(skip))
	l.SugaredLogger = l.zapLogger.Sugar()
}

// Zap 返回底层 *zap.Logger
func (l *coreLogger) Zap() *zap.Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.zapLogger
}

// Clone 克隆 Logger 并增加一层 caller skip，不修改原实例；共享 rotWriter，不转移所有权
func (l *coreLogger) Clone() Logger {
	l.mu.Lock()
	zl := l.zapLogger.WithOptions(zap.AddCallerSkip(1))
	rw := l.rotWriter
	al := l.atomicLevel
	l.mu.Unlock()
	return &coreLogger{
		SugaredLogger: zl.Sugar(),
		zapLogger:     zl,
		atomicLevel:   al,
		rotWriter:     rw,
		ownsWriter:    false,
	}
}

// Sync 刷盘，不关闭写入器
func (l *coreLogger) Sync() error {
	l.mu.Lock()
	zl := l.zapLogger
	l.mu.Unlock()
	return zl.Sync()
}

// Close 刷盘并释放对本 logger 持有的 rotWriter 引用
func (l *coreLogger) Close() error {
	var err error
	l.closeOnce.Do(func() {
		err = l.Sync()
		if l.ownsWriter && l.rotWriter != nil {
			l.rotWriter.release()
			l.ownsWriter = false
		}
	})
	return err
}

var (
	defaultMu     sync.RWMutex // 保护 defaultLogger 并发读写
	defaultLogger Logger
	defaultOnce   sync.Once // 首次调用全局方法时懒加载默认 Logger
)

// transferOwnership 将 rotWriter 所有权从 src 转到 dst（均为 *coreLogger）
func transferOwnership(src, dst *coreLogger) {
	if src == nil || dst == nil {
		return
	}
	if !src.ownsWriter || src.rotWriter == nil {
		return
	}
	dst.rotWriter = src.rotWriter
	dst.ownsWriter = true
	src.ownsWriter = false
}

// getDefault 获取全局默认 Logger（懒加载 + 读锁）
func getDefault() Logger {
	defaultOnce.Do(func() {
		defaultMu.Lock()
		defer defaultMu.Unlock()
		if defaultLogger != nil {
			return
		}
		root := MustNewLogger(defaultConfig())
		rootCore, ok := root.(*coreLogger)
		if !ok {
			panic("golog: internal error: root logger is not *coreLogger")
		}
		clonedCore, ok := root.Clone().(*coreLogger)
		if !ok {
			panic("golog: internal error: cloned logger is not *coreLogger")
		}
		transferOwnership(rootCore, clonedCore)
		defaultLogger = clonedCore
	})
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultLogger
}

// SetDefaultLogger 设置全局默认 Logger（内部 Clone，不修改传入实例的 caller skip）。
// 若传入实例持有文件 writer 所有权，将转移给全局 default；替换时 Close 旧 default。
func SetDefaultLogger(logger Logger) {
	cloned := logger.Clone()
	if src, ok := logger.(*coreLogger); ok {
		if dst, ok := cloned.(*coreLogger); ok {
			transferOwnership(src, dst)
		}
	}

	defaultMu.Lock()
	prev := defaultLogger
	defaultLogger = cloned
	defaultMu.Unlock()

	if prev != nil {
		if err := prev.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "golog: close previous default logger: %v\n", err)
		}
	}
}

// Debug logs a debug message using the default logger.
func Debug(args ...any) {
	getDefault().Debug(args...)
}

// Debugf logs a formatted debug message using the default logger.
func Debugf(format string, args ...any) {
	getDefault().Debugf(format, args...)
}

// Debugw logs a structured debug message using the default logger.
func Debugw(msg string, keysAndValues ...any) {
	getDefault().Debugw(msg, keysAndValues...)
}

// Info logs an info message using the default logger.
func Info(args ...any) {
	getDefault().Info(args...)
}

// Infof logs a formatted info message using the default logger.
func Infof(format string, args ...any) {
	getDefault().Infof(format, args...)
}

// Infow logs a structured info message using the default logger.
func Infow(msg string, keysAndValues ...any) {
	getDefault().Infow(msg, keysAndValues...)
}

// Warn logs a warning message using the default logger.
func Warn(args ...any) {
	getDefault().Warn(args...)
}

// Warnf logs a formatted warning message using the default logger.
func Warnf(format string, args ...any) {
	getDefault().Warnf(format, args...)
}

// Warnw logs a structured warning message using the default logger.
func Warnw(msg string, keysAndValues ...any) {
	getDefault().Warnw(msg, keysAndValues...)
}

// Error logs an error message using the default logger.
func Error(args ...any) {
	getDefault().Error(args...)
}

// Errorf logs a formatted error message using the default logger.
func Errorf(format string, args ...any) {
	getDefault().Errorf(format, args...)
}

// Errorw logs a structured error message using the default logger.
func Errorw(msg string, keysAndValues ...any) {
	getDefault().Errorw(msg, keysAndValues...)
}

// Fatal logs a fatal message using the default logger and exits.
func Fatal(args ...any) {
	getDefault().Fatal(args...)
}

// Fatalf logs a formatted fatal message using the default logger and exits.
func Fatalf(format string, args ...any) {
	getDefault().Fatalf(format, args...)
}

// Fatalw logs a structured fatal message using the default logger and exits.
func Fatalw(msg string, keysAndValues ...any) {
	getDefault().Fatalw(msg, keysAndValues...)
}

// SetLevel sets the log level on the default logger.
func SetLevel(lvl Level) {
	getDefault().SetLevel(lvl)
}

// GetLevel returns the current log level of the default logger.
func GetLevel() Level {
	return getDefault().GetLevel()
}

// AddCallerSkip increases caller skip on the default logger.
func AddCallerSkip(skip int) {
	getDefault().AddCallerSkip(skip)
}

// Sync 刷盘全局默认 Logger，不关闭
func Sync() error {
	return getDefault().Sync()
}

// Close 关闭全局默认 Logger（刷盘并释放文件写入器）
func Close() error {
	defaultMu.RLock()
	l := defaultLogger
	defaultMu.RUnlock()
	if l == nil {
		return nil
	}
	return l.Close()
}
