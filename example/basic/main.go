package main

import (
	"github.com/treeforest/golog/v2"
)

func main() {
	cfg := golog.NewConfig(
		golog.WithLogInConsole(true),
		golog.WithLevel(golog.DebugLevel),
	)
	golog.SetDefaultLogger(golog.NewLogger(cfg))

	defer func() {
		if err := golog.Sync(); err != nil {
			panic(err)
		}
	}()

	golog.Debug("debug message")
	golog.Info("info message")
	golog.Warn("warn message")
	golog.Error("error message")

	golog.SetLevel(golog.WarnLevel)

	golog.Debug("debug message") // 不会输出
	golog.Info("info message")   // 不会输出
	golog.Warn("warn message")
	golog.Error("error message")

	customLogger := golog.NewLogger(golog.NewConfig(golog.WithLevel(golog.DebugLevel)))
	defer func() {
		if err := customLogger.Sync(); err != nil {
			panic(err)
		}
	}()
	customLogger.Debug("hello world")
}
