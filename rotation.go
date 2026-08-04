package golog

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/DeRuina/timberjack"
)

const backupTimeFormat = "20060102150405" // YYYYMMDDHHmmss（本地时区）

// writerRegistry 按绝对路径复用 rotatingWriter，保证同文件仅一个 timberjack 实例
var writerRegistry sync.Map // key: absPath -> *rotatingWriter

// rotatingWriter 封装 timberjack；进程内同路径单例，经引用计数安全关闭
type rotatingWriter struct {
	*timberjack.Logger
	path           string
	maxSizeMB      int
	maxAge         int
	maxBackups     int
	compress       bool
	rotationHours  int
	refs           atomic.Int32
	closeOnce      sync.Once
}

// getRotatingWriter 按绝对路径获取或创建 rotatingWriter，并 acquire 一次引用
func getRotatingWriter(cfg *Config) (*rotatingWriter, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("log path is empty")
	}
	absPath, err := filepath.Abs(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("resolve log path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return nil, err
	}

	maxSize := int(cfg.RotationSizeMB)
	if maxSize <= 0 {
		maxSize = defaultRotationSizeMB
	}

	if existing, ok := writerRegistry.Load(absPath); ok {
		rw := existing.(*rotatingWriter)
		warnIfConfigMismatch(rw, cfg, maxSize)
		rw.acquire()
		return rw, nil
	}

	compression := "none"
	if cfg.Compress {
		compression = "gzip"
	}

	tj := &timberjack.Logger{
		Filename:           absPath,
		MaxSize:            maxSize,
		MaxAge:             cfg.MaxAgeDays,
		MaxBackups:         cfg.MaxBackups,
		Compression:        compression,
		LocalTime:          true,
		BackupTimeFormat:   backupTimeFormat,
		AppendTimeAfterExt: true,
	}
	if cfg.RotationHours > 0 {
		tj.RotationInterval = time.Duration(cfg.RotationHours) * time.Hour
	}

	rw := &rotatingWriter{
		Logger:        tj,
		path:          absPath,
		maxSizeMB:     maxSize,
		maxAge:        cfg.MaxAgeDays,
		maxBackups:    cfg.MaxBackups,
		compress:      cfg.Compress,
		rotationHours: cfg.RotationHours,
	}

	actual, loaded := writerRegistry.LoadOrStore(absPath, rw)
	if loaded {
		existing := actual.(*rotatingWriter)
		warnIfConfigMismatch(existing, cfg, maxSize)
		existing.acquire()
		return existing, nil
	}
	rw.acquire()
	return rw, nil
}

func warnIfConfigMismatch(rw *rotatingWriter, cfg *Config, maxSize int) {
	if rw.maxSizeMB != maxSize ||
		rw.maxAge != cfg.MaxAgeDays ||
		rw.maxBackups != cfg.MaxBackups ||
		rw.compress != cfg.Compress ||
		rw.rotationHours != cfg.RotationHours {
		log.Printf("golog: reusing rotating writer for %s; new config ignored (first config wins)", rw.path)
	}
}

func (rw *rotatingWriter) acquire() {
	rw.refs.Add(1)
}

// release 减少引用；归零时关闭 timberjack 并从 registry 移除
func (rw *rotatingWriter) release() {
	if rw.refs.Add(-1) > 0 {
		return
	}
	rw.closeOnce.Do(func() {
		_ = rw.Logger.Close()
		writerRegistry.Delete(rw.path)
	})
}
