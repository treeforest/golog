package golog

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
)

// Logger 是通用的结构化日志记录器接口。
//
// 该接口提供了常见的 Debug、Info、Warn、Error、Fatal 级别的日志输出能力，
// 涵盖普通输出、格式化输出（f 系列）与结构化键值输出（w 系列），
// 同时提供流式不可变派生（With / WithContext / WithCallerSkip）与生命周期管理方法。
// 该接口的所有实现必须保证并发安全（Concurrency Safe）。
type Logger interface {
	// Debug 使用默认参数列表记录 DEBUG 级别的日志。
	Debug(args ...any)
	// Debugf 根据指定的格式化模板记录 DEBUG 级别的日志。
	Debugf(format string, args ...any)
	// Debugw 记录包含成对键值对的 DEBUG 级别结构化日志。
	Debugw(msg string, keysAndValues ...any)

	// Info 使用默认参数列表记录 INFO 级别的日志。
	Info(args ...any)
	// Infof 根据指定的格式化模板记录 INFO 级别的日志。
	Infof(format string, args ...any)
	// Infow 记录包含成对键值对的 INFO 级别结构化日志。
	Infow(msg string, keysAndValues ...any)

	// Warn 使用默认参数列表记录 WARN 级别的日志。
	Warn(args ...any)
	// Warnf 根据指定的格式化模板记录 WARN 级别的日志。
	Warnf(format string, args ...any)
	// Warnw 记录包含成对键值对的 WARN 级别结构化日志。
	Warnw(msg string, keysAndValues ...any)

	// Error 使用默认参数列表记录 ERROR 级别的日志。
	Error(args ...any)
	// Errorf 根据指定的格式化模板记录 ERROR 级别的日志。
	Errorf(format string, args ...any)
	// Errorw 记录包含成对键值对的 ERROR 级别结构化日志。
	Errorw(msg string, keysAndValues ...any)

	// Fatal 使用默认参数列表记录 FATAL 级别的日志，并在输出后终止进程（os.Exit(1)）。
	Fatal(args ...any)
	// Fatalf 根据指定的格式化模板记录 FATAL 级别的日志，并在输出后终止进程（os.Exit(1)）。
	Fatalf(format string, args ...any)
	// Fatalw 记录包含成对键值对的 FATAL 级别结构化日志，并在输出后终止进程（os.Exit(1)）。
	Fatalw(msg string, keysAndValues ...any)

	// WithContext 派生绑定了指定 context 字段（如 trace_id / request_id）的 Logger 独立副本（不可变操作）。
	WithContext(ctx context.Context) Logger
	// With 派生附加了指定键值对字段的 Logger 独立副本（不可变操作）。
	With(keysAndValues ...any) Logger
	// WithCallerSkip 派生调整了调用栈跳跃层级的 Logger 独立副本（不可变操作）。
	WithCallerSkip(skip int) Logger

	// SetLevel 动态调整当前 Logger 的日志级别（线程安全）。
	SetLevel(lvl Level)
	// GetLevel 获取当前 Logger 的日志级别。
	GetLevel() Level

	// Sync 将缓冲区中未写入的数据强制刷入底层存储（不关闭文件句柄）。
	Sync() error
	// Close 刷盘并减少底层写入器的引用计数。当所有引用均释放后，关闭底层文件句柄。
	Close() error
}

// coreLogger 是基于 Uber Zap 实现的 Logger 核心结构体。
type coreLogger struct {
	*zap.SugaredLogger
	zapLogger   *zap.Logger     // 底层 zap.Logger 实例
	atomicLevel zap.AtomicLevel // 原子级别控制器，支持运行时无锁并发读写
	rotWriter   *rotatingWriter // 底层文件轮转写入器（通过引用计数复用与释放）
	mu          sync.Mutex      // 互斥锁，保护并发操作
	closeOnce   sync.Once       // 确保 Close 操作只执行一次
}

