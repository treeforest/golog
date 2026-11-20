package golog

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type ctxKey struct{}       // context 中存储 Logger 实例的键类型
type ctxFieldsKey struct{} // context 中存储全链路字段切片（[]zap.Field）的键类型

// ContextWithLogger 将指定的 Logger 实例绑定并存储到 context.Context 中返回。
func ContextWithLogger(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// ContextWithTrace 将标准分布式链路追踪三元组（trace_id, request_id, 以及可选的 span_id）注入 context.Context。
//
// 使用示例：
//
//	// 注入 trace_id 与 request_id
//	ctx = golog.ContextWithTrace(ctx, "trace-app-881920", "req-mob-001")
//
//	// 注入完整三元组（包含 span_id）
//	ctx = golog.ContextWithTrace(ctx, "trace-app-881920", "req-mob-001", "span-001")
func ContextWithTrace(ctx context.Context, traceID, requestID string, spanID ...string) context.Context {
	fields := make([]any, 0, 6)
	if traceID != "" {
		fields = append(fields, "trace_id", traceID)
	}
	if requestID != "" {
		fields = append(fields, "request_id", requestID)
	}
	if len(spanID) > 0 && spanID[0] != "" {
		fields = append(fields, "span_id", spanID[0])
	}
	return ContextWithFields(ctx, fields...)
}

// ContextWithFields 将通用的结构化业务字段（如租户 ID、环境标识、用户 ID 等）注入 context.Context。
// 下游所有通过 LoggerFromContext 或 *Ctx 系列方法打印的日志均会自动携带这些字段。
//
// 使用示例：
//
//	ctx = golog.ContextWithFields(ctx, "tenant_id", "VIP-001", "client_ip", "10.0.0.1")
func ContextWithFields(ctx context.Context, keysAndValues ...any) context.Context {
	if len(keysAndValues) == 0 {
		return ctx
	}
	newFields := toZapFields(keysAndValues...)
	if existing, ok := ctx.Value(ctxFieldsKey{}).([]zap.Field); ok && len(existing) > 0 {
		merged := make([]zap.Field, len(existing)+len(newFields))
		copy(merged, existing)
		copy(merged[len(existing):], newFields)
		return context.WithValue(ctx, ctxFieldsKey{}, merged)
	}
	return context.WithValue(ctx, ctxFieldsKey{}, newFields)
}

// toZapFields 将成对出现的键值切片转换为 []zap.Field。
func toZapFields(keysAndValues ...any) []zap.Field {
	if len(keysAndValues) == 0 {
		return nil
	}
	fields := make([]zap.Field, 0, (len(keysAndValues)+1)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key, ok := keysAndValues[i].(string)
			if !ok {
				key = fmt.Sprintf("%v", keysAndValues[i])
			}
			fields = append(fields, zap.Any(key, keysAndValues[i+1]))
		} else {
			fields = append(fields, zap.Any("EXTRA_VALUE_AT_END", keysAndValues[i]))
		}
	}
	return fields
}

// LoggerFromContext 从 context.Context 中提取 Logger 实例。
//
// 提取规则：
//  1. 若 context 为 nil，返回绑定了全局默认 Logger 的实例；
//  2. 若 context 中包含由 ContextWithLogger 注入的 Logger，优先使用该实例；
//  3. 若 context 中无注入的 Logger，则回退使用全局默认 Logger；
//  4. 最终均会自动提取 context 中的上下文追踪字段并派生为不可变 Logger 返回。
func LoggerFromContext(ctx context.Context) Logger {
	if ctx == nil {
		return getDefault()
	}
	if l, ok := ctx.Value(ctxKey{}).(Logger); ok && l != nil {
		return withContextFields(l, ctx)
	}
	return withContextFields(getDefault(), ctx)
}

// withContextFields 从 context 中提取链路追踪与业务字段，并派生绑定了字段的独立 Logger 副本。
func withContextFields(l Logger, ctx context.Context) Logger {
	cl, ok := l.(*coreLogger)
	if !ok {
		return l
	}
	fields, ok := ctx.Value(ctxFieldsKey{}).([]zap.Field)
	if !ok || len(fields) == 0 {
		return l
	}
	zl := cl.zapLogger.With(fields...)
	rw := cl.rotWriter
	if rw != nil {
		rw.acquire()
	}
	return &coreLogger{
		SugaredLogger: zl.Sugar(),
		zapLogger:     zl,
		atomicLevel:   cl.atomicLevel,
		rotWriter:     rw,
	}
}

