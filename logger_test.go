package golog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustNew(t *testing.T, cfg *Config) Logger {
	t.Helper()
	l, err := NewLogger(cfg)
	require.NoError(t, err)
	closeLogger(t, l)
	return l
}

func closeLogger(t *testing.T, l Logger) {
	t.Helper()
	t.Cleanup(func() { require.NoError(t, l.Close()) })
}

func mustCoreLogger(t *testing.T, l Logger) *coreLogger {
	t.Helper()
	cl, ok := l.(*coreLogger)
	require.True(t, ok)
	return cl
}

func TestBasicLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	config := NewConfig(
		WithPath(logPath),
		WithLevel(InfoLevel),
		WithLogInFile(true),
		WithLogInConsole(false),
		WithMaxAgeDays(1),
		WithRotationSizeMB(10),
		WithJSONFormat(false),
		WithShowLine(true),
	)

	logger := mustNew(t, config)

	logger.Debug("这应该不会出现")
	logger.Info("测试信息")
	logger.Warn("测试警告")
	logger.Error("测试错误")

	_, err := os.Stat(logPath)
	require.NoError(t, err)

	content := readLogFile(t, logPath)
	logOutput := string(content)

	assert.NotContains(t, logOutput, "这应该不会出现")
	assert.Contains(t, logOutput, "测试信息")
	assert.Contains(t, logOutput, "测试警告")
	assert.Contains(t, logOutput, "测试错误")
	assert.Contains(t, logOutput, "INFO")
	assert.NotContains(t, logOutput, "[INFO]")
	assert.Contains(t, logOutput, "logger_test.go")
}

func TestJSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "json.log")

	config := NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
		WithJSONFormat(true),
	)

	logger := mustNew(t, config)

	logger.Infow("JSON测试", "key", "value")

	content := readLogFile(t, logPath)
	assert.Contains(t, string(content), `"level":"INFO"`)
	assert.Contains(t, string(content), `"key":"value"`)
}

func TestSizeBasedRotation(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "size.log")

	config := NewConfig(
		WithPath(logPath),
		WithRotationSizeMB(1),
		WithMaxAgeDays(60),
		WithLogInFile(true),
		WithLogInConsole(false),
	)

	logger := mustNew(t, config)

	chunk := strings.Repeat("A", 1024*64)
	for range 20 {
		logger.Info(chunk)
	}

	files, err := filepath.Glob(filepath.Join(tmpDir, "size-*.log"))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(files), 1, "应该生成滚动备份文件")

	re := regexp.MustCompile(`size-\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}\.\d{3}\.log$`)
	matched := false
	for _, f := range files {
		if re.MatchString(filepath.Base(f)) {
			matched = true
			break
		}
	}
	assert.True(t, matched, "备份文件名应匹配 lumberjack 时间戳格式")
}

func TestConcurrentLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "concurrent.log")

	config := NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
	)
	logger := mustNew(t, config)

	var wg sync.WaitGroup
	const workers = 50

	wg.Add(workers)
	for i := range workers {
		go func(id int) {
			defer wg.Done()
			for j := range 50 {
				logger.Infof("Worker %d: %d", id, j)
			}
		}(i)
	}
	wg.Wait()

	content := readLogFile(t, logPath)
	lines := strings.Split(string(content), "\n")
	assert.Greater(t, len(lines), workers*50-10)
}

func TestStackTrace(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "stacktrace.log")

	config := NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
		WithStackTraceLevel(ErrorLevel),
	)

	logger := mustNew(t, config)

	logger.Error("触发错误")

	content := readLogFile(t, logPath)
	assert.Contains(t, string(content), "testing.tRunner")
}