var _ Logger = (*coreLogger)(nil)

// SetLevel 动态调整当前 coreLogger 的最低输出级别。
func (l *coreLogger) SetLevel(lvl Level) {
	l.atomicLevel.SetLevel(lvl.ZapLevel())
}

// GetLevel 获取当前 coreLogger 的生效级别。
func (l *coreLogger) GetLevel() Level {
	lvl, err := ParseLevel(l.atomicLevel.Level().String())
	if err != nil {
		return InfoLevel
	}
	return lvl
}

// WithCallerSkip 返回调整了 caller skip 的独立 Logger 副本（不可变）。
func (l *coreLogger) WithCallerSkip(skip int) Logger {
	l.mu.Lock()
	zl := l.zapLogger.WithOptions(zap.AddCallerSkip(skip))
	rw := l.rotWriter
	al := l.atomicLevel
	l.mu.Unlock()

	if rw != nil {
		rw.acquire()
	}
	return &coreLogger{
		SugaredLogger: zl.Sugar(),
		zapLogger:     zl,
		atomicLevel:   al,
		rotWriter:     rw,
	}
}

// With 返回附加了额外结构化键值对的独立 Logger 副本（不可变）。
func (l *coreLogger) With(keysAndValues ...any) Logger {
	l.mu.Lock()
	zl := l.zapLogger
	rw := l.rotWriter
	al := l.atomicLevel
	l.mu.Unlock()

	sugar := zl.Sugar().With(keysAndValues...)
	if rw != nil {
		rw.acquire()
	}
	return &coreLogger{
		SugaredLogger: sugar,
		zapLogger:     sugar.Desugar(),
		atomicLevel:   al,
		rotWriter:     rw,
	}
}

// WithContext 返回绑定了上下文链路字段（trace_id / request_id 等）的独立 Logger 副本。
func (l *coreLogger) WithContext(ctx context.Context) Logger {
	return withContextFields(l, ctx)
}

// Sync 执行日志缓冲区刷盘。
func (l *coreLogger) Sync() error {
	l.mu.Lock()
	zl := l.zapLogger
	l.mu.Unlock()
	return zl.Sync()
}

// Close 执行刷盘并递减底层文件写入器的引用计数；当引用归零时由底层安全释放句柄。
func (l *coreLogger) Close() error {
	var err error
	l.closeOnce.Do(func() {
		err = l.Sync()
		if l.rotWriter != nil {
			l.rotWriter.release()
		}
	})
	return err
}

var (
	defaultLogger atomic.Pointer[coreLogger] // 全局默认 Logger 指针（原子读写，免锁）
	defaultOnce   sync.Once                  // 保证全局默认 Logger 仅懒加载初始化一次
)

// initDefaultLogger 执行全局默认 Logger 的懒加载初始化。
func initDefaultLogger() *coreLogger {
	defaultOnce.Do(func() {
		root := MustNewLogger(defaultConfig())
		rootCore, ok := root.(*coreLogger)
		if !ok {
			panic("golog: internal error: root logger is not *coreLogger")
		}
		defaultLogger.Store(rootCore)
	})
	return defaultLogger.Load()
}

// getDefault 获取全局默认 Logger 实例（无额外包装，skip=0）。
func getDefault() *coreLogger {
	if l := defaultLogger.Load(); l != nil {
		return l
	}
	return initDefaultLogger()
}

// getPackageLogger 获取专门适配包级全局函数（如 golog.Info）的 SugaredLogger（内建 AddCallerSkip(1)）。
func getPackageLogger() *zap.SugaredLogger {
	base := getDefault()
	return base.zapLogger.WithOptions(zap.AddCallerSkip(1)).Sugar()
}

