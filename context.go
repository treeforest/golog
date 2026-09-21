package golog

import (
	"context"

	"go.uber.org/zap"
)

type ctxKey struct{}       // context 中存储 Logger 的键
type traceIDKey struct{}   // context 中存储 trace_id 的键
type requestIDKey struct{} // context 中存储 request_id 的键

// ContextWithLogger 将 Logger 存入 context
func ContextWithLogger(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// ContextWithTraceID 将 trace_id 存入 context
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// ContextWithRequestID 将 request_id 存入 context
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// LoggerFromContext 从 context 获取 Logger；若无则返回全局默认 Logger
func LoggerFromContext(ctx context.Context) Logger {
	if ctx == nil {
		return getDefault()
	}
	if l, ok := ctx.Value(ctxKey{}).(Logger); ok && l != nil {
		return withContextFields(l, ctx)
	}
	return withContextFields(getDefault(), ctx)
}

// withContextFields 从 context 提取 trace_id / request_id 并绑定到派生 Logger
func withContextFields(l Logger, ctx context.Context) Logger {
	cl, ok := l.(*coreLogger)
	if !ok {
		return l
	}
	fields := contextFields(ctx)
	if len(fields) == 0 {
		return l
	}
	zl := cl.zapLogger.With(fields...)
	return &coreLogger{
		SugaredLogger: zl.Sugar(),
		zapLogger:     zl,
		atomicLevel:   cl.atomicLevel,
		rotWriter:     cl.rotWriter,
		ownsWriter:    false, // Context 派生不取得所有权
	}
}

// contextFields 从 context 构建 zap 字段列表
func contextFields(ctx context.Context) []zap.Field {
	var fields []zap.Field
	if v, ok := ctx.Value(traceIDKey{}).(string); ok && v != "" {
		fields = append(fields, zap.String("trace_id", v))
	}
	if v, ok := ctx.Value(requestIDKey{}).(string); ok && v != "" {
		fields = append(fields, zap.String("request_id", v))
	}
	return fields
}

// DebugCtx logs a debug message from the context logger.
func DebugCtx(ctx context.Context, args ...any) {
	LoggerFromContext(ctx).Debug(args...)
}

// DebugfCtx logs a formatted debug message from the context logger.
func DebugfCtx(ctx context.Context, format string, args ...any) {
	LoggerFromContext(ctx).Debugf(format, args...)
}

// DebugwCtx logs a structured debug message from the context logger.
func DebugwCtx(ctx context.Context, msg string, keysAndValues ...any) {
	LoggerFromContext(ctx).Debugw(msg, keysAndValues...)
}

// InfoCtx logs an info message from the context logger.
func InfoCtx(ctx context.Context, args ...any) {
	LoggerFromContext(ctx).Info(args...)
}

// InfofCtx logs a formatted info message from the context logger.
func InfofCtx(ctx context.Context, format string, args ...any) {
	LoggerFromContext(ctx).Infof(format, args...)
}

// InfowCtx logs a structured info message from the context logger.
func InfowCtx(ctx context.Context, msg string, keysAndValues ...any) {
	LoggerFromContext(ctx).Infow(msg, keysAndValues...)
}

// WarnCtx logs a warning message from the context logger.
func WarnCtx(ctx context.Context, args ...any) {
	LoggerFromContext(ctx).Warn(args...)
}

// WarnfCtx logs a formatted warning message from the context logger.
func WarnfCtx(ctx context.Context, format string, args ...any) {
	LoggerFromContext(ctx).Warnf(format, args...)
}

// WarnwCtx logs a structured warning message from the context logger.
func WarnwCtx(ctx context.Context, msg string, keysAndValues ...any) {
	LoggerFromContext(ctx).Warnw(msg, keysAndValues...)
}

// ErrorCtx logs an error message from the context logger.
func ErrorCtx(ctx context.Context, args ...any) {
	LoggerFromContext(ctx).Error(args...)
}

// ErrorfCtx logs a formatted error message from the context logger.
func ErrorfCtx(ctx context.Context, format string, args ...any) {
	LoggerFromContext(ctx).Errorf(format, args...)
}

// ErrorwCtx logs a structured error message from the context logger.
func ErrorwCtx(ctx context.Context, msg string, keysAndValues ...any) {
	LoggerFromContext(ctx).Errorw(msg, keysAndValues...)
}

// FatalCtx logs a fatal message from the context logger and exits.
func FatalCtx(ctx context.Context, args ...any) {
	LoggerFromContext(ctx).Fatal(args...)
}

// FatalfCtx logs a formatted fatal message from the context logger and exits.
func FatalfCtx(ctx context.Context, format string, args ...any) {
	LoggerFromContext(ctx).Fatalf(format, args...)
}

// FatalwCtx logs a structured fatal message from the context logger and exits.
func FatalwCtx(ctx context.Context, msg string, keysAndValues ...any) {
	LoggerFromContext(ctx).Fatalw(msg, keysAndValues...)
}

// WithContext 返回绑定了 context 字段（trace_id / request_id）的派生 Logger
func (l *coreLogger) WithContext(ctx context.Context) Logger {
	return withContextFields(l, ctx)
}
