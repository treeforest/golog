# golog

[![Go Reference](https://pkg.go.dev/badge/github.com/treeforest/golog/v2.svg)](https://pkg.go.dev/github.com/treeforest/golog/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/treeforest/golog/v2)](https://goreportcard.com/report/github.com/treeforest/golog/v2)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

`golog` 是一个基于 [Uber Zap](https://github.com/uber-go/zap) 的高性能生产级结构化日志库，兼顾极致性能与易用性。

## 核心特性

- **标准轮转策略**：基于业界标准 [lumberjack.v2](https://gopkg.in/natefinch/lumberjack.v2)；同路径进程内单例 + 并发安全引用计数生命周期管理。
- **备份命名**：标准时间戳命名 `app-YYYY-MM-DDTHH-MM-SS.sss.log`（本地时区）。
- **多格式输出**：JSON 与纯文本控制台；文件与终端分离编码（终端可彩色级别，文件保持纯文本）。
- **不可变派生**：提供 `With`、`WithContext`、`WithCallerSkip` 不可变派生能力，无状态污染。
- **高并发零锁**：全局默认 Logger 使用 `atomic.Pointer` 无锁原子读取，消除高并发场景下的读写锁竞争。
- **精确行号定位**：彻底修复调用栈跳跃，无论是全局方法还是 Context 派生实例均精准定位到实际代码行。
- **Context 透传**：标准 `trace_id` / `request_id` 链路追踪透传。
- **动态分级与采样**：运行时动态调整日志级别；高流量场景支持基于时间窗口的请求采样防打爆。

## 安装

```bash
go get github.com/treeforest/golog/v2@v2.0.0
```

## 快速开始

### 基础使用

全局 Logger 在**首次调用** `golog.Info` 等方法时懒加载（默认仅控制台、Info 级别）。生产环境推荐在 `main` 函数显式初始化：

```go
package main

import (
    "github.com/treeforest/golog/v2"
)

func main() {
    cfg := golog.NewConfig(
        golog.WithPath("/var/log/myapp/app.log"),
        golog.WithLogInFile(true),
        golog.WithLevel(golog.InfoLevel),
    )
    golog.SetDefaultLogger(golog.MustNewLogger(cfg))
    defer func() { _ = golog.Close() }() // 退出时刷盘并安全释放文件写入器

    golog.Info("hello world")
    _ = golog.Sync() // 运行中可多次刷盘，不关闭句柄
}
```

### Context 链路追踪与不可变派生（Sub-Logger）

在微服务场景下，推荐在请求入口绑定 Context，并在下游业务中复用；也可以基于 `With` 派生模块专属 Logger：

```go
// 1. 模拟 Web 入口单行注入全链路追踪三元组与业务标签
ctx := golog.ContextWithTrace(context.Background(), "trace-app-881920", "req-mob-001", "span-001")
ctx = golog.ContextWithFields(ctx, "tenant_id", "VIP-China")

// 2. 从 Context 获取绑定了 trace 信息的不可变 Logger（精准记录行号）
logger := golog.LoggerFromContext(ctx)
logger.Infow("网关收到外部请求", "path", "/api/checkout", "user_id", 10001)

// 3. 基于 With() 派生子模块专用 Logger（不可变，不污染父级 Logger）
orderLogger := logger.With("service", "order", "env", "prod")
orderLogger.Infow("订单创建中", "order_id", "ORD-20261120-001")

// 4. 也支持使用包级全局便捷函数
golog.InfowCtx(ctx, "业务处理完毕", "status", "success")
```

### Sync 与 Close 生命周期

| API | 行为 | 适用场景 |
| :--- | :--- | :--- |
| `Sync()` | 刷盘，**不**关闭文件底层写入器 | 业务运行中定期或关键事务提交后执行 |
| `Close()` | 刷盘并根据引用计数释放底层写入器 | 进程退出前（如 `defer golog.Close()`）执行 |

## 日志配置参数

| 参数名 | 类型 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `Module` | `string` | `""` | 模块名称 |
| `Component` | `string` | `""` | 组件名称（日志中显示为 `@name`） |
| `Path` | `string` | `"./logs/app.log"` | 日志文件路径（`LogInFile` 时必填） |
| `Level` | `golog.Level` | `InfoLevel` | 日志级别 (Debug / Info / Warn / Error / Fatal) |
| `MaxAgeDays` | `int` | `30` | 备份文件最长保留天数 |
| `MaxBackups` | `int` | `0` | 最多保留备份个数（0 表示仅按保留天数清理） |
| `RotationSizeMB` | `int64` | `100` | 单个日志文件大小滚动阈值（MB） |
| `Compress` | `bool` | `false` | 是否 gzip 压缩滚动备份文件 |
| `JSONFormat` | `bool` | `false` | 是否使用 JSON 格式输出 |
| `UseUTC` | `bool` | `true` | JSON 时间戳是否使用 UTC (RFC3339Nano) |
| `ShowLine` | `bool` | `true` | 是否打印调用文件和精准行号 |
| `LogInFile` | `bool` | `false` | 是否写入文件 |
| `LogInConsole` | `bool` | `true` | 是否写入 stderr 控制台 |
| `ShowColor` | `bool` | `false` | 控制台输出是否使用 ANSI 彩色（不污染文件） |
| `Brief` | `bool` | `false` | 简洁模式（仅保留时间和消息正文） |
| `StackTraceLevel` | `golog.Level` | `ErrorLevel` | 记录堆栈跟踪的最低日志级别 |
| `Sampling` | `SamplingConfig` | 关闭 | 日志采样频率控制 |

## 运行示例

仓库 `example/` 目录提供了贴近实际工业级生产场景的可直接运行示例：

```bash
go run ./example/basic        # 1. 基础使用：控制台输出、ANSI 彩色渲染与运行时动态升降级
go run ./example/jsonformat   # 2. 云原生日志：标准 JSON 格式输出，与 ELK / Loki 契约无缝对齐
go run ./example/custom       # 3. 生产级配置：控制台彩色 + 文件纯文本双分流，基于 With() 的子模块派生
go run ./example/tracing      # 4. 链路追踪：模拟 HTTP 请求网关->业务服务->数据层的全链路 Context 自动透传
go run ./example/rotation     # 5. 存储与归档：大文件按大小自动切割、lumberjack ISO 时间戳归档与 gzip 压缩
```

## 授权许可

本项目采用 Apache 许可证 2.0 版本，详见 [LICENSE](LICENSE)。
