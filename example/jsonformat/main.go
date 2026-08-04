package main

import (
	"github.com/treeforest/golog/v2"
	"go.uber.org/zap"
)

// JSON 格式输出（控制台）；也可用 Zap() 写强类型字段
func main() {
	cfg := golog.NewConfig(
		golog.WithModule("api"),
		golog.WithJsonFormat(true),
		golog.WithUseUTC(true),
		golog.WithLogInConsole(true),
		golog.WithLevel(golog.DebugLevel),
	)
	logger := golog.MustNewLogger(cfg)
	golog.SetDefaultLogger(logger)
	defer func() { _ = golog.Close() }()

	golog.Debug("debug message")
	golog.Info("info message")
	golog.Warn("warn message")
	golog.Errorw("error message", "key", "value", "code", 500)

	logger.Zap().Info("typed zap fields",
		zap.String("route", "/v1/health"),
		zap.Int("latency_ms", 12),
	)
}
