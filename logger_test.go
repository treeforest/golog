package golog

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func mustNew(t *testing.T, cfg *Config) Logger {
	t.Helper()
	l, err := NewLogger(cfg)
	require.NoError(t, err)
	return l
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
		WithRotationHours(1),
		WithRotationSizeMB(10),
		WithJsonFormat(false),
		WithShowLine(true),
	)

	logger := mustNew(t, config)
	defer func() { _ = logger.Close() }()

	logger.Debug("这应该不会出现")
	logger.Info("测试信息")
	logger.Warn("测试警告")
	logger.Error("测试错误")

	_, err := os.Stat(logPath)
	require.NoError(t, err)

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
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
		WithJsonFormat(true),
	)

	logger := mustNew(t, config)
	defer func() { _ = logger.Close() }()

	logger.Infow("JSON测试", "key", "value")

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
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
		WithRotationHours(0),
		WithLogInFile(true),
		WithLogInConsole(false),
	)

	logger := mustNew(t, config)
	defer func() { _ = logger.Close() }()

	const chunk = "ABCDEFGHIJ"
	for i := 0; i < 1024*103; i++ {
		logger.Info(chunk)
	}

	files, err := filepath.Glob(logPath + "-*")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(files), 1, "应该生成滚动备份文件")

	re := regexp.MustCompile(`-\d{14}-(size|time)(\.gz)?$`)
	matched := false
	for _, f := range files {
		if re.MatchString(filepath.Base(f)) {
			matched = true
			break
		}
	}
	assert.True(t, matched, "备份文件名应匹配 path-YYYYMMDDHHmmss-size|time 格式")
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
	defer func() { _ = logger.Close() }()

	var wg sync.WaitGroup
	const workers = 100

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				logger.Infof("Worker %d: %d", id, j)
			}
		}(i)
	}
	wg.Wait()

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	lines := strings.Split(string(content), "\n")
	assert.Greater(t, len(lines), workers*100-10)
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
	defer func() { _ = logger.Close() }()

	logger.Error("触发错误")

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
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
	defer func() { _ = logger.Close() }()
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&stderrBuf, r)
		close(done)
	}()

	logger.Info("带颜色信息")
	_ = w.Close()
	os.Stderr = origStderr
	<-done

	fileContent, err := os.ReadFile(logPath)
	require.NoError(t, err)
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
	defer func() { _ = logger.Close() }()

	logger.Debug("调试信息1")
	content := readLogFile(t, logPath)
	assert.NotContains(t, string(content), "调试信息1")

	logger.SetLevel(DebugLevel)
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

func TestCloneDoesNotMutateOriginal(t *testing.T) {
	config := NewConfig(WithLogInConsole(true), WithLogInFile(false))
	original := mustNew(t, config)
	defer func() { _ = original.Close() }()

	_ = original.Clone()
	original.Info("original caller")

	assert.NotNil(t, original.Zap())
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
	defer func() { _ = logger.Close() }()

	const total = 5000
	for i := 0; i < total; i++ {
		logger.Info("flood")
	}
	require.NoError(t, logger.Sync())

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
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
		WithJsonFormat(true),
	)

	logger := mustNew(t, config)
	defer func() { _ = logger.Close() }()

	ctx := ContextWithLogger(context.Background(), logger)
	ctx = ContextWithTraceID(ctx, "trace-abc")
	ctx = ContextWithRequestID(ctx, "req-xyz")
	InfowCtx(ctx, "ctx message", "k", "v")
	require.NoError(t, logger.Sync())

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), `"trace_id":"trace-abc"`)
	assert.Contains(t, string(content), `"request_id":"req-xyz"`)
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
	defer func() { _ = Close() }()

	Info("via global default")
	require.NoError(t, Sync())

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "via global default")
}

func TestLoggerZap(t *testing.T) {
	config := NewConfig(WithLogInConsole(true), WithLogInFile(false))
	logger := mustNew(t, config)
	defer func() { _ = logger.Close() }()
	zl := logger.Zap()
	require.NotNil(t, zl)
	zl.Info("typed", zap.String("k", "v"))
}

