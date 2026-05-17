package golog

// 默认配置
const (
	defaultMaxAgeDays         = 30  // 日志最长保存时间，单位：天
	defaultRotationHours      = 24  // 日志滚动间隔，单位：小时
	defaultRotationSizeMB     = 100 // 默认的日志滚动大小，单位：MB
	defaultSamplingInitial    = 100 // 采样：每秒前 N 条全量记录
	defaultSamplingThereafter = 100 // 采样：之后每 M 条记录 1 条
)

// SamplingConfig 日志采样配置（用于高流量场景限制磁盘写入）
type SamplingConfig struct {
	Enabled    bool // 是否启用采样
	Initial    int  // 每秒前 N 条全量记录
	Thereafter int  // 之后每 M 条记录 1 条
}

// Config 日志配置结构体
type Config struct {
	Module          string         // 模块名称（显示在日志中）
	Component       string         // 组件名称（显示在日志中）
	Path            string         // 日志文件存储路径（如：/var/log/app.log）
	Level           Level          // 日志级别（DebugLevel/InfoLevel/WarnLevel/ErrorLevel/FatalLevel）
	MaxAgeDays      int            // 日志文件最长保留天数（超过将自动删除）
	MaxBackups      int            // 最多保留备份文件个数（0 表示仅按 MaxAgeDays 清理）
	RotationHours   int            // 日志滚动时间间隔（小时），定时调用 lumberjack.Rotate
	RotationSizeMB  int64          // 日志滚动大小阈值（单位：MB）
	Compress        bool           // 是否对轮转后的旧日志进行 gzip 压缩
	JsonFormat      bool           // 是否使用 JSON 格式输出日志
	UseUTC          bool           // JSON 时间是否使用 UTC（RFC3339Nano）
	ShowLine        bool           // 是否显示调用文件名和行号
	LogInFile       bool           // 是否输出日志到文件
	LogInConsole    bool           // 是否同时在控制台输出日志
	ShowColor       bool           // 是否在控制台对级别字段显示彩色（不影响文件）
	IsBrief         bool           // 是否启用简洁模式（不显示级别、调用位置等信息）
	StackTraceLevel Level          // 当日志级别 >= 该级别时记录调用堆栈
	Sampling        SamplingConfig // 日志采样配置
}

// defaultConfig 返回默认日志配置
func defaultConfig() *Config {
	return &Config{
		Path:            "./logs/app.log",
		Level:           InfoLevel,
		MaxAgeDays:      defaultMaxAgeDays,
		RotationHours:   defaultRotationHours,
		RotationSizeMB:  defaultRotationSizeMB,
		JsonFormat:      false,
		UseUTC:          true,
		ShowLine:        true,
		LogInFile:       false,
		LogInConsole:    true,
		ShowColor:       false,
		IsBrief:         false,
		StackTraceLevel: ErrorLevel,
		Sampling: SamplingConfig{
			Enabled:    false,
			Initial:    defaultSamplingInitial,
			Thereafter: defaultSamplingThereafter,
		},
	}
}

// Option 配置项函数类型
type Option func(c *Config)

// NewConfig 根据选项创建配置（未指定的项使用默认值）
func NewConfig(opts ...Option) *Config {
	conf := defaultConfig()
	for _, opt := range opts {
		opt(conf)
	}
	return conf
}

// WithModule 设置模块名称
func WithModule(module string) Option {
	return func(c *Config) {
		c.Module = module
	}
}

// WithComponent 设置组件名称
func WithComponent(component string) Option {
	return func(c *Config) {
		c.Component = component
	}
}

// WithPath 设置日志文件路径
func WithPath(path string) Option {
	return func(c *Config) {
		c.Path = path
	}
}

// WithLevel 设置日志级别
func WithLevel(level Level) Option {
	return func(c *Config) {
		c.Level = level
	}
}

// WithMaxAgeDays 设置日志最长保留天数
func WithMaxAgeDays(days int) Option {
	return func(c *Config) {
		c.MaxAgeDays = days
	}
}

// WithMaxBackups 设置最多保留的备份文件个数
func WithMaxBackups(backups int) Option {
	return func(c *Config) {
		c.MaxBackups = backups
	}
}

// WithRotationHours 设置按时间轮转的间隔（小时）
func WithRotationHours(hours int) Option {
	return func(c *Config) {
		c.RotationHours = hours
	}
}

// WithRotationSizeMB 设置按大小轮转的阈值（MB）
func WithRotationSizeMB(mb int64) Option {
	return func(c *Config) {
		c.RotationSizeMB = mb
	}
}

// WithCompress 设置是否压缩旧日志
func WithCompress(compress bool) Option {
	return func(c *Config) {
		c.Compress = compress
	}
}

// WithJsonFormat 设置是否使用 JSON 格式
func WithJsonFormat(jsonFormat bool) Option {
	return func(c *Config) {
		c.JsonFormat = jsonFormat
	}
}

// WithUseUTC 设置 JSON 时间是否使用 UTC
func WithUseUTC(useUTC bool) Option {
	return func(c *Config) {
		c.UseUTC = useUTC
	}
}

// WithShowLine 设置是否显示调用位置
func WithShowLine(showLine bool) Option {
	return func(c *Config) {
		c.ShowLine = showLine
	}
}

// WithLogInFile 设置是否写入文件
func WithLogInFile(logInFile bool) Option {
	return func(c *Config) {
		c.LogInFile = logInFile
	}
}

// WithLogInConsole 设置是否输出到控制台
func WithLogInConsole(logInConsole bool) Option {
	return func(c *Config) {
		c.LogInConsole = logInConsole
	}
}

// WithShowColor 设置控制台是否彩色显示级别
func WithShowColor(showColor bool) Option {
	return func(c *Config) {
		c.ShowColor = showColor
	}
}

// WithIsBrief 设置是否启用简洁模式
func WithIsBrief(isBrief bool) Option {
	return func(c *Config) {
		c.IsBrief = isBrief
	}
}

// WithStackTraceLevel 设置记录堆栈的最低级别
func WithStackTraceLevel(level Level) Option {
	return func(c *Config) {
		c.StackTraceLevel = level
	}
}

// WithSampling 设置采样配置
func WithSampling(s SamplingConfig) Option {
	return func(c *Config) {
		c.Sampling = s
	}
}
