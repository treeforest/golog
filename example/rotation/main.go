// 示例：工业级日志轮转、归档压缩与备份生命周期管理（基于 lumberjack.v2）。
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/treeforest/golog/v2"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("运行失败: %v", err)
	}
}

func run() error {
	dir := "./logs/rotation_demo"
	_ = os.RemoveAll(dir) // 演示前重置目录
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	logPath := filepath.Join(dir, "app.log")

	fmt.Println("=== 1. 初始化标准文件轮转配置 ===")
	cfg := golog.NewConfig(
		golog.WithModule("storage"),
		golog.WithPath(logPath),
		golog.WithLogInFile(true),
		golog.WithLogInConsole(false),
		golog.WithRotationSizeMB(1), // 单文件超过 1MB 自动切分滚动
		golog.WithMaxBackups(5),     // 最多保留 5 个备份文件
		golog.WithMaxAgeDays(7),     // 最长保留 7 天
		golog.WithCompress(true),    // 对滚动后的旧日志进行 gzip 压缩
	)
	golog.SetDefaultLogger(golog.MustNewLogger(cfg))

	fmt.Println("=== 2. 模拟高频业务数据写入，触发文件大小轮转 ===")
	// 单块 payload 64KB，写入多次迅速触发跨文件轮转
	payload := strings.Repeat("0123456789ABCDEF", 4096) // 64KB
	for i := range 40 {
		golog.Infof("批量数据存档块 [%02d] 数据载荷: %s", i, payload)
	}

	// 强制刷盘并安全关闭
	if err := golog.Close(); err != nil {
		return fmt.Errorf("关闭 Logger 失败: %w", err)
	}

	fmt.Println("\n=== 3. 检查生成的活动日志与滚动备份文件 ===")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	type fileItem struct {
		name string
		size int64
	}
	var files []fileItem
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fileItem{name: e.Name(), size: info.Size()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })

	fmt.Printf("文件目录: %s\n", dir)
	fmt.Printf("%-45s  %-12s  %s\n", "文件名", "大小(Bytes)", "说明")
	fmt.Println(strings.Repeat("-", 75))
	for _, f := range files {
		desc := "归档备份文件 (lumberjack ISO 时间戳)"
		if f.name == "app.log" {
			desc = "当前活跃日志文件 (Active)"
		} else if strings.HasSuffix(f.name, ".gz") {
			desc = "已 gzip 压缩的归档日志"
		}
		fmt.Printf("%-45s  %-12d  %s\n", f.name, f.size, desc)
	}

	fmt.Println(strings.Repeat("-", 75))
	fmt.Println("说明：文件命名格式为 app-YYYY-MM-DDTHH-MM-SS.sss.log(.gz)，符合工业界标准归档规范。")
	return nil
}
