package golog

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// sinkKind 标识日志输出目标类型，用于选择不同的编码策略
type sinkKind int

const (
	sinkFile    sinkKind = iota // 文件输出（纯文本，无 ANSI）
	sinkConsole                 // 控制台输出（可彩色级别）
	sinkExtra                   // 额外 io.Writer（与文件侧编码一致）
)

// noopSyncWriter 是对 zapcore.WriteSyncer 的装饰器实现
// 功能：包装一个基础 WriteSyncer 并覆盖其 Sync 方法，使其不执行任何同步操作
// 设计目的：解决对某些特殊输出流（如 os.Stderr）调用 Sync 方法时可能出现的
//
//	"sync /dev/stderr: invalid argument" 错误
//
// 适用场景：当底层写入器不需要或不支持 Sync 操作时使用
type noopSyncWriter struct {
	zapcore.WriteSyncer
}

// Sync 空实现，不执行任何操作
func (n *noopSyncWriter) Sync() error {
	return nil
}

// NewLogger 根据配置创建 Logger，可附加自定义 io.Writer 作为额外输出
func NewLogger(logConfig *Config, writer ...io.Writer) Logger {
	if logConfig == nil {
		logConfig = defaultConfig()
	}
	zapLogger, aLevel, rotWriter := initZapLogger(logConfig, writer...)
	return &coreLogger{
		SugaredLogger: zapLogger.Sugar(),
		zapLogger:     zapLogger,
		atomicLevel:   aLevel,
		rotWriter:     rotWriter,
	}
}

// initZapLogger 初始化 zap.Logger 及原子级别控制器
func initZapLogger(logConfig *Config, writer ...io.Writer) (*zap.Logger, zap.AtomicLevel, *rotatingWriter) {
	aLevel := zap.NewAtomicLevel()
	aLevel.SetLevel(logConfig.Level.ZapLevel())

	zapLogger, rotWriter := newZapLogger(logConfig, aLevel, writer...)
	return zapLogger, aLevel, rotWriter
}

// newZapLogger 创建底层 zap.Logger，按配置组装 Tee Core（文件 / 控制台 / 额外输出）
func newZapLogger(logConfig *Config, level zap.AtomicLevel, writer ...io.Writer) (*zap.Logger, *rotatingWriter) {
	var cores []zapcore.Core
	var rotWriter *rotatingWriter

	if logConfig.LogInFile {
		rw, err := newRotatingWriter(logConfig)
		if err != nil {
			log.Fatalf("new zap logger create rotating writer failed: %s", err)
		}
		rotWriter = rw
		cores = append(cores, buildCore(logConfig, level, zapcore.AddSync(rw), sinkFile))
	}

	if logConfig.LogInConsole {
		cores = append(cores, buildCore(logConfig, level, &noopSyncWriter{WriteSyncer: zapcore.AddSync(os.Stderr)}, sinkConsole))
	}

	for _, w := range writer {
		cores = append(cores, buildCore(logConfig, level, zapcore.AddSync(w), sinkExtra))
	}

	if len(cores) == 0 {
		log.Fatal("no log output target")
	}

	var core zapcore.Core
	if len(cores) == 1 {
		core = cores[0]
	} else {
		core = zapcore.NewTee(cores...)
	}

	name := loggerName(logConfig)
	logger := zap.New(core).Named(name)

	if logConfig.ShowLine {
		logger = logger.WithOptions(zap.AddCaller())
	}
	if lvl := logConfig.StackTraceLevel.ZapLevel(); lvl != zapcore.InvalidLevel {
		logger = logger.WithOptions(zap.AddStacktrace(lvl))
	}

	return logger, rotWriter
}

// loggerName 组装纯文本 logger 名称（不含 ANSI 颜色码）
func loggerName(cfg *Config) string {
	if cfg.Component != "" {
		return fmt.Sprintf("%s @%s", cfg.Module, cfg.Component)
	}
	return cfg.Module
}

// buildCore 为指定输出目标构建 zapcore.Core（含可选采样）
func buildCore(cfg *Config, level zap.AtomicLevel, ws zapcore.WriteSyncer, kind sinkKind) zapcore.Core {
	enc := newEncoder(cfg, kind)
	core := zapcore.NewCore(enc, ws, level)
	core = wrapSampling(cfg, core)
	return core
}

// wrapSampling 按配置包裹采样 Core
func wrapSampling(cfg *Config, core zapcore.Core) zapcore.Core {
	if !cfg.Sampling.Enabled {
		return core
	}
	initial := cfg.Sampling.Initial
	if initial <= 0 {
		initial = defaultSamplingInitial
	}
	thereafter := cfg.Sampling.Thereafter
	if thereafter <= 0 {
		thereafter = defaultSamplingThereafter
	}
	return zapcore.NewSamplerWithOptions(core, time.Second, initial, thereafter)
}

// newEncoder 根据格式与输出目标创建编码器
func newEncoder(cfg *Config, kind sinkKind) zapcore.Encoder {
	encCfg := encoderConfig(cfg, kind)
	if cfg.JsonFormat {
		return zapcore.NewJSONEncoder(encCfg)
	}
	return zapcore.NewConsoleEncoder(encCfg)
}

// encoderConfig 构建编码器配置（文件与控制台可不同）
func encoderConfig(cfg *Config, kind sinkKind) zapcore.EncoderConfig {
	if cfg.IsBrief {
		return zapcore.EncoderConfig{
			TimeKey:    "time",
			MessageKey: "msg",
			EncodeTime: encodeTimeForSink(cfg, kind),
			LineEnding: zapcore.DefaultLineEnding,
		}
	}

	levelEnc := zapcore.CapitalLevelEncoder
	if kind == sinkConsole && cfg.ShowColor {
		levelEnc = colorLevelEncoder
	}

	return zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "line",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    levelEnc,
		EncodeTime:     encodeTimeForSink(cfg, kind),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}
}

// encodeTimeForSink 按输出格式选择时间编码器
func encodeTimeForSink(cfg *Config, kind sinkKind) zapcore.TimeEncoder {
	if cfg.JsonFormat {
		return rfc3339TimeEncoder(cfg.UseUTC)
	}
	return consoleTimeEncoder
}

// consoleTimeEncoder 控制台/文本格式时间：本地时间，精确到毫秒
func consoleTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Local().Format("2006-01-02 15:04:05.000"))
}

// rfc3339TimeEncoder JSON 格式时间：RFC3339Nano，可选 UTC
func rfc3339TimeEncoder(useUTC bool) zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		if useUTC {
			t = t.UTC()
		}
		enc.AppendString(t.Format(time.RFC3339Nano))
	}
}
