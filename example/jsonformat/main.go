// 示例：云原生 JSON 结构化日志输出（适合 ELK / Loki / CloudWatch 收集与分析）。
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/treeforest/golog/v2"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("运行失败: %v", err)
	}
}

func run() error {
	fmt.Println("=== 云原生 JSON 结构化输出模式 ===")
	cfg := golog.NewConfig(
		golog.WithModule("gateway"),
		golog.WithComponent("http-inbound"),
		golog.WithJSONFormat(true), // 开启标准 JSON 输出格式
		golog.WithUseUTC(true),     // 时间戳使用 UTC (RFC3339Nano)，契约规范统一
		golog.WithShowLine(true),   // 记录源码行号方便溯源
		golog.WithLogInConsole(true),
		golog.WithLevel(golog.DebugLevel),
	)
	logger := golog.MustNewLogger(cfg)
	golog.SetDefaultLogger(logger)
	defer func() { _ = golog.Close() }()

	// 1. 普通文本转 JSON
	golog.Info("API 网关服务配置就绪")

	// 2. 结构化 API 访问请求日志
	start := time.Now()
	// 模拟请求处理耗时
	time.Sleep(15 * time.Millisecond)
	latency := time.Since(start)

	golog.Infow("HTTP 请求处理完毕",
		"method", "POST",
		"path", "/api/v1/orders",
		"status", 200,
		"latency_ms", latency.Milliseconds(),
		"client_ip", "203.0.113.195",
		"user_agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
	)

	// 3. 错误场景下的结构化指标输出
	golog.Errorw("下单业务校验未通过",
		"order_id", "ORD-20261120-0091",
		"user_id", 89412,
		"error_code", "ERR_INSUFFICIENT_INVENTORY",
		"sku_id", 100234,
		"requested_qty", 5,
		"available_qty", 1,
	)

	return nil
}
