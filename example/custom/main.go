// 自定义配置：模块/组件、文件+控制台、Context 透传。
package main

import (
	"context"
	"log"

	"github.com/treeforest/golog/v2"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := golog.NewConfig(
		golog.WithModule("user"),
		golog.WithComponent("login"),
		golog.WithPath("./logs/custom.log"),
		golog.WithLevel(golog.InfoLevel),
		golog.WithLogInFile(true),
		golog.WithLogInConsole(true),
		golog.WithShowColor(true),
		golog.WithRotationHours(0),
		golog.WithRotationSizeMB(100),
	)
	golog.SetDefaultLogger(golog.MustNewLogger(cfg))
	defer func() {
		if err := golog.Close(); err != nil {
			log.Printf("close default logger: %v", err)
		}
	}()

	golog.Info("service started")
	golog.Infow("user login", "user_id", 1001, "ip", "10.0.0.1")

	ctx := golog.ContextWithTraceID(context.Background(), "trace-demo")
	ctx = golog.ContextWithRequestID(ctx, "req-001")
	logger := golog.LoggerFromContext(ctx)
	logger.Infow("request handled", "status", 200)
	return nil
}
