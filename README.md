# golog

`golog` 是一个基于 [Uber Zap](https://github.com/uber-go/zap) 的增强型日志库，旨在简化日志输出的实现和使用。

## 核心特性

- **智能轮转策略**：基于 [timberjack](https://github.com/DeRuina/timberjack) 按大小与时间间隔轮转；同路径进程内单例 + 引用计数。
- **备份命名**：活动日志 `app.log`，备份为 `app.log-YYYYMMDDHHmmss-size` / `app.log-YYYYMMDDHHmmss-time`（服务器本地时区）。
- **多格式输出**：JSON 与文本格式；文件与控制台可分离编码（控制台可彩色级别，文件保持纯文本）。
- **分级管理**：DEBUG / INFO / WARN / ERROR / FATAL。
- **动态调整**：运行时修改日志级别。
- **多端输出**：文件、控制台及自定义 `io.Writer`。
- **采样**：高流量场景可限制磁盘写入（Tee 外统一采样）。
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
    golog.SetDefaultLogger(golog.MustNewLogger(cfg))
    defer func() { _ = golog.Close() }() // 退出时刷盘并关闭文件写入器

    golog.Info("hello")
    _ = golog.Sync() // 仅刷盘，不关闭；运行中可多次调用
}
```

`NewLogger` 返回 `(Logger, error)`；失败需处理时可用它，简单场景用 `MustNewLogger`。

### Sync 与 Close

| API | 行为 |
|-----|------|
| `Sync()` | 刷盘，**不**关闭文件；运行中可安全调用 |
| `Close()` | 刷盘并释放文件写入器所有权；进程退出时调用一次 |

`SetDefaultLogger` 会接管传入 Logger 的文件写入器所有权，并 `Close` 旧的全局 default。

### Context 与 trace_id

推荐在请求入口绑定一次，热路径复用，避免每条 `*Ctx` 都新建派生 Logger：

```go
ctx := golog.ContextWithTraceID(context.Background(), "trace-abc")
ctx = golog.ContextWithRequestID(ctx, "req-123")
logger := golog.LoggerFromContext(ctx) // 或 root.WithContext(ctx)
logger.Infow("handled", "status", 200)
```

### 强类型日志（Zap）

```go
logger := golog.MustNewLogger(cfg)
logger.Zap().Info("event", zap.String("user", "alice"))
```

## 日志配置参数

| 参数名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `Module` | `string` | `""` | 模块名称 |
| `Component` | `string` | `""` | 组件名称（日志中显示为 `@name`） |
| `Path` | `string` | `"./logs/app.log"` | 日志文件路径（`LogInFile` 时不可为空） |
| `Level` | `golog.Level` | `InfoLevel` | 日志级别 |
| `MaxAgeDays` | `int` | `30` | 备份最长保留天数 |
| `MaxBackups` | `int` | `0` | 最多保留备份个数（0=仅按 MaxAge） |
| `RotationHours` | `int` | `24` | 时间轮转间隔（小时），0 禁用；仅在有写入时触发 |
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
- 文件轮转基于 [timberjack](https://github.com/DeRuina/timberjack)；备份命名为 `app.log-YYYYMMDDHHmmss-size|time`（本地时区）。
- `NewLogger` 返回 `(Logger, error)`；新增 `MustNewLogger`、`Close()`。
- `Sync()` **仅刷盘**；关闭文件请用 `Close()` / `golog.Close()`。
- JSON `level` 字段为标准大写（如 `"INFO"`），不再使用 `[INFO]`。
- 包 `init` 不再自动创建默认 Logger；首次全局调用时懒加载。
- `SetDefaultLogger` 使用 `Clone()` 并接管文件 writer 所有权；替换时关闭旧 default。
- 彩色输出仅作用于**控制台**；文件始终为纯文本。

## 注意事项

- **同进程同路径自动单例**：多次 `NewLogger` 写同一 `Path` 会复用同一写入器（**首次配置生效**；后续不同配置会打 stderr 警告并忽略）。
- **多进程**写同一日志文件仍不支持。
- 容器部署请使用**绝对路径**配置 `Path`。
- 进程退出前请调用 `golog.Close()`（或持有所有权的 `logger.Close()`）。

## 示例

```bash
go run ./example/basic        # 控制台 + 动态级别
go run ./example/custom       # 文件/控制台 + Context
go run ./example/jsonformat   # JSON 输出
go run ./example/rotation     # 按大小轮转压测（生成多个备份文件）
```

## 授权许可

本项目采用 Apache 许可证 2.0 版本，详见 [LICENSE](https://www.apache.org/licenses/LICENSE-2.0.txt)。
