// =============================================================================
// 文件: internal/config/loader.go
// 模块: 基础设施
// 类型: infra
// 职责: 加载并解析配置文件，支持环境覆盖与安全校验。
// 依赖: github.com/spf13/viper
// =============================================================================

package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Load 从文件加载配置，并允许使用环境变量覆盖（前缀 workbench_，层级用下划线）。
// 运行模式与路径选择保留现有 WORKBENCH_MODE 兼容合同。
func Load() (*Config, error) {
	configPath := "configs/config.yaml"
	if os.Getenv("WORKBENCH_MODE") == "dev" {
		configPath = "configs/config.dev.yaml"
	}

	return LoadFromPath(configPath)
}

// LoadFromPath 从指定路径加载配置并执行验证。
func LoadFromPath(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetEnvPrefix("workbench")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 显式绑定支持的配置键，确保环境变量（即使文件中缺失）能生效
	bindEnvKeys(v)

	v.SetDefault("app.name", "workbench")
	v.SetDefault("session.cookieName", "workbench_sid")
	v.SetDefault("session.lifetimeHours", 2)
	v.SetDefault("ratelimit.globalRPS", 100)
	v.SetDefault("upload.maxSizeMB", 10)
	v.SetDefault("upload.localDir", "uploads")
	v.SetDefault("upload.allowedTypes", []string{"image/jpeg", "image/png", "application/pdf"})
	v.SetDefault("databaseReadonly.sessionVariables", "ob_read_consistency=Weak")

	if err := v.ReadInConfig(); err != nil {
		return nil, errors.New("read config file failed: check file availability and syntax")
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, errors.New("unmarshal config failed: check configuration field types")
	}

	// 补全默认值
	if cfg.RateLimit.GlobalRPS <= 0 {
		cfg.RateLimit.GlobalRPS = 100
	}
	if cfg.Upload.MaxSizeMB <= 0 {
		cfg.Upload.MaxSizeMB = 10
	}
	if strings.TrimSpace(cfg.Upload.LocalDir) == "" {
		cfg.Upload.LocalDir = "uploads"
	}
	if len(cfg.Upload.AllowedTypes) == 0 {
		cfg.Upload.AllowedTypes = []string{"image/jpeg", "image/png", "application/pdf"}
	}

	if err := Validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func bindEnvKeys(v *viper.Viper) {
	keys := []string{
		"app.name", "app.env", "app.addr",
		"database.host", "database.port", "database.user", "database.password",
		"database.dbname", "database.charset", "database.loc", "database.parseTime", "database.sessionVariables",
		"databaseReadonly.host", "databaseReadonly.port", "databaseReadonly.user", "databaseReadonly.password",
		"databaseReadonly.dbname", "databaseReadonly.charset", "databaseReadonly.loc", "databaseReadonly.parseTime", "databaseReadonly.sessionVariables",
		"zentao.url", "zentao.api", "zentao.account", "zentao.password", "zentao.requestType",
		"session.cookieName", "session.lifetimeHours", "session.cookieSecure",
		"log.level", "log.dir",
		"layout.nav",
		"ratelimit.globalRPS",
		"upload.maxSizeMB", "upload.localDir",
	}
	for _, k := range keys {
		_ = v.BindEnv(k)
	}
}

// Validate 校验配置完整性与安全性约束。
// 错误信息不包含任何秘密明文。
func Validate(cfg *Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}

	env := strings.ToLower(strings.TrimSpace(cfg.App.Env))
	if env == "" {
		return errors.New("missing required config: app.env")
	}
	switch env {
	case "prod", "production", "dev", "development", "test", "local":
		// valid
	default:
		return errors.New("invalid app.env: expected prod/dev/test/local")
	}

	if strings.TrimSpace(cfg.Database.Host) == "" {
		return errors.New("missing required config: database.host")
	}
	if cfg.Database.Port < 1 || cfg.Database.Port > 65535 {
		return fmt.Errorf("invalid database.port: %d (expected 1-65535)", cfg.Database.Port)
	}
	if strings.TrimSpace(cfg.Database.User) == "" {
		return errors.New("missing required config: database.user")
	}
	if strings.TrimSpace(cfg.Database.DBName) == "" {
		return errors.New("missing required config: database.dbname")
	}
	zentaoURL, err := url.ParseRequestURI(strings.TrimSpace(cfg.Zentao.URL))
	if err != nil || zentaoURL.Scheme == "" || zentaoURL.Host == "" || (zentaoURL.Scheme != "http" && zentaoURL.Scheme != "https") {
		return errors.New("invalid required config: zentao.url must be an absolute http(s) URL")
	}
	if api := strings.TrimSpace(cfg.Zentao.API); api != "" {
		apiURL, err := url.ParseRequestURI(api)
		if err != nil || apiURL.Scheme == "" || apiURL.Host == "" || (apiURL.Scheme != "http" && apiURL.Scheme != "https") {
			return errors.New("invalid config: zentao.api must be an absolute http(s) URL when set")
		}
	}

	// Optional replica connection errors are handled by bootstrap degradation.

	if strings.TrimSpace(cfg.Session.CookieName) == "" {
		return errors.New("missing required config: session.cookieName")
	}
	if cfg.Session.LifetimeHours <= 0 {
		return errors.New("invalid session.lifetimeHours: must be positive")
	}

	// 生产模式拒绝不安全配置
	if env == "prod" || env == "production" {
		if !cfg.Session.CookieSecure {
			return errors.New("insecure production configuration: session.cookieSecure must be true in prod")
		}
		if strings.TrimSpace(cfg.Database.Password) == "" {
			return errors.New("insecure production configuration: database.password cannot be empty in prod")
		}
	}

	return nil
}
