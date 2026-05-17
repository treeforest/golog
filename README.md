# golog

`golog` 是一个基于 [Uber Zap](https://github.com/uber-go/zap) 的增强型日志库，旨在简化日志输出的实现和使用。

## 核心特性

- **智能轮转策略**：基于 [lumberjack](https://github.com/natefinch/lumberjack) 按大小轮转，并支持定时 `Rotate()`（`RotationHours`）。
- **多格式输出**：JSON 与文本格式；文件与控制台可分离编码（控制台可彩色级别，文件保持纯文本）。
- **分级管理**：DEBUG / INFO / WARN / ERROR / FATAL。
- **动态调整**：运行时修改日志级别。
- **多端输出**：文件、控制台及自定义 `io.Writer`。
- **采样**：高流量场景可限制磁盘写入。
- **Context**：`trace_id` / `request_id` 透传。

## 安装

```bash
go get github.com/treeforest/golog/v2@v2.0.0
```

## 快速开始

### 基础使用

全局 Logger 在**首次调用** `golog.Info` 等时懒加载（默认仅控制台、Info 级别）。生产环境建议在 `main` 中显式配置：

```go
import "github.com/treeforest/golog/v2"

func main() {
    cfg := golog.NewConfig(
        golog.WithPath("/var/log/myapp/app.log"),
        golog.WithLogInFile(true),
        golog.WithLevel(golog.InfoLevel),
    )
    golog.SetDefaultLogger(golog.NewLogger(cfg))
    defer func() { _ = golog.Sync() }()

    golog.Info("hello")
}
```

### Context 与 trace_id

```go
ctx := golog.ContextWithTraceID(context.Background(), "trace-abc")
ctx = golog.ContextWithRequestID(ctx, "req-123")
golog.InfowCtx(ctx, "handled", "status", 200)
```

### 强类型日志（Zap）

```go
logger := golog.NewLogger(cfg)
logger.Zap().Info("event", zap.String("user", "alice"))
```

## 日志配置参数

| 参数名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `Module` | `string` | `""` | 模块名称 |
| `Component` | `string` | `""` | 组件名称（日志中显示为 `@name`） |
| `Path` | `string` | `"./logs/app.log"` | 日志文件路径 |
| `Level` | `golog.Level` | `InfoLevel` | 日志级别 |
| `MaxAgeDays` | `int` | `30` | 备份最长保留天数 |
| `MaxBackups` | `int` | `0` | 最多保留备份个数（0=仅按 MaxAge） |
| `RotationHours` | `int` | `24` | 定时轮转间隔（小时），调用 lumberjack `Rotate()` |
| `RotationSizeMB` | `int64` | `100` | 按大小轮转阈值（MB） |
| `Compress` | `bool` | `false` | 是否 gzip 压缩旧日志 |
| `JsonFormat` | `bool` | `false` | JSON 输出 |
| `UseUTC` | `bool` | `true` | JSON 时间使用 UTC RFC3339Nano |
| `ShowLine` | `bool` | `true` | 显示调用文件与行号 |
| `LogInFile` | `bool` | `false` | 写入文件 |
| `LogInConsole` | `bool` | `true` | 写入 stderr |
| `ShowColor` | `bool` | `false` | 控制台级别着色（不影响文件） |
| `IsBrief` | `bool` | `false` | 简洁模式 |
| `StackTraceLevel` | `golog.Level` | `ErrorLevel` | 堆栈记录级别 |
| `Sampling` | `SamplingConfig` | 关闭 | 采样配置 |

### 采样配置

```go
golog.WithSampling(golog.SamplingConfig{
    Enabled:    true,
    Initial:    100,  // 每秒前 100 条全量
    Thereafter: 100,  // 之后每 100 条记 1 条
})
```

## 从 v1 迁移

```bash
# 旧
import "github.com/treeforest/golog"

# 新（v2.0.0+）
import "github.com/treeforest/golog/v2"
```

```bash
go get github.com/treeforest/golog/v2@v2.0.0
```

## v2 破坏性变更

- 模块路径改为 `github.com/treeforest/golog/v2`。
- 移除 `file-rotatelogs`，改用 **lumberjack**；备份文件命名为 `name-2006-01-02T15-04-05.000.log`。
- JSON `level` 字段为标准大写（如 `"INFO"`），不再使用 `[INFO]`。
- 包 `init` 不再自动创建默认 Logger；首次全局调用时懒加载。
- `SetDefaultLogger` 使用 `Clone()`，**不会修改**传入的 Logger 实例。
- 彩色输出仅作用于**控制台**；文件始终为纯文本。

## 注意事项

- lumberjack 假定**单进程**写入同一日志文件；多进程写同一文件可能异常。
- 容器部署请使用**绝对路径**配置 `Path`。
- 进程退出前请调用 `golog.Sync()` 刷盘并停止轮转 ticker。

## 授权许可

本项目采用 Apache 许可证 2.0 版本，详见 [LICENSE](https://www.apache.org/licenses/LICENSE-2.0.txt)。