func TestColorOutput(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "color.log")

	var stderrBuf bytes.Buffer
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	config := NewConfig(
		WithModule("testing"),
		WithComponent("color"),
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(true),
		WithShowColor(true),
	)

	logger := mustNew(t, config)
	done := make(chan struct{})
	go func() {
		_, err := io.Copy(&stderrBuf, r)
		require.NoError(t, err)
		close(done)
	}()

	logger.Info("带颜色信息")
	require.NoError(t, w.Close())
	os.Stderr = origStderr
	<-done

	fileContent := readLogFile(t, logPath)
	assert.Contains(t, string(fileContent), "@color")
	assert.NotContains(t, string(fileContent), "\033[")

	stderrOut := stderrBuf.String()
	assert.Contains(t, stderrOut, "\033[")
	assert.Contains(t, stderrOut, "INFO")
}

func TestDynamicLevel(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "dynamic.log")

	config := NewConfig(
		WithPath(logPath),
		WithLevel(InfoLevel),
		WithLogInFile(true),
		WithLogInConsole(false),
	)

	logger := mustNew(t, config)
	assert.Equal(t, InfoLevel, logger.GetLevel())

	logger.Debug("调试信息1")
	content := readLogFile(t, logPath)
	assert.NotContains(t, string(content), "调试信息1")

	logger.SetLevel(DebugLevel)
	assert.Equal(t, DebugLevel, logger.GetLevel())
	logger.Debug("调试信息2")
	content = readLogFile(t, logPath)
	assert.Contains(t, string(content), "调试信息2")
}

func readLogFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)
	return content
}

func TestWithCallerSkipDoesNotMutateOriginal(t *testing.T) {
	config := NewConfig(WithLogInConsole(true), WithLogInFile(false))
	original := mustNew(t, config)

	derived := original.WithCallerSkip(1)
	original.Info("original caller")
	derived.Info("derived caller")

	require.NotNil(t, original)
}

func TestSampling(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "sample.log")

	config := NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
		WithLevel(InfoLevel),
		WithSampling(SamplingConfig{
			Enabled:    true,
			Initial:    10,
			Thereafter: 10,
		}),
	)

	logger := mustNew(t, config)

	const total = 5000
	for range total {
		logger.Info("flood")
	}
	require.NoError(t, logger.Sync())

	content := readLogFile(t, logPath)
	lineCount := strings.Count(string(content), "flood")
	assert.Less(t, lineCount, total/2, "采样后输出应明显少于调用次数")
}

func TestContextTraceID(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "ctx.log")

	config := NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
		WithJSONFormat(true),
	)

	logger := mustNew(t, config)

	ctx := ContextWithLogger(context.Background(), logger)
	ctx = ContextWithTrace(ctx, "trace-abc", "req-xyz", "span-001")
	ctx = ContextWithFields(ctx, "tenant_id", "VIP-China")
	InfowCtx(ctx, "ctx message", "k", "v")
	require.NoError(t, logger.Sync())

	content := readLogFile(t, logPath)
	assert.Contains(t, string(content), `"trace_id":"trace-abc"`)
	assert.Contains(t, string(content), `"request_id":"req-xyz"`)
	assert.Contains(t, string(content), `"span_id":"span-001"`)
	assert.Contains(t, string(content), `"tenant_id":"VIP-China"`)
}

func TestSetDefaultLoggerPreservesFileOutput(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "default.log")

	cfg := NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
	)
	SetDefaultLogger(mustNew(t, cfg))
	t.Cleanup(func() { require.NoError(t, Close()) })

	Info("via global default")
	require.NoError(t, Sync())

	content := readLogFile(t, logPath)
	assert.Contains(t, string(content), "via global default")
}

func TestWithAndWithContext(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "with.log")

	config := NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
		WithJSONFormat(true),
	)
	var l Logger = mustNew(t, config)

	ctx := ContextWithTrace(context.Background(), "t-123", "r-456")
	derived := l.WithContext(ctx).With("service", "order").WithCallerSkip(0)
	derived.Infow("processed", "amount", 100)
	require.NoError(t, l.Sync())

	content := readLogFile(t, logPath)
	assert.Contains(t, string(content), `"trace_id":"t-123"`)
	assert.Contains(t, string(content), `"service":"order"`)
	assert.Contains(t, string(content), `"amount":100`)
}

