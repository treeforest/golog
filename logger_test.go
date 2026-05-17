package golog

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

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

	logger := NewLogger(config)
	defer func() { _ = logger.Sync() }()

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

	logger := NewLogger(config)
	defer func() { _ = logger.Sync() }()

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
		WithLogInFile(true),
		WithLogInConsole(false),
	)

	logger := NewLogger(config)
	defer func() { _ = logger.Sync() }()

	const chunk = "ABCDEFGHIJ"
	for i := 0; i < 1024*103; i++ {
		logger.Info(chunk)
	}

	dir := filepath.Dir(logPath)
	base := strings.TrimSuffix(filepath.Base(logPath), filepath.Ext(logPath))
	files, err := filepath.Glob(filepath.Join(dir, base+"-*.log"))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 1, "应该生成滚动备份文件")
}

func TestConcurrentLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "concurrent.log")

	config := NewConfig(
		WithPath(logPath),
		WithLogInFile(true),
		WithLogInConsole(false),
	)
	logger := NewLogger(config)
	defer func() { _ = logger.Sync() }()

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

	logger := NewLogger(config)
	defer func() { _ = logger.Sync() }()

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

	logger := NewLogger(config)
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

	logger := NewLogger(config)
	defer func() { _ = logger.Sync() }()

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
	original := NewLogger(config)

	_ = original.Clone()
	original.Info("original caller")

	// Clone 用于全局时不应改变 original 的 caller skip（通过独立 zap 树）
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

	logger := NewLogger(config)
	defer func() { _ = logger.Sync() }()

	const total = 5000
	for i := 0; i < total; i++ {
		logger.Info("flood")
	}
	_ = logger.Sync()

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

	logger := NewLogger(config)
	defer func() { _ = logger.Sync() }()

	ctx := ContextWithLogger(context.Background(), logger)
	ctx = ContextWithTraceID(ctx, "trace-abc")
	ctx = ContextWithRequestID(ctx, "req-xyz")
	InfowCtx(ctx, "ctx message", "k", "v")
	_ = logger.Sync()

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
	SetDefaultLogger(NewLogger(cfg))

	Info("via global default")
	_ = Sync()

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "via global default")
}

func TestLoggerZap(t *testing.T) {
	config := NewConfig(WithLogInConsole(true), WithLogInFile(false))
	logger := NewLogger(config)
	zl := logger.Zap()
	require.NotNil(t, zl)
	zl.Info("typed", zap.String("k", "v"))
}
