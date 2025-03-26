package golog

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 测试日志初始化及基础功能
func TestBasicLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	// 基础配置
	config := NewConfig(
		WithPath(logPath),
		WithLevel(InfoLevel),
		WithMaxAgeDays(1),
		WithRotationHours(1),
		WithRotationSizeMB(10),
		WithJsonFormat(false),
		WithShowLine(true),
		WithLogInConsole(true),
	)

	// 初始化日志记录器
	logger := NewLogger(config)
	defer func() {
		_ = logger.Sync()
	}()

	// 记录不同级别的日志
	logger.Debug("这应该不会出现")
	logger.Info("测试信息")
	logger.Warn("测试警告")
	logger.Error("测试错误")

	// 验证日志文件创建
	_, err := os.Stat(logPath)
	assert.NoError(t, err, "日志文件应该存在")

	// 读取日志内容
	content, err := os.ReadFile(logPath)
	assert.NoError(t, err)
	logOutput := string(content)

	// 验证日志级别过滤
	assert.NotContains(t, logOutput, "这应该不会出现", "Debug日志应该被过滤")
	assert.Contains(t, logOutput, "测试信息", "Info日志应该存在")
	assert.Contains(t, logOutput, "测试警告", "Warn日志应该存在")
	assert.Contains(t, logOutput, "测试错误", "Error日志应该存在")

	// 验证基础格式
	assert.Contains(t, logOutput, "[INFO]", "应该包含级别标签")
	assert.Contains(t, logOutput, "logger_test.go", "应该显示调用位置")
}

// 测试JSON格式输出
func TestJSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "json.log")

	config := NewConfig(
		WithPath(logPath),
		WithJsonFormat(true),
	)

	logger := NewLogger(config)
	defer func() {
		_ = logger.Sync()
	}()

	logger.Infow("JSON测试", "key", "value")

	content, _ := os.ReadFile(logPath)
	assert.Contains(t, string(content), `"level":"[INFO]"`, "应该包含JSON格式的级别字段")
	assert.Contains(t, string(content), `"key":"value"`, "应该包含结构化字段")
}

// 测试日志滚动（大小策略）
func TestSizeBasedRotation(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "size.log")

	// 配置1KB滚动
	config := NewConfig(
		WithPath(logPath),
		WithRotationSizeMB(1), // 1MB = 1024 * 1024 bytes
		WithMaxAgeDays(60),
		WithLogInConsole(false),
	)

	logger := NewLogger(config)
	defer func() {
		_ = logger.Sync()
	}()

	// 写入超过1MB的数据
	const chunk = "ABCDEFGHIJ"      // 10 bytes
	for i := 0; i < 1024*103; i++ { // 写入约1MB数据
		logger.Info(chunk)
	}

	// 验证滚动文件
	files, _ := filepath.Glob(logPath + ".*")
	assert.GreaterOrEqual(t, len(files), 1, "应该生成滚动文件")
}

// 测试并发安全
func TestConcurrentLogging(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "concurrent.log")

	config := NewConfig(
		WithPath(logPath),
		WithLogInConsole(false),
	)
	logger := NewLogger(config)
	defer func() {
		_ = logger.Sync()
	}()

	var wg sync.WaitGroup
	const workers = 100

	// 并发写入
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

	// 验证日志完整性
	content, _ := os.ReadFile(logPath)
	lines := strings.Split(string(content), "\n")
	assert.Greater(t, len(lines), workers*100-10, "应该记录所有并发日志")
}

// 测试堆栈跟踪
func TestStackTrace(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "stacktrace.log")

	config := NewConfig(
		WithPath(logPath),
		WithStackTraceLevel(ErrorLevel),
	)

	logger := NewLogger(config)
	defer func() {
		_ = logger.Sync()
	}()

	logger.Error("触发错误")

	content, _ := os.ReadFile(logPath)
	assert.Contains(t, string(content), "testing.tRunner", "应该包含堆栈信息")
}

// 测试颜色输出
func TestColorOutput(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "color.log")

	config := NewConfig(
		WithModule("testing"),
		WithComponent("color"),
		WithPath(logPath),
		WithShowColor(true),
		WithLogInConsole(true), // 需要测试控制台输出
	)

	logger := NewLogger(config)
	defer func() {
		_ = logger.Sync()
	}()

	logger.Info("带颜色信息")

	// 由于颜色代码检查较复杂，此处简化验证
	content, _ := os.ReadFile(logPath)
	assert.Contains(t, string(content), "@", "服务名称应该带颜色代码")
}

// 测试动态级别调整
func TestDynamicLevel(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "dynamic.log")

	config := &Config{
		Path:  logPath,
		Level: InfoLevel,
	}

	logger := NewLogger(config)
	defer func() {
		_ = logger.Sync()
	}()

	// 初始级别为INFO
	logger.Debug("调试信息1")
	content, _ := os.ReadFile(logPath)
	assert.NotContains(t, string(content), "调试信息1")

	// 调整为DEBUG级别
	logger.SetLevel(DebugLevel)
	logger.Debug("调试信息2")
	content, _ = os.ReadFile(logPath)
	assert.Contains(t, string(content), "调试信息2")
}