func TestBackupFilenameFormat(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "app.log")

	cfg := NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
		WithRotationSizeMB(100),
	)
	logger := mustNew(t, cfg)

	logger.Info("before rotate")
	cl := mustCoreLogger(t, logger)
	require.NotNil(t, cl.rotWriter)
	require.NoError(t, cl.rotWriter.Rotate())

	files, err := filepath.Glob(filepath.Join(tmpDir, "app-*.log"))
	require.NoError(t, err)
	require.Len(t, files, 1)

	base := filepath.Base(files[0])
	re := regexp.MustCompile(`^app-\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}\.\d{3}\.log$`)
	require.True(t, re.MatchString(base), "备份名应符合 lumberjack 格式，实际: %s", base)
}

func TestSharedRotatingWriter(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "shared.log")

	cfg := NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
	)
	l1 := mustNew(t, cfg)
	l2 := mustNew(t, cfg)

	rw1 := mustCoreLogger(t, l1).rotWriter
	rw2 := mustCoreLogger(t, l2).rotWriter
	require.NotNil(t, rw1)
	assert.Same(t, rw1, rw2, "同路径应复用同一 rotatingWriter")
	assert.Equal(t, int32(2), rw1.refs.Load())

	var wg sync.WaitGroup
	const n = 100
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := range n {
			l1.Infof("a-%d", i)
		}
	}()
	go func() {
		defer wg.Done()
		for i := range n {
			l2.Infof("b-%d", i)
		}
	}()
	wg.Wait()

	content := readLogFile(t, logPath)
	assert.Contains(t, string(content), "a-0")
	assert.Contains(t, string(content), "b-0")
	assert.GreaterOrEqual(t, strings.Count(string(content), "\n"), n*2)
}

func TestDerivedLoggerPreservesRotWriter(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "derived.log")

	cfg := NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
	)
	orig := mustNew(t, cfg)
	derived := orig.With("module", "sub")

	assert.Same(t, mustCoreLogger(t, orig).rotWriter, mustCoreLogger(t, derived).rotWriter)

	SetDefaultLogger(orig)
	Info("via default after derived")
	require.NoError(t, Sync())

	content := readLogFile(t, logPath)
	assert.Contains(t, string(content), "via default after derived")

	abs, err := filepath.Abs(logPath)
	require.NoError(t, err)
	assert.True(t, isWriterRegistered(abs), "Sync 不应移除 registry")

	require.NoError(t, Close())
	_ = derived.Close()
	assert.False(t, isWriterRegistered(abs), "所有引用释放后应从 registry 移除")
}

func TestSyncDoesNotCloseWriter(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "sync.log")

	logger := mustNew(t, NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
	))

	logger.Info("before sync")
	require.NoError(t, logger.Sync())
	logger.Info("after sync")
	require.NoError(t, logger.Sync())

	content := readLogFile(t, logPath)
	assert.Contains(t, string(content), "before sync")
	assert.Contains(t, string(content), "after sync")
}

func TestCloseReleasesWriter(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "close.log")

	logger, err := NewLogger(NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
	))
	require.NoError(t, err)
	logger.Info("closing")
	require.NoError(t, logger.Close())
	require.NoError(t, logger.Close()) // idempotent

	abs, err := filepath.Abs(logPath)
	require.NoError(t, err)
	assert.False(t, isWriterRegistered(abs))
}

