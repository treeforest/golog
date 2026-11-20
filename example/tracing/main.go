// 示例：分布式链路追踪与 Context 传播（模拟 HTTP 网关 -> 业务服务 -> 数据仓储层）。
package main

import (
	"context"
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
	fmt.Println("=== 分布式链路追踪 (Trace ID & Request ID) 模拟演示 ===")
	cfg := golog.NewConfig(
		golog.WithModule("eshop-api"),
		golog.WithLogInConsole(true),
		golog.WithShowColor(true),
		golog.WithShowLine(true), // 验证全链路精准代码行号
	)
	golog.SetDefaultLogger(golog.MustNewLogger(cfg))
	defer func() { _ = golog.Close() }()

	// 模拟请求 A：来自移动端 APP 的结账请求（单行优雅注入链路三元组及租户信息）
	fmt.Println("\n--- 处理请求 A (Client: Mobile APP) ---")
	ctxA := golog.ContextWithTrace(context.Background(), "trace-app-881920", "req-mob-001", "span-001")
	ctxA = golog.ContextWithFields(ctxA, "tenant_id", "VIP-China", "client_platform", "iOS")
	handleHTTPRequest(ctxA, "/api/checkout", 10001)

	// 模拟请求 B：来自后台管理系统的退款请求
	fmt.Println("\n--- 处理请求 B (Client: Admin Console) ---")
	ctxB := golog.ContextWithTrace(context.Background(), "trace-adm-334411", "req-adm-992")
	handleHTTPRequest(ctxB, "/api/refund", 10002)

	return nil
}

// 模拟 HTTP 入口处理层
func handleHTTPRequest(ctx context.Context, apiPath string, userID int64) {
	// 推荐方式：在方法入口从 ctx 获取绑定的 logger 并复用
	logger := golog.LoggerFromContext(ctx)
	logger.Infow("网关收到外部 HTTP 请求", "path", apiPath, "user_id", userID)

	// 传递 ctx 到业务逻辑服务层
	processBusinessService(ctx, userID)
}

// 模拟业务逻辑服务层 (Service)
func processBusinessService(ctx context.Context, userID int64) {
	// 方式 B：也可直接使用全局便捷函数 golog.InfowCtx 记录
	golog.InfowCtx(ctx, "业务层: 开始校验用户信用与账户状态", "user_id", userID)

	// 传递 ctx 到数据访问层 (Repository)
	queryDatabase(ctx, userID)

	golog.InfowCtx(ctx, "业务层: 业务逻辑校验通过，完成事务提交", "status", "success")
}

// 模拟数据访问持久层 (Repository)
func queryDatabase(ctx context.Context, userID int64) {
	logger := golog.LoggerFromContext(ctx)
	logger.Infow("数据库层: 执行 SQL 查询", "query", "SELECT balance FROM accounts WHERE user_id = ?", "latency_ms", 3)
}
