package main

import (
	"github.com/treeforest/golog"
)

func main() {
	defer golog.Sync()

	golog.Debug("debug message")
	golog.Info("info message")
	golog.Warn("warn message")
	golog.Error("error message")

	// 更改日志级别
	golog.SetLevel(golog.WarnLevel)

	golog.Debug("debug message") // 不会输出
	golog.Info("info message")   // 不会输出
	golog.Warn("warn message")
	golog.Error("error message")

	customLogger := golog.NewLogger(nil)
	defer customLogger.Sync()
	customLogger.Debug("hello world")
}
