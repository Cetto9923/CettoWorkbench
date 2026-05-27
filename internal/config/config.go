// =============================================================================
// 文件: internal/config/config.go
// 模块: 基础设施
// 类型: infra
// 职责: 定义应用配置结构体与配置项映射。
// 依赖: 无
// =============================================================================

package config

// Config 根配置，字段名与 YAML 键一致。
type Config struct {
	App      App          `mapstructure:"app"`
	Database Database     `mapstructure:"database"`
	Zentao   ZentaoConfig `mapstructure:"zentao"`
	Session  Session      `mapstructure:"session"`
	Log      Log          `mapstructure:"log"`
	Layout   Layout       `mapstructure:"layout"`

	// 内置配置
	RateLimit RateLimit `mapstructure:"ratelimit"`
	Upload    Upload    `mapstructure:"upload"`
	Backup    Backup    `mapstructure:"backup"`
}

// App 应用基础信息。
type App struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
	Addr string `mapstructure:"addr"`
}

// Database 数据库连接参数。
type Database struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	User      string `mapstructure:"user"`
	Password  string `mapstructure:"password"`
	DBName    string `mapstructure:"dbname"`
	Charset   string `mapstructure:"charset"`
	Loc       string `mapstructure:"loc"`
	ParseTime bool   `mapstructure:"parseTime"`
}

// ZentaoConfig 禅道配置。
type ZentaoConfig struct {
	URL      string `mapstructure:"url"`
	Account  string `mapstructure:"account"`
	Password string `mapstructure:"password"`
}

// Session 会话相关。
type Session struct {
	CookieName    string `mapstructure:"cookieName"`
	LifetimeHours int    `mapstructure:"lifetimeHours"`
}

// Log 日志。
type Log struct {
	Level string `mapstructure:"level"`
	Dir   string `mapstructure:"dir"`
}

// Layout 布局配置。
type Layout struct {
	Nav string `mapstructure:"nav"`
}

// RateLimit 限流配置。
type RateLimit struct {
	GlobalRPS int `mapstructure:"globalRPS"`
}

// Upload 上传配置。
type Upload struct {
	MaxSizeMB    int      `mapstructure:"maxSizeMB"`
	AllowedTypes []string `mapstructure:"allowedTypes"`
	LocalDir     string   `mapstructure:"localDir"`
}

// Backup 备份配置。
type Backup struct {
	Dir      string `mapstructure:"dir"`
	KeepDays int    `mapstructure:"keepDays"`
	AutoCron string `mapstructure:"autoCron"`
}
