package golog

// 默认配置
const (
	defaultMaxAgeDays     = 30  // 日志最长保存时间，单位：天
	defaultRotationHours  = 24  // 日志滚动间隔，单位：小时
	defaultRotationSizeMB = 100 // 默认的日志滚动大小，单位：MB
)

// Config 日志配置结构体
type Config struct {
	Module          string // 模块名称（显示在日志中）
	Component       string // 组件名称（显示在日志中）
	Path            string // 日志文件存储路径（如：/var/log/app.log）
	Level           Level  // 日志级别（DebugLevel/InfoLevel/WarnLevel/ErrorLevel）
	MaxAgeDays      int    // 日志文件最长保留天数（超过将自动删除）
	RotationHours   int    // 日志滚动时间间隔（小时）
	RotationSizeMB  int64  // 日志滚动大小阈值（单位：MB）
	JsonFormat      bool   // 是否使用JSON格式输出日志
	ShowLine        bool   // 是否显示调用文件名和行号
	LogInConsole    bool   // 是否同时在控制台输出日志
	ShowColor       bool   // 是否在控制台显示彩色日志
	IsBrief         bool   // 是否启用简洁模式（不显示级别、调用位置等信息）
	StackTraceLevel Level  // 当日志级别>=该级别时记录调用堆栈
}

// defaultConfig 默认日志配置
func defaultConfig() *Config {
	return &Config{
		Path:            "./log/app.log",
		Level:           DebugLevel,
		MaxAgeDays:      defaultMaxAgeDays,
		RotationHours:   defaultRotationHours,
		RotationSizeMB:  defaultRotationSizeMB,
		JsonFormat:      false,
		ShowLine:        true,
		LogInConsole:    true,
		ShowColor:       false,
		IsBrief:         false,
		StackTraceLevel: ErrorLevel,
	}
}

type Option func(c *Config)

func NewConfig(opts ...Option) *Config {
	conf := defaultConfig()
	for _, opt := range opts {
		opt(conf)
	}
	return conf
}

func WithModule(module string) Option {
	return func(c *Config) {
		c.Module = module
	}
}

func WithComponent(component string) Option {
	return func(c *Config) {
		c.Component = component
	}
}

func WithPath(path string) Option {
	return func(c *Config) {
		c.Path = path
	}
}

func WithLevel(level Level) Option {
	return func(c *Config) {
		c.Level = level
	}
}

func WithMaxAgeDays(days int) Option {
	return func(c *Config) {
		c.MaxAgeDays = days
	}
}

func WithRotationHours(hours int) Option {
	return func(c *Config) {
		c.RotationHours = hours
	}
}

func WithRotationSizeMB(mb int64) Option {
	return func(c *Config) {
		c.RotationSizeMB = mb
	}
}

func WithJsonFormat(jsonFormat bool) Option {
	return func(c *Config) {
		c.JsonFormat = jsonFormat
	}
}

func WithShowLine(showLine bool) Option {
	return func(c *Config) {
		c.ShowLine = showLine
	}
}

func WithLogInConsole(logInConsole bool) Option {
	return func(c *Config) {
		c.LogInConsole = logInConsole
	}
}

func WithShowColor(showColor bool) Option {
	return func(c *Config) {
		c.ShowColor = showColor
	}
}

func WithIsBrief(isBrief bool) Option {
	return func(c *Config) {
		c.IsBrief = isBrief
	}
}

func WithStackTraceLevel(level Level) Option {
	return func(c *Config) {
		c.StackTraceLevel = level
	}
}