// getContextLoggerWithSkip 获取适用于包级全局 *Ctx 系列函数的 SugaredLogger（增加了 1 层 caller skip 保证行号精准）。
func getContextLoggerWithSkip(ctx context.Context) *zap.SugaredLogger {
	l := LoggerFromContext(ctx)
	if cl, ok := l.(*coreLogger); ok {
		return cl.zapLogger.WithOptions(zap.AddCallerSkip(1)).Sugar()
	}
	return l.(*coreLogger).SugaredLogger
}

// DebugCtx 从 context 中解析 Logger 与全链路追踪标识，记录 DEBUG 级别日志。
func DebugCtx(ctx context.Context, args ...any) {
	getContextLoggerWithSkip(ctx).Debug(args...)
}

// DebugfCtx 从 context 中解析 Logger 与全链路追踪标识，根据格式化模板记录 DEBUG 级别日志。
func DebugfCtx(ctx context.Context, format string, args ...any) {
	getContextLoggerWithSkip(ctx).Debugf(format, args...)
}

// DebugwCtx 从 context 中解析 Logger 与全链路追踪标识，记录包含键值对的 DEBUG 级别结构化日志。
func DebugwCtx(ctx context.Context, msg string, keysAndValues ...any) {
	getContextLoggerWithSkip(ctx).Debugw(msg, keysAndValues...)
}

// InfoCtx 从 context 中解析 Logger 与全链路追踪标识，记录 INFO 级别日志。
func InfoCtx(ctx context.Context, args ...any) {
	getContextLoggerWithSkip(ctx).Info(args...)
}

// InfofCtx 从 context 中解析 Logger 与全链路追踪标识，根据格式化模板记录 INFO 级别日志。
func InfofCtx(ctx context.Context, format string, args ...any) {
	getContextLoggerWithSkip(ctx).Infof(format, args...)
}

// InfowCtx 从 context 中解析 Logger 与全链路追踪标识，记录包含键值对的 INFO 级别结构化日志。
func InfowCtx(ctx context.Context, msg string, keysAndValues ...any) {
	getContextLoggerWithSkip(ctx).Infow(msg, keysAndValues...)
}

// WarnCtx 从 context 中解析 Logger 与全链路追踪标识，记录 WARN 级别日志。
func WarnCtx(ctx context.Context, args ...any) {
	getContextLoggerWithSkip(ctx).Warn(args...)
}

// WarnfCtx 从 context 中解析 Logger 与全链路追踪标识，根据格式化模板记录 WARN 级别日志。
func WarnfCtx(ctx context.Context, format string, args ...any) {
	getContextLoggerWithSkip(ctx).Warnf(format, args...)
}

// WarnwCtx 从 context 中解析 Logger 与全链路追踪标识，记录包含键值对的 WARN 级别结构化日志。
func WarnwCtx(ctx context.Context, msg string, keysAndValues ...any) {
	getContextLoggerWithSkip(ctx).Warnw(msg, keysAndValues...)
}

// ErrorCtx 从 context 中解析 Logger 与全链路追踪标识，记录 ERROR 级别日志。
func ErrorCtx(ctx context.Context, args ...any) {
	getContextLoggerWithSkip(ctx).Error(args...)
}

// ErrorfCtx 从 context 中解析 Logger 与全链路追踪标识，根据格式化模板记录 ERROR 级别日志。
func ErrorfCtx(ctx context.Context, format string, args ...any) {
	getContextLoggerWithSkip(ctx).Errorf(format, args...)
}

// ErrorwCtx 从 context 中解析 Logger 与全链路追踪标识，记录包含键值对的 ERROR 级别结构化日志。
func ErrorwCtx(ctx context.Context, msg string, keysAndValues ...any) {
	getContextLoggerWithSkip(ctx).Errorw(msg, keysAndValues...)
}

// FatalCtx 从 context 中解析 Logger 与全链路追踪标识，记录 FATAL 级别日志并退出进程（os.Exit(1)）。
func FatalCtx(ctx context.Context, args ...any) {
	getContextLoggerWithSkip(ctx).Fatal(args...)
}

// FatalfCtx 从 context 中解析 Logger 与全链路追踪标识，根据格式化模板记录 FATAL 级别日志并退出进程（os.Exit(1)）。
func FatalfCtx(ctx context.Context, format string, args ...any) {
	getContextLoggerWithSkip(ctx).Fatalf(format, args...)
}

// FatalwCtx 从 context 中解析 Logger 与全链路追踪标识，记录包含键值对的 FATAL 级别结构化日志并退出进程（os.Exit(1)）。
func FatalwCtx(ctx context.Context, msg string, keysAndValues ...any) {
	getContextLoggerWithSkip(ctx).Fatalw(msg, keysAndValues...)
}
