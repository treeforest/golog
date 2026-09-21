// 基础用法：控制台输出、动态级别、独立 Logger 实例。
package main

import (
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
		golog.WithModule("demo"),
		golog.WithLogInConsole(true),
		golog.WithShowColor(true),
		golog.WithLevel(golog.DebugLevel),
	)
	golog.SetDefaultLogger(golog.MustNewLogger(cfg))
	defer func() {
		if err := golog.Close(); err != nil {
			log.Printf("close default logger: %v", err)
		}
	}()

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
	defer func() {
		if err := local.Close(); err != nil {
			log.Printf("close local logger: %v", err)
		}
	}()
	local.Debug("local logger debug")
	return nil
}
