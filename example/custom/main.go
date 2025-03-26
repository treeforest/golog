package main

import "github.com/treeforest/golog"

func main() {
	logConfig := golog.NewConfig(
		golog.WithModule("user"),   // 模块名
		golog.WithService("login"), // 服务名
		golog.WithJsonFormat(true), // 以json格式输出
	)
	golog.SetDefaultLogger(golog.NewLogger(logConfig))

	defer func() {
		if err := golog.Sync(); err != nil {
			panic(err)
		}
	}()

	golog.Debug("debug message")
	golog.Info("info message")

	golog.SetLevel(golog.InfoLevel)

	golog.Debug("debug message")
	golog.Info("info message")

	golog.Infow("info kvs", "hello", "world")
}