func TestSetDefaultLoggerClosesPrevious(t *testing.T) {
	tmpDir := t.TempDir()
	path1 := filepath.Join(tmpDir, "a.log")
	path2 := filepath.Join(tmpDir, "b.log")

	SetDefaultLogger(mustNew(t, NewConfig(
		WithPath(path1),
		WithLogInFile(true),
		WithLogInConsole(false),
	)))
	Info("on a")

	abs1, err := filepath.Abs(path1)
	require.NoError(t, err)

	SetDefaultLogger(mustNew(t, NewConfig(
		WithPath(path2),
		WithLogInFile(true),
		WithLogInConsole(false),
	)))
	t.Cleanup(func() { require.NoError(t, Close()) })

	assert.False(t, isWriterRegistered(abs1), "旧路径 writer 应在替换时释放")

	Info("on b")
	require.NoError(t, Sync())
	content := readLogFile(t, path2)
	assert.Contains(t, string(content), "on b")
}

func TestEmptyPathError(t *testing.T) {
	_, err := NewLogger(NewConfig(
		WithPath(""),
		WithLogInFile(true),
		WithLogInConsole(false),
	))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestNoOutputTargetError(t *testing.T) {
	_, err := NewLogger(NewConfig(
		WithLogInFile(false),
		WithLogInConsole(false),
	))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no log output")
}

func TestConfigMismatchReusesWriter(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "mismatch.log")

	l1 := mustNew(t, NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
		WithRotationSizeMB(10),
	))
	l2 := mustNew(t, NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
		WithRotationSizeMB(50),
	))

	assert.Same(t, mustCoreLogger(t, l1).rotWriter, mustCoreLogger(t, l2).rotWriter)
}

// 补充测试：精准行号验证
func TestCallerLineAccuracy(t *testing.T) {
	var buf bytes.Buffer
	cfg := NewConfig(
		WithLogInConsole(false),
		WithLogInFile(false),
		WithShowLine(true),
	)
	l, err := NewLogger(cfg, &buf)
	require.NoError(t, err)
	SetDefaultLogger(l)
	defer Close()

	_, _, expectedLine, _ := runtime.Caller(0)
	Info("hello accuracy") // expectedLine + 1
	expectedLine++

	out := buf.String()
	expectedPattern := fmt.Sprintf("logger_test.go:%d", expectedLine)
	assert.Contains(t, out, expectedPattern, "行号必须准确匹配实际调用行")
}

// 补充测试：全局函数全覆盖
func TestGlobalAndContextFuncs(t *testing.T) {
	var buf bytes.Buffer
	cfg := NewConfig(
		WithLogInConsole(false),
		WithLogInFile(false),
		WithLevel(DebugLevel),
	)
	l := MustNewLogger(cfg, &buf)
	SetDefaultLogger(l)
	defer Close()

	Debug("d")
	Debugf("df %s", "1")
	Debugw("dw", "k", "v")
	Infof("if %s", "2")
	Infow("iw", "k", "v")
	Warn("w")
	Warnf("wf %s", "3")
	Warnw("ww", "k", "v")
	Errorf("ef %s", "4")
	Errorw("ew", "k", "v")

	SetLevel(WarnLevel)
	assert.Equal(t, WarnLevel, GetLevel())

	ctx := ContextWithTrace(context.Background(), "tr-1", "req-1")
	DebugCtx(ctx, "dctx")
	DebugfCtx(ctx, "dfctx %s", "1")
	DebugwCtx(ctx, "dwctx", "k", "v")
	InfoCtx(ctx, "ictx")
	InfofCtx(ctx, "ifctx %s", "2")
	WarnCtx(ctx, "wctx")
	WarnfCtx(ctx, "wfctx %s", "3")
	WarnwCtx(ctx, "wwctx", "k", "v")
	ErrorCtx(ctx, "ectx")
	ErrorfCtx(ctx, "efctx %s", "4")
	ErrorwCtx(ctx, "ewctx", "k", "v")

	origFatalf := logFatalf
	var fatalCount int
	logFatalf = func(format string, args ...any) {
		fatalCount++
	}
	defer func() { logFatalf = origFatalf }()

	_ = MustNewLogger(&Config{}) // 默认配置正常创建
	_, _ = NewLogger(nil)
}
