package golog

// 日志库内部使用的缺省配置常量
const (
	defaultMaxAgeDays         = 30  // 默认日志最长保留天数（单位：天）
	defaultRotationSizeMB     = 100 // 默认单日志文件大小滚动阈值（单位：MB）
	defaultSamplingInitial    = 100 // 采样模式下：每秒前 N 条日志全量记录
	defaultSamplingThereafter = 100 // 采样模式下：超出阈值后每 M 条记录 1 条
)

// SamplingConfig 定义高流量场景下的日志采样控制策略。
type SamplingConfig struct {
	Enabled    bool // 是否启用采样限流
	Initial    int  // 每秒内前 N 条相同消息日志全量记录
	Thereafter int  // 超出 Initial 阈值后，后续每 M 条相同消息仅记录 1 条
}

// Config 定义创建 Logger 所需的完整配置选项。
type Config struct {
	Module          string         // 模块名称（输出时标识来源系统或模块）
	Component       string         // 组件名称（输出时以 @component 格式展示）
	Path            string         // 日志文件存储路径（如：./logs/app.log，LogInFile=true 时必填）
	Level           Level          // 最低输出日志级别（DebugLevel/InfoLevel/WarnLevel/ErrorLevel/FatalLevel）
	MaxAgeDays      int            // 旧日志文件最长保留天数（超过该天数的文件将被自动清理）
	MaxBackups      int            // 最多保留的旧日志备份文件个数（0 表示不限制数量，仅按 MaxAgeDays 清理）
	RotationSizeMB  int64          // 单个日志文件按大小滚动的触发阈值（单位：MB）
	Compress        bool           // 是否对滚动生成的旧日志文件进行 gzip 压缩
	JSONFormat      bool           // 是否以 JSON 结构化格式输出日志
	UseUTC          bool           // JSON 格式下的时间戳是否转换为 UTC 时区（RFC3339Nano）
	ShowLine        bool           // 是否在日志中打印调用者的源码文件名与精确代码行号
	LogInFile       bool           // 是否将日志持久化写入到文件
	LogInConsole    bool           // 是否将日志同步输出到终端控制台（stderr）
	ShowColor       bool           // 控制台输出时是否对日志级别字段启用 ANSI 彩色渲染（不影响文件）
	Brief           bool           // 是否启用极简模式（隐藏模块、组件、级别和调用行号，仅保留时间和正文）
	StackTraceLevel Level          // 当日志级别大于或等于该级别时自动追加调用堆栈跟踪（默认 ErrorLevel）
	Sampling        SamplingConfig // 高并发高流量场景下的日志采样配置
}

// defaultConfig 返回日志库的默认基础配置。
func defaultConfig() *Config {
	return &Config{
		Path:            "./logs/app.log",
		Level:           InfoLevel,
		MaxAgeDays:      defaultMaxAgeDays,
		RotationSizeMB:  defaultRotationSizeMB,
		JSONFormat:      false,
		UseUTC:          true,
		ShowLine:        true,
		LogInFile:       false,
		LogInConsole:    true,
		ShowColor:       false,
		Brief:           false,
		StackTraceLevel: ErrorLevel,
		Sampling: SamplingConfig{
			Enabled:    false,
			Initial:    defaultSamplingInitial,
			Thereafter: defaultSamplingThereafter,
		},
	}
}

// Option 定义修改 Config 的函数选项类型。
type Option func(c *Config)

// NewConfig 根据传入的一系列 Option 函数构造并返回 Config 指针，未指定的项将采用推荐默认值。
func NewConfig(opts ...Option) *Config {
	conf := defaultConfig()
	for _, opt := range opts {
		opt(conf)
	}
	return conf
}

// WithModule 设置模块名称。
func WithModule(module string) Option {
	return func(c *Config) {
		c.Module = module
	}
}

// WithComponent 设置组件名称（日志中展示为 @component）。
func WithComponent(component string) Option {
	return func(c *Config) {
		c.Component = component
	}
}

// WithPath 设置日志文件持久化存储路径。
func WithPath(path string) Option {
	return func(c *Config) {
		c.Path = path
	}
}

// WithLevel 设置最低日志输出级别。
func WithLevel(level Level) Option {
	return func(c *Config) {
		c.Level = level
	}
}

// WithMaxAgeDays 设置旧日志文件的最长保留天数。
func WithMaxAgeDays(days int) Option {
	return func(c *Config) {
		c.MaxAgeDays = days
	}
}

// WithMaxBackups 设置最多保留的备份文件数量。
func WithMaxBackups(backups int) Option {
	return func(c *Config) {
		c.MaxBackups = backups
	}
}

// WithRotationSizeMB 设置按文件大小轮转的阈值（单位：MB）。
func WithRotationSizeMB(mb int64) Option {
	return func(c *Config) {
		c.RotationSizeMB = mb
	}
}

// WithCompress 设置是否对滚动归档后的旧日志文件进行 gzip 压缩。
func WithCompress(compress bool) Option {
	return func(c *Config) {
		c.Compress = compress
	}
}

// WithJSONFormat 设置是否使用 JSON 格式输出日志。
func WithJSONFormat(jsonFormat bool) Option {
	return func(c *Config) {
		c.JSONFormat = jsonFormat
	}
}

// WithUseUTC 设置在 JSON 格式下时间戳是否使用 UTC 时区。
func WithUseUTC(useUTC bool) Option {
	return func(c *Config) {
		c.UseUTC = useUTC
	}
}

// WithShowLine 设置是否输出调用方源码文件名和精确行号。
func WithShowLine(showLine bool) Option {
	return func(c *Config) {
		c.ShowLine = showLine
	}
}

// WithLogInFile 设置是否将日志写入文件。
func WithLogInFile(logInFile bool) Option {
	return func(c *Config) {
		c.LogInFile = logInFile
	}
}

// WithLogInConsole 设置是否将日志同步输出至控制台（stderr）。
func WithLogInConsole(logInConsole bool) Option {
	return func(c *Config) {
		c.LogInConsole = logInConsole
	}
}

// WithShowColor 设置控制台输出时是否对日志级别字段启用 ANSI 彩色渲染。
func WithShowColor(showColor bool) Option {
	return func(c *Config) {
		c.ShowColor = showColor
	}
}

// WithBrief 设置是否启用极简模式（不输出级别、模块与行号信息）。
func WithBrief(brief bool) Option {
	return func(c *Config) {
		c.Brief = brief
	}
}

// WithStackTraceLevel 设置触发自动打印堆栈跟踪的最低日志级别。
func WithStackTraceLevel(level Level) Option {
	return func(c *Config) {
		c.StackTraceLevel = level
	}
}

// WithSampling 设置高流量下的日志采样限流配置。
func WithSampling(s SamplingConfig) Option {
	return func(c *Config) {
		c.Sampling = s
	}
}
