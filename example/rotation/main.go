// 压测按大小轮转：并发写入约数 MB，验证生成多个备份文件。
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/treeforest/golog/v2"
)

// 预期：
//   - 活动文件：rotation.log
//   - 备份：rotation.log-YYYYMMDDHHmmss-size（本地时区）
func main() {
	os.Exit(run())
}

func run() int {
	dir := "./logs/rotation"
	if err := os.RemoveAll(dir); err != nil {
		log.Printf("remove dir: %v", err)
		return 1
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		log.Printf("mkdir: %v", err)
		return 1
	}
	logPath := filepath.Join(dir, "rotation.log")

	cfg := golog.NewConfig(
		golog.WithModule("bench"),
		golog.WithPath(logPath),
		golog.WithLevel(golog.InfoLevel),
		golog.WithLogInFile(true),
		golog.WithLogInConsole(false),
		golog.WithShowLine(false),
		golog.WithRotationHours(0),
		golog.WithRotationSizeMB(1),
		golog.WithMaxBackups(20),
		golog.WithMaxAgeDays(1),
	)
	golog.SetDefaultLogger(golog.MustNewLogger(cfg))
	defer func() {
		if err := golog.Close(); err != nil {
			log.Printf("close default logger: %v", err)
		}
	}()

	const (
		workers   = 8
		perWorker = 80000
		payload   = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	)

	start := time.Now()
	var written atomic.Int64
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := range workers {
		go func(id int) {
			defer wg.Done()
			for i := range perWorker {
				golog.Infof("w=%d i=%d pad=%s", id, i, payload)
				written.Add(1)
			}
		}(w)
	}
	wg.Wait()
	if err := golog.Sync(); err != nil {
		log.Printf("sync: %v", err)
		return 1
	}
	elapsed := time.Since(start)

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("readdir: %v", err)
		return 1
	}

	type fileInfo struct {
		name string
		size int64
	}
	var files []fileInfo
	var total int64
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{name: e.Name(), size: info.Size()})
		total += info.Size()
	}
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })

	backupRE := regexp.MustCompile(`^rotation\.log-\d{14}-size$`)
	var backups int
	var hasActive bool
	fmt.Printf("dir=%s\n", dir)
	fmt.Printf("lines=%d elapsed=%s throughput=%.0f lines/s\n",
		written.Load(), elapsed.Round(time.Millisecond),
		float64(written.Load())/elapsed.Seconds())
	fmt.Println("--- files ---")
	for _, f := range files {
		kind := "other"
		switch {
		case f.name == "rotation.log":
			kind = "active"
			hasActive = true
		case backupRE.MatchString(f.name):
			kind = "backup"
			backups++
		}
		fmt.Printf("%8d  %-40s  [%s]\n", f.size, f.name, kind)
	}
	fmt.Printf("--- summary: files=%d backups=%d total_bytes=%d ---\n",
		len(files), backups, total)

	if !hasActive {
		fmt.Println("FAIL: missing active rotation.log")
		return 1
	}
	if backups < 2 {
		fmt.Printf("FAIL: expected >=2 size backups, got %d\n", backups)
		return 1
	}
	fmt.Println("OK: rotation naming and multi-file output look correct")
	return 0
}
