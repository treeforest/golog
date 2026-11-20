// 示例：基础使用、控制台彩色高亮与运行时动态调整日志级别。
package main

import (
	"fmt"
	"log"

	"github.com/treeforest/golog/v2"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("运行失败: %v", err)
	}
}

func run() error {
	fmt.Println("=== 1. 初始化控制台彩色 Logger ===")
	cfg := golog.NewConfig(
		golog.WithModule("user-service"),
		golog.WithComponent("auth"),
		golog.WithLogInConsole(true),
		golog.WithShowColor(true),        // 开启终端彩色渲染
		golog.WithShowLine(true),         // 开启精确源码行号定位
		golog.WithLevel(golog.InfoLevel), // 初始级别设为 INFO
	)
	golog.SetDefaultLogger(golog.MustNewLogger(cfg))
	defer func() { _ = golog.Close() }()

	fmt.Println("\n=== 2. 不同级别的日志输出展示 ===")
	golog.Debug("这是 DEBUG 日志，当前级别为 INFO，该条不会打印")
	golog.Info("用户服务已成功启动")
	golog.Infof("当前服务监听端口: %d, 活跃协程数: %d", 8080, 24)
	golog.Warn("数据库连接池空闲连接数偏低 (当前: 2, 最小建议: 5)")
	golog.Errorw("第三方支付服务响应超时", "provider", "wechat", "timeout_ms", 3000, "retry_count", 3)

	fmt.Println("\n=== 3. 运行时动态调整日志级别（无需重启进程） ===")
	fmt.Printf("调整前级别: %s\n", golog.GetLevel())
	// 动态下调级别至 DEBUG（如用于线上排障）
	golog.SetLevel(golog.DebugLevel)
	fmt.Printf("调整后级别: %s\n", golog.GetLevel())
	golog.Debugw("动态开启 DEBUG: 输出详细排障上下文", "memory_alloc_mb", 128, "gc_pause_ns", 15200)

	// 动态提升级别至 WARN（压制普通信息）
	golog.SetLevel(golog.WarnLevel)
	golog.Info("此时 INFO 已被压制，不会输出")
	golog.Warn("警告信息依然可以正常输出")

	return nil
}
