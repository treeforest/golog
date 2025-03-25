package main

import "github.com/treeforest/golog"

func main() {
	logConfig := golog.NewConfig(golog.WithJsonFormat(true))
	customLogger := golog.NewLogger(logConfig)
	golog.SetDefaultLogger(customLogger)
	defer golog.Sync()

	golog.Debug("debug message")
	golog.Info("info message")
	golog.Warn("warn message")
	golog.Errorw("error message", "key", "value")
}
