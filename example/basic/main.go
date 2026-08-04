package main

import (
	"github.com/treeforest/golog/v2"
)

// 基础用法：控制台输出、动态级别、独立 Logger 实例
func main() {
	cfg := golog.NewConfig(
		golog.WithModule("demo"),
		golog.WithLogInConsole(true),
		golog.WithShowColor(true),
		golog.WithLevel(golog.DebugLevel),
	)
	golog.SetDefaultLogger(golog.MustNewLogger(cfg))
	defer func() { _ = golog.Close() }()

	golog.Debug("debug message")
	golog.Info("info message")
	golog.Warn("warn message")
	golog.Error("error message")

	golog.SetLevel(golog.WarnLevel)
	golog.Debug("debug skipped") // 不会输出
	golog.Info("info skipped")   // 不会输出
	golog.Warn("warn after SetLevel")
	golog.Error("error after SetLevel")

	local := golog.MustNewLogger(golog.NewConfig(
		golog.WithLevel(golog.DebugLevel),
		golog.WithLogInConsole(true),
		golog.WithShowColor(false),
	))
	defer func() { _ = local.Close() }()
	local.Debug("local logger debug")
}