func TestBackupFilenameFormat(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "app.log")

	cfg := NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
		WithRotationHours(0),
		WithRotationSizeMB(100),
	)
	logger := mustNew(t, cfg)
	defer func() { _ = logger.Close() }()

	logger.Info("before rotate")
	cl := logger.(*coreLogger)
	require.NotNil(t, cl.rotWriter)
	require.NoError(t, cl.rotWriter.Rotate())

	files, err := filepath.Glob(logPath + "-*")
	require.NoError(t, err)
	require.Len(t, files, 1)

	base := filepath.Base(files[0])
	re := regexp.MustCompile(`^app\.log-(\d{14})-(size|time)$`)
	m := re.FindStringSubmatch(base)
	require.NotNil(t, m, "备份名应为 app.log-YYYYMMDDHHmmss-size|time，实际: %s", base)

	ts, err := time.ParseInLocation(backupTimeFormat, m[1], time.Local)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), ts, 5*time.Second)
}

func TestSharedRotatingWriter(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "shared.log")

	cfg := NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
		WithRotationHours(0),
	)
	l1 := mustNew(t, cfg)
	l2 := mustNew(t, cfg)
	defer func() {
		_ = l1.Close()
		_ = l2.Close()
	}()

	rw1 := l1.(*coreLogger).rotWriter
	rw2 := l2.(*coreLogger).rotWriter
	require.NotNil(t, rw1)
	assert.Same(t, rw1, rw2, "同路径应复用同一 rotatingWriter")
	assert.Equal(t, int32(2), rw1.refs.Load())

	var wg sync.WaitGroup
	const n = 200
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			l1.Infof("a-%d", i)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			l2.Infof("b-%d", i)
		}
	}()
	wg.Wait()

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "a-0")
	assert.Contains(t, string(content), "b-0")
	assert.GreaterOrEqual(t, strings.Count(string(content), "\n"), n*2)
}

func TestClonePreservesRotWriter(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "clone.log")

	cfg := NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
		WithRotationHours(0),
	)
	orig := mustNew(t, cfg)
	cloned := orig.Clone()

	assert.Same(t, orig.(*coreLogger).rotWriter, cloned.(*coreLogger).rotWriter)
	assert.False(t, cloned.(*coreLogger).ownsWriter)

	SetDefaultLogger(orig)
	Info("via default after clone")
	require.NoError(t, Sync())

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "via default after clone")

	abs, err := filepath.Abs(logPath)
	require.NoError(t, err)
	_, stillRegistered := writerRegistry.Load(abs)
	assert.True(t, stillRegistered, "Sync 不应移除 registry")

	require.NoError(t, Close())
	_, stillRegistered = writerRegistry.Load(abs)
	assert.False(t, stillRegistered, "Close 后应从 registry 移除")
}

func TestSyncDoesNotCloseWriter(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "sync.log")

	logger := mustNew(t, NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
		WithRotationHours(0),
	))
	defer func() { _ = logger.Close() }()

	logger.Info("before sync")
	require.NoError(t, logger.Sync())
	logger.Info("after sync")
	require.NoError(t, logger.Sync())

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "before sync")
	assert.Contains(t, string(content), "after sync")
}

func TestCloseReleasesWriter(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "close.log")

	logger := mustNew(t, NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
		WithRotationHours(0),
	))
	logger.Info("closing")
	require.NoError(t, logger.Close())
	require.NoError(t, logger.Close()) // idempotent

	abs, err := filepath.Abs(logPath)
	require.NoError(t, err)
	_, ok := writerRegistry.Load(abs)
	assert.False(t, ok)
}

func TestSetDefaultLoggerClosesPrevious(t *testing.T) {
	tmpDir := t.TempDir()
	path1 := filepath.Join(tmpDir, "a.log")
	path2 := filepath.Join(tmpDir, "b.log")

	SetDefaultLogger(mustNew(t, NewConfig(
		WithPath(path1),
		WithLogInFile(true),
		WithLogInConsole(false),
		WithRotationHours(0),
	)))
	Info("on a")

	abs1, err := filepath.Abs(path1)
	require.NoError(t, err)

	SetDefaultLogger(mustNew(t, NewConfig(
		WithPath(path2),
		WithLogInFile(true),
		WithLogInConsole(false),
		WithRotationHours(0),
	)))
	defer func() { _ = Close() }()

	_, ok := writerRegistry.Load(abs1)
	assert.False(t, ok, "旧路径 writer 应在替换时 Close 释放")

	Info("on b")
	require.NoError(t, Sync())
	content, err := os.ReadFile(path2)
	require.NoError(t, err)
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
		WithRotationHours(0),
	))
	l2 := mustNew(t, NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
		WithRotationSizeMB(50), // different, ignored
		WithRotationHours(0),
	))
	defer func() {
		_ = l1.Close()
		_ = l2.Close()
	}()

	assert.Same(t, l1.(*coreLogger).rotWriter, l2.(*coreLogger).rotWriter)
}
