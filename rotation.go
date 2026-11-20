package golog

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"gopkg.in/natefinch/lumberjack.v2"
)

// 全局文件轮转写入器注册表及其保护互斥锁。
// 作用：保证同一绝对路径的文件在整个进程内始终复用同一个底层 *lumberjack.Logger 实例，
// 避免多个 Goroutine 或多个 Logger 实例并发向同一文件写入时发生文件句柄冲突或轮转错乱。
var (
	writerRegistryMu sync.Mutex
	writerRegistry   = make(map[string]*rotatingWriter)
)

// rotatingWriter 是对 lumberjack.Logger 的封装，增加了进程内单例管理与基于引用计数的安全生命周期控制。
type rotatingWriter struct {
	*lumberjack.Logger
	path       string       // 日志文件规范化绝对路径
	maxSizeMB  int          // 单文件轮转大小阈值（MB）
	maxAge     int          // 归档日志最长保留天数
	maxBackups int          // 最多保留备份数
	compress   bool         // 是否启用 gzip 压缩
	refs       atomic.Int32 // 活跃引用计数器（当归零时安全关闭底层句柄并从注册表中剔除）
	closed     bool         // 内部状态标记，防止重复关闭
}

// getRotatingWriter 按绝对路径获取已有的 rotatingWriter 实例或创建新实例，并原子自增一次引用计数。
// 该方法受全局互斥锁 writerRegistryMu 保护，彻底避免并发场景下的 TOCTOU 竞态漏洞。
func getRotatingWriter(cfg *Config) (*rotatingWriter, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("log path is empty")
	}
	absPath, err := filepath.Abs(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("resolve log path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o750); err != nil {
		return nil, err
	}

	maxSize := int(cfg.RotationSizeMB)
	if maxSize <= 0 {
		maxSize = defaultRotationSizeMB
	}

	writerRegistryMu.Lock()
	defer writerRegistryMu.Unlock()

	// 若已存在未关闭的实例，直接复用并增加引用计数
	if rw, exists := writerRegistry[absPath]; exists && !rw.closed {
		warnIfConfigMismatch(rw, cfg, maxSize)
		rw.acquire()
		return rw, nil
	}

	lj := &lumberjack.Logger{
		Filename:   absPath,
		MaxSize:    maxSize,
		MaxAge:     cfg.MaxAgeDays,
		MaxBackups: cfg.MaxBackups,
		Compress:   cfg.Compress,
		LocalTime:  true,
	}

	rw := &rotatingWriter{
		Logger:     lj,
		path:       absPath,
		maxSizeMB:  maxSize,
		maxAge:     cfg.MaxAgeDays,
		maxBackups: cfg.MaxBackups,
		compress:   cfg.Compress,
	}
	rw.acquire()
	writerRegistry[absPath] = rw
	return rw, nil
}

// warnIfConfigMismatch 当同一路径复用已有实例且传入不同轮转配置时，输出警告提示（以首发配置为准）。
func warnIfConfigMismatch(rw *rotatingWriter, cfg *Config, maxSize int) {
	if rw.maxSizeMB != maxSize ||
		rw.maxAge != cfg.MaxAgeDays ||
		rw.maxBackups != cfg.MaxBackups ||
		rw.compress != cfg.Compress {
		log.Printf("golog: reusing rotating writer for %s; new config ignored (first config wins)", rw.path)
	}
}

// Sync 实现 zapcore.WriteSyncer 接口。lumberjack 内部由 OS 负责页面缓存刷盘，此处提供空实现以满足接口。
func (rw *rotatingWriter) Sync() error {
	return nil
}

// isWriterRegistered 检查指定路径的写入器是否仍处于注册且未关闭状态（主要用于测试断言）。
func isWriterRegistered(absPath string) bool {
	writerRegistryMu.Lock()
	defer writerRegistryMu.Unlock()
	rw, exists := writerRegistry[absPath]
	return exists && !rw.closed
}

// acquire 增加一次活跃引用计数。
func (rw *rotatingWriter) acquire() {
	rw.refs.Add(1)
}

// release 递减一次活跃引用计数。
// 在全局互斥锁保护下检查计数，当计数归零时立即从注册表中删除并关闭底层文件句柄，杜绝并发获取到关闭实例。
func (rw *rotatingWriter) release() {
	writerRegistryMu.Lock()
	defer writerRegistryMu.Unlock()

	if rw.refs.Add(-1) > 0 {
		return
	}
	if rw.closed {
		return
	}
	rw.closed = true
	delete(writerRegistry, rw.path)
	if err := rw.Close(); err != nil {
		log.Printf("golog: close rotating writer %s: %v", rw.path, err)
	}
}
