package golog

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// rotatingWriter 封装 lumberjack，并支持按时间间隔定时 Rotate
type rotatingWriter struct {
	*lumberjack.Logger
	stopCh chan struct{} // 用于停止定时轮转 goroutine
	once   sync.Once
}

// newRotatingWriter 创建带轮转能力的文件写入器
func newRotatingWriter(cfg *Config) (*rotatingWriter, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.Path), 0o755); err != nil {
		return nil, err
	}

	maxSize := int(cfg.RotationSizeMB)
	if maxSize <= 0 {
		maxSize = defaultRotationSizeMB
	}

	lj := &lumberjack.Logger{
		Filename:   cfg.Path,
		MaxSize:    maxSize,
		MaxAge:     cfg.MaxAgeDays,
		MaxBackups: cfg.MaxBackups,
		Compress:   cfg.Compress,
		LocalTime:  true,
	}

	rw := &rotatingWriter{
		Logger: lj,
		stopCh: make(chan struct{}),
	}

	if cfg.RotationHours > 0 {
		go rw.runRotationTicker(time.Duration(cfg.RotationHours) * time.Hour)
	}

	return rw, nil
}

// runRotationTicker 按固定间隔触发日志轮转
func (rw *rotatingWriter) runRotationTicker(d time.Duration) {
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_ = rw.Rotate()
		case <-rw.stopCh:
			return
		}
	}
}

// close 停止定时轮转并关闭当前日志文件
func (rw *rotatingWriter) close() {
	rw.once.Do(func() {
		close(rw.stopCh)
		_ = rw.Logger.Close()
	})
}
