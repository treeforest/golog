// Package golog 提供了一个基于 Uber Zap 构建的高性能、生产级结构化日志库。
//
// 该库兼顾了 Uber Zap 的极致性能与类似 Logrus 的极简易用性，主要特性包括：
//
//   - 工业级文件轮转：基于 gopkg.in/natefinch/lumberjack.v2 实现按文件大小与天数自动滚动；
//   - 并发安全生命周期：同路径进程内单例复用，通过带锁引用计数安全管理底层写入器的开启与释放；
//   - 不可变派生（Immutable）：提供 With、WithContext、WithCallerSkip 等流式不可变派生能力，杜绝状态污染；
//   - 上下文链路透传：无缝支持 context.Context 中 trace_id 与 request_id 等链路字段的提取与自动注入；
//   - 高并发无锁读取：全局默认 Logger 采用 sync/atomic.Pointer 实现，消除多核高并发下的读写锁竞争与缓存行颠簸；
//   - 分离式编码输出：支持文件端纯文本输出与终端彩色级别输出的独立配置，支持自定义 io.Writer 扩展；
//   - 精准调用栈定位：彻底校准 Caller Skip 跳跃层级，确保全局方法与 Context 派生实例均精准记录调用行号。
//
// # 基础使用示例
//
//	package main
//
//	import (
//		"github.com/treeforest/golog/v2"
//	)
//
//	func main() {
//		// 1. 初始化配置
//		cfg := golog.NewConfig(
//			golog.WithPath("/var/log/app/app.log"),
//			golog.WithLogInFile(true),
//			golog.WithLogInConsole(true),
//			golog.WithLevel(golog.InfoLevel),
//		)
//
//		// 2. 设置为全局默认 Logger
//		golog.SetDefaultLogger(golog.MustNewLogger(cfg))
//		// 进程退出前务必调用 Close，刷盘并释放底层文件句柄
//		defer func() { _ = golog.Close() }()
//
//		// 3. 打印日志
//		golog.Info("服务启动成功")
//		golog.Infof("监听端口: %d", 8080)
//		golog.Infow("用户登录", "user_id", 1001, "ip", "192.168.1.1")
//	}
//
// # Context 链路追踪与全链路标签
//
//	// 单行优雅注入分布式链路三元组（trace_id, request_id, 可选 span_id）
//	ctx := golog.ContextWithTrace(context.Background(), "trace-abc-123", "req-xyz-789", "span-001")
//	// 注入业务通用标签（如租户、环境）
//	ctx = golog.ContextWithFields(ctx, "tenant_id", "VIP-001")
//
//	// 方式 A：从 Context 获取绑定的 Logger 实例直接使用（推荐热路径使用）
//	logger := golog.LoggerFromContext(ctx)
//	logger.Infow("处理请求", "status", 200)
//
//	// 方式 B：使用全局 Context 便捷函数
//	golog.InfowCtx(ctx, "处理请求", "status", 200)
//
// # 并发安全与优雅退出
//
// 所有导出的 Logger 实例以及包级全局函数均为并发安全（Goroutine-safe），可以在任意并发环境下调用。
//
// 在应用正常终止或捕获到系统退出信号时，应当调用 Close() 或 golog.Close()，确保缓冲区中的日志彻底刷入磁盘，
// 并安全关闭文件流。在长生命周期运行过程中，也可以按需调用 Sync() 强制刷盘。
package golog
