package main

import (
	"context"

	"github.com/treeforest/golog/v2"
)

func main() {
	logConfig := golog.NewConfig(
		golog.WithModule("user"),
		golog.WithComponent("login"),
		golog.WithPath("./logs/app.log"),
		golog.WithJsonFormat(true),
		golog.WithLogInFile(true),
		golog.WithLogInConsole(true),
		golog.WithRotationSizeMB(1),
	)
	golog.SetDefaultLogger(golog.NewLogger(logConfig))

	defer func() {
		if err := golog.Sync(); err != nil {
			panic(err)
		}
	}()

	for i := 0; i < 10000; i++ {
		golog.Info("info message")
		golog.Infow("info kvs", "hello", "world")
	}
	
	ctx := golog.ContextWithTraceID(context.Background(), "trace-demo")
	golog.InfowCtx(ctx, "request handled", "status", 200)
}
