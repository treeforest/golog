package golog

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

func NewLogger(logConfig *Config, writer ...io.Writer) Logger {
	if logConfig == nil {
		logConfig = defaultConfig()
	}
	sugaredLogger, aLevel := initSugarLogger(logConfig, writer...)
	return &coreLogger{
		SugaredLogger: sugaredLogger,
		atomicLevel:   aLevel,
	}
}

// initSugarLogger 初始化并返回一个日志记录器
func initSugarLogger(logConfig *Config, writer ...io.Writer) (*zap.SugaredLogger, zap.AtomicLevel) {
	logLevel := logConfig.Level.ZapLevel()

	aLevel := zap.NewAtomicLevel()
	aLevel.SetLevel(logLevel)

	sugaredLogger := newZapLogger(logConfig, aLevel, writer...).Sugar()
	return sugaredLogger, aLevel
}

// newZapLogger 创建日志记录器
func newZapLogger(logConfig *Config, level zap.AtomicLevel, writer ...io.Writer) *zap.Logger {
	// 配置多路输出
	var syncers []zapcore.WriteSyncer
	if logConfig.LogInFile {
		// 初始化日志滚动钩子
		hook, err := getHook(logConfig.Path, logConfig.MaxAgeDays, logConfig.RotationHours, logConfig.RotationSizeMB)
		if err != nil {
			log.Fatalf("new zap logger get hook failed, %s", err)
		}
		syncers = append(syncers, zapcore.AddSync(hook)) // 日志文件输出
	}
	if logConfig.LogInConsole {
		syncers = append(syncers, zapcore.AddSync(&noopSyncWriter{os.Stderr})) // 控制台输出
	}
	// 添加额外输出目标
	for _, outSyncer := range writer {
		syncers = append(syncers, zapcore.AddSync(outSyncer))
	}

	if len(syncers) == 0 {
		log.Fatal("no log output target")
	}

	// 创建多路同步器
	var syncer zapcore.WriteSyncer
	syncer = zapcore.NewMultiWriteSyncer(syncers...)

	// 构建编码器配置
	var encoderConfig zapcore.EncoderConfig
	if logConfig.IsBrief {
		encoderConfig = zapcore.EncoderConfig{
			TimeKey:    "time",
			MessageKey: "msg",
			EncodeTime: customTimeEncoder,
			LineEnding: zapcore.DefaultLineEnding,
		}
	} else {
		encoderConfig = zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "line",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    customLevelEncoder,
			EncodeTime:     customTimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
			EncodeName:     zapcore.FullNameEncoder,
		}
	}

	// 创建编码器（JSON或Console格式）
	var encoder zapcore.Encoder
	if logConfig.JsonFormat {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 创建核心日志器
	core := zapcore.NewCore(
		encoder,
		syncer,
		level,
	)

	// 构造组件名称显示格式
	componentName := fmt.Sprintf("@%s", logConfig.Component)
	if logConfig.ShowColor {
		componentName = getColorServiceName(componentName)
	}

	// 组装完整记录器名称
	var name string
	if logConfig.Component != "" {
		name = fmt.Sprintf("%s %s", logConfig.Module, componentName)
	} else {
		name = logConfig.Module
	}

	// 创建基础日志器
	logger := zap.New(core).Named(name)

	// 添加调用位置显示
	if logConfig.ShowLine {
		logger = logger.WithOptions(zap.AddCaller())
	}

	// 配置堆栈追踪级别
	if lvl := logConfig.StackTraceLevel.ZapLevel(); lvl != zapcore.InvalidLevel {
		logger = logger.WithOptions(zap.AddStacktrace(lvl))
	}

	return logger
}

// getHook 创建日志滚动处理器
func getHook(filename string, maxAgeDays, rotationHours int, rotationSizeMB int64) (io.Writer, error) {
	var opts []rotatelogs.Option
	opts = append(opts, rotatelogs.WithLinkName(filename))
	if rotationHours > 0 {
		opts = append(opts, rotatelogs.WithRotationTime(time.Hour*time.Duration(rotationHours)))
	}
	if rotationSizeMB > 0 {
		opts = append(opts, rotatelogs.WithRotationSize(rotationSizeMB*1024*1024))
	}
	if maxAgeDays > 0 {
		opts = append(opts, rotatelogs.WithMaxAge(time.Hour*24*time.Duration(maxAgeDays)))
	}

	hook, err := rotatelogs.New(filename+".%Y%m%d%H", opts...)
	if err != nil {
		return nil, err
	}
	return hook, nil
}

// customLevelEncoder 自定义日志级别显示格式
func customLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]") // 格式示例：[INFO]
}

// customTimeEncoder 自定义时间显示格式
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000")) // 精确到毫秒的时间格式
}