// SetDefaultLogger 设置全局默认 Logger。替换前将自动关闭并安全释放旧的全局默认 Logger。
func SetDefaultLogger(logger Logger) {
	newCore, ok := logger.(*coreLogger)
	if !ok {
		panic("golog: SetDefaultLogger requires *coreLogger implementation")
	}

	prev := defaultLogger.Swap(newCore)
	if prev != nil {
		if err := prev.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "golog: close previous default logger: %v\n", err)
		}
	}
}

// Debug 使用全局默认 Logger 记录 DEBUG 级别日志。
func Debug(args ...any) {
	getPackageLogger().Debug(args...)
}

// Debugf 使用全局默认 Logger 根据格式化模板记录 DEBUG 级别日志。
func Debugf(format string, args ...any) {
	getPackageLogger().Debugf(format, args...)
}

// Debugw 使用全局默认 Logger 记录包含键值对的 DEBUG 级别结构化日志。
func Debugw(msg string, keysAndValues ...any) {
	getPackageLogger().Debugw(msg, keysAndValues...)
}

// Info 使用全局默认 Logger 记录 INFO 级别日志。
func Info(args ...any) {
	getPackageLogger().Info(args...)
}

// Infof 使用全局默认 Logger 根据格式化模板记录 INFO 级别日志。
func Infof(format string, args ...any) {
	getPackageLogger().Infof(format, args...)
}

// Infow 使用全局默认 Logger 记录包含键值对的 INFO 级别结构化日志。
func Infow(msg string, keysAndValues ...any) {
	getPackageLogger().Infow(msg, keysAndValues...)
}

// Warn 使用全局默认 Logger 记录 WARN 级别日志。
func Warn(args ...any) {
	getPackageLogger().Warn(args...)
}

// Warnf 使用全局默认 Logger 根据格式化模板记录 WARN 级别日志。
func Warnf(format string, args ...any) {
	getPackageLogger().Warnf(format, args...)
}

// Warnw 使用全局默认 Logger 记录包含键值对的 WARN 级别结构化日志。
func Warnw(msg string, keysAndValues ...any) {
	getPackageLogger().Warnw(msg, keysAndValues...)
}

// Error 使用全局默认 Logger 记录 ERROR 级别日志。
func Error(args ...any) {
	getPackageLogger().Error(args...)
}

// Errorf 使用全局默认 Logger 根据格式化模板记录 ERROR 级别日志。
func Errorf(format string, args ...any) {
	getPackageLogger().Errorf(format, args...)
}

// Errorw 使用全局默认 Logger 记录包含键值对的 ERROR 级别结构化日志。
func Errorw(msg string, keysAndValues ...any) {
	getPackageLogger().Errorw(msg, keysAndValues...)
}

// Fatal 使用全局默认 Logger 记录 FATAL 级别日志并终止程序（os.Exit(1)）。
func Fatal(args ...any) {
	getPackageLogger().Fatal(args...)
}

// Fatalf 使用全局默认 Logger 根据格式化模板记录 FATAL 级别日志并终止程序（os.Exit(1)）。
func Fatalf(format string, args ...any) {
	getPackageLogger().Fatalf(format, args...)
}

// Fatalw 使用全局默认 Logger 记录包含键值对的 FATAL 级别结构化日志并终止程序（os.Exit(1)）。
func Fatalw(msg string, keysAndValues ...any) {
	getPackageLogger().Fatalw(msg, keysAndValues...)
}

// SetLevel 设置全局默认 Logger 的日志级别。
func SetLevel(lvl Level) {
	getDefault().SetLevel(lvl)
}

// GetLevel 获取全局默认 Logger 的日志级别。
func GetLevel() Level {
	return getDefault().GetLevel()
}

// Sync 将全局默认 Logger 缓冲区中的日志数据强制刷入底层存储（不关闭文件句柄）。
func Sync() error {
	return getDefault().Sync()
}

// Close 关闭全局默认 Logger，执行最终刷盘并安全释放底层文件句柄。
func Close() error {
	l := defaultLogger.Load()
	if l == nil {
		return nil
	}
	return l.Close()
}
