// 示例：文件持久化与终端彩色双分流，以及基于 With() 的不可变子模块派生（Sub-Logger）。
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/treeforest/golog/v2"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("运行失败: %v", err)
	}
}

func run() error {
	logFilePath := "./logs/custom.log"
	_ = os.Remove(logFilePath) // 演示前清理旧日志

	fmt.Println("=== 1. 初始化文件与控制台双输出配置 ===")
	cfg := golog.NewConfig(
		golog.WithModule("shop-system"),
		golog.WithPath(logFilePath),
		golog.WithLogInFile(true),    // 持久化到文件（纯文本，无 ANSI 控制字符）
		golog.WithLogInConsole(true), // 同步到控制台（带 ANSI 彩色渲染）
		golog.WithShowColor(true),
		golog.WithLevel(golog.InfoLevel),
		golog.WithRotationSizeMB(10), // 单文件 10MB 滚动
	)
	rootLogger := golog.MustNewLogger(cfg)
	golog.SetDefaultLogger(rootLogger)
	defer func() { _ = golog.Close() }()

	golog.Info("主应用启动，正在初始化各核心业务子模块...")

	fmt.Println("\n=== 2. 使用 With() 派生业务专属子模块 Logger（不可变，不互相污染） ===")

	// 派生出订单模块专用 Logger，自动附带固定的环境与模块上下文
	orderLogger := rootLogger.With("sub_module", "order_service", "env", "production")
	orderLogger.Infow("订单系统就绪，接收新订单", "order_no", "ORD-20261120-1001", "amount_cents", 9900)

	// 派生出支付通道专属 Logger
	paymentLogger := rootLogger.With("sub_module", "payment_channel", "channel", "alipay")
	paymentLogger.Infow("发起渠道扣款", "trade_no", "ALI-88912384712", "amount_cents", 9900)
	paymentLogger.Warnw("渠道扣款通知稍有延迟", "retry_interval_sec", 3)

	// 原 rootLogger 保持纯净，不受子模块派生字段污染
	rootLogger.Info("根 Logger 依然保持干净的全局上下文")

	fmt.Printf("\n=== 3. 验证日志已持久化落盘 ===\n日志文件已写入: %s\n", logFilePath)
	return nil
}
