# golog

`golog` 是一个基于 Uber Zap 的增强型日志库，旨在简化日志输出的实现和使用。

## 核心特性

- 🕒 **智能轮转策略**：支持双维度切割（时间 + 大小）。
- 🎭 **多格式输出**：自由切换 JSON 和文本格式。
- 📊 **分级管理**：提供 5 级日志分类（DEBUG / INFO / WARN / ERROR / FATAL）。
- 🌈 **终端友好**：支持 ANSI 彩色输出，提升可读性。
- 🔧 **动态调整**：可在运行时修改日志级别。
- 📂 **多端同步**：支持文件、控制台及自定义输出。
## 安装

```bash
go get github.com/treeforest/golog
```

## 快速开始

### 基础使用

```go
import "github.com/treeforest/golog"

defer golog.Sync()

golog.Debug("debug message")
golog.Info("info message")
golog.Warn("warn message")
golog.Error("error message")

// 运行时修改日志级别
golog.SetLevel(golog.WarnLevel)

golog.Debug("debug message") // 不会输出
golog.Info("info message")   // 不会输出
golog.Warn("warn message")
golog.Error("error message")
```

### 日志配置参数

| 参数名           | 类型            | 默认值               | 说明                      |
|------------------|---------------|-------------------|-------------------------|
| `Module`         | `string`      | `""`              | 模块名称（显示在日志中）            |
| `Component`      | `string`      | `""`              | 组件名称（显示在日志中）            |
| `Path`           | `string`      | `"./log/app.log"` | 日志文件存储路径                |
| `Level`          | `golog.Level` | `DebugLevel`      | 日志级别                    |
| `MaxAgeDays`     | `int`         | `30`              | 日志文件最长保留天数（超过将自动删除）     |
| `RotationHours`  | `int`         | `24`              | 日志滚动时间间隔（小时）            |
| `RotationSizeMB` | `int64`       | `100`             | 日志滚动大小阈值（单位：MB）         |
| `JsonFormat`     | `bool`        | `false`           | 是否使用JSON格式输出日志          |
| `ShowLine`       | `bool`        | `true`            | 是否显示调用文件名和行号            |
| `LogInConsole`   | `bool`        | `true`            | 是否同时在控制台输出日志            |
| `ShowColor`      | `bool`        | `false`           | 是否在控制台显示彩色日志            |
| `IsBrief`        | `bool`        | `false`           | 是否启用简洁模式（不显示级别、调用位置等信息） |
| `StackTraceLevel`| `golog.Level`       | `ErrorLevel`      | 当日志级别 >= 该级别时记录调用堆栈     |

## 授权许可

本项目采用 Apache 许可证 2.0 版本，详细信息请参见 [LICENSE](https://www.apache.org/licenses/LICENSE-2.0.txt)