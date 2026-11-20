package golog

import (
	"fmt"
	"io"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// sinkKind 标识日志输出目标类型，用于按需选择不同的编码与着色策略。
type sinkKind int

const (
	sinkFile    sinkKind = iota // 文件输出目标（强制纯文本编码，无 ANSI 转义字符）
	sinkConsole                 // 控制台输出目标（支持对级别字段进行 ANSI 终端着色）
	sinkExtra                   // 额外自定义 io.Writer 输出目标（采用纯文本编码）
)

// noopSyncWriter 是对 zapcore.WriteSyncer 的装饰器实现。
//
// 设计背景：
// 在特定操作系统或容器化环境（如某些 Linux 变体）中，对系统标准错误输出流（os.Stderr / /dev/stderr）
// 调用系统级 Sync (fsync) 操作时，可能会返回 "sync /dev/stderr: invalid argument" 错误。
// 该装饰器覆写了 Sync 方法并直接返回 nil，避免对无须刷新或不支持刷新的流操作产生误报警报。
type noopSyncWriter struct {
	zapcore.WriteSyncer
}

// Sync 空实现，不执行任何底层系统同步。
func (n *noopSyncWriter) Sync() error {
	return nil
}

// NewLogger 根据给定的配置与可选的自定义输出流创建并返回 Logger 实例。
//
// 调用方若启用了文件持久化输出，在应用程序正常退出前应当对返回的 Logger 调用 Close() 以确保刷盘并释放资源。
func NewLogger(logConfig *Config, writer ...io.Writer) (Logger, error) {
	if logConfig == nil {
		logConfig = defaultConfig()
	}
	zapLogger, aLevel, rotWriter, err := initZapLogger(logConfig, writer...)
	if err != nil {
		return nil, err
	}
	return &coreLogger{
		SugaredLogger: zapLogger.Sugar(),
		zapLogger:     zapLogger,
		atomicLevel:   aLevel,
		rotWriter:     rotWriter,
	}, nil
}

// MustNewLogger 功能同 NewLogger，但当创建失败时会直接输出错误并终止进程（os.Exit(1)）。
func MustNewLogger(logConfig *Config, writer ...io.Writer) Logger {
	l, err := NewLogger(logConfig, writer...)
	if err != nil {
		logFatalf("golog: create logger failed: %v", err)
	}
	return l
}

// logFatalf 供内部在发生致命初始化错误时调用，测试中可被替换以验证行为。
var logFatalf = func(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// initZapLogger 构造底层 zap.Logger 实例、原子日志级别控制器及文件轮转写入器。
func initZapLogger(logConfig *Config, writer ...io.Writer) (*zap.Logger, zap.AtomicLevel, *rotatingWriter, error) {
	aLevel := zap.NewAtomicLevel()
	aLevel.SetLevel(logConfig.Level.ZapLevel())

	zapLogger, rotWriter, err := newZapLogger(logConfig, aLevel, writer...)
	if err != nil {
		return nil, aLevel, nil, err
	}
	return zapLogger, aLevel, rotWriter, nil
}

// newZapLogger 创建底层 zap.Logger，根据配置组装 Tee Core（同时分流至文件、控制台及额外 Writer）。
func newZapLogger(logConfig *Config, level zap.AtomicLevel, writer ...io.Writer) (*zap.Logger, *rotatingWriter, error) {
	var cores []zapcore.Core
	var rotWriter *rotatingWriter

	if logConfig.LogInFile {
		rw, err := getRotatingWriter(logConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("create rotating writer: %w", err)
		}
		rotWriter = rw
		// 文件目标：纯文本编码，确保文件内不含 ANSI 彩色控制字符
		cores = append(cores, buildCore(logConfig, level, rw, sinkFile))
	}

	if logConfig.LogInConsole {
		// 控制台目标：包装 noopSyncWriter，按需启用彩色级别编码
		cores = append(cores, buildCore(logConfig, level, &noopSyncWriter{WriteSyncer: zapcore.AddSync(os.Stderr)}, sinkConsole))
	}

	for _, w := range writer {
		cores = append(cores, buildCore(logConfig, level, zapcore.AddSync(w), sinkExtra))
	}

	if len(cores) == 0 {
		return nil, nil, fmt.Errorf("no log output target")
	}

	var core zapcore.Core
	if len(cores) == 1 {
		core = cores[0]
	} else {
		core = zapcore.NewTee(cores...)
	}
	// 采样包装在 Tee 核心外层，确保所有输出目标采样的记录保持一致
	core = wrapSampling(logConfig, core)

	name := loggerName(logConfig)
	logger := zap.New(core).Named(name)

	if logConfig.ShowLine {
		logger = logger.WithOptions(zap.AddCaller())
	}
	if lvl := logConfig.StackTraceLevel.ZapLevel(); lvl != zapcore.InvalidLevel {
		logger = logger.WithOptions(zap.AddStacktrace(lvl))
	}

	return logger, rotWriter, nil
}

// loggerName 拼接纯文本 Logger 模块与组件名称（格式为 "Module @Component" 或 "Module"）。
func loggerName(cfg *Config) string {
	if cfg.Component != "" {
		return fmt.Sprintf("%s @%s", cfg.Module, cfg.Component)
	}
	return cfg.Module
}

// buildCore 根据配置、级别、写入器和目标类型构建独立的 zapcore.Core。
func buildCore(cfg *Config, level zap.AtomicLevel, ws zapcore.WriteSyncer, kind sinkKind) zapcore.Core {
	enc := newEncoder(cfg, kind)
	return zapcore.NewCore(enc, ws, level)
}

// wrapSampling 按配置将 zapcore.Core 包装为采样限流 Core。
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

// newEncoder 根据输出格式（JSON 或 控制台文本）和目标类型创建对应的编码器。
func newEncoder(cfg *Config, kind sinkKind) zapcore.Encoder {
	encCfg := encoderConfig(cfg, kind)
	if cfg.JSONFormat {
		return zapcore.NewJSONEncoder(encCfg)
	}
	return zapcore.NewConsoleEncoder(encCfg)
}

// encoderConfig 根据配置和输出目标类型（终端/文件）构建编码器具体参数。
func encoderConfig(cfg *Config, kind sinkKind) zapcore.EncoderConfig {
	if cfg.Brief {
		return zapcore.EncoderConfig{
			TimeKey:    "time",
			MessageKey: "msg",
			EncodeTime: encodeTimeForSink(cfg),
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
		EncodeTime:     encodeTimeForSink(cfg),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}
}

// encodeTimeForSink 根据输出格式选择对应的时间格式化器。
func encodeTimeForSink(cfg *Config) zapcore.TimeEncoder {
	if cfg.JSONFormat {
		return rfc3339TimeEncoder(cfg.UseUTC)
	}
	return consoleTimeEncoder
}

// consoleTimeEncoder 控制台与纯文本格式时间格式化：本地时区，精确至毫秒。
func consoleTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Local().Format("2006-01-02 15:04:05.000"))
}

// rfc3339TimeEncoder JSON 格式时间格式化：RFC3339Nano 格式，可选择是否转换为 UTC。
func rfc3339TimeEncoder(useUTC bool) zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		if useUTC {
			t = t.UTC()
		}
		enc.AppendString(t.Format(time.RFC3339Nano))
	}
}
