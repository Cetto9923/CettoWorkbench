// =============================================================================
// 文件: internal/config/loader.go
// 模块: 基础设施
// 类型: infra
// 职责: 加载并解析配置文件，返回统一配置对象。
// 依赖: 无
// =============================================================================

package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Load 从文件加载配置，并允许使用环境变量覆盖（前缀 GOFRAMEWORK_，层级用下划线）。
// GOFRAMEWORK_MODE=dev 时读取 configs/config.dev.yaml，否则读取 configs/config.yaml。
func Load() (*Config, error) {
	configPath := "configs/config.yaml"
	if os.Getenv("GOFRAMEWORK_MODE") == "dev" {
		configPath = "configs/config.dev.yaml"
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetEnvPrefix("goframework")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetDefault("ratelimit.globalRPS", 100)
	v.SetDefault("upload.maxSizeMB", 10)
	v.SetDefault("upload.localDir", "uploads")
	v.SetDefault("upload.allowedTypes", []string{"image/jpeg", "image/png", "application/pdf"})
	v.SetDefault("backup.dir", "backups")
	v.SetDefault("backup.keepDays", 30)
	v.SetDefault("backup.autoCron", "0 2 * * *")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
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
	if strings.TrimSpace(cfg.Backup.Dir) == "" {
		cfg.Backup.Dir = "backups"
	}
	if cfg.Backup.KeepDays <= 0 {
		cfg.Backup.KeepDays = 30
	}
	if strings.TrimSpace(cfg.Backup.AutoCron) == "" {
		cfg.Backup.AutoCron = "0 2 * * *"
	}
	return &cfg, nil
}
