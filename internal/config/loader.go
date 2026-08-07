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

// Load 从文件加载配置，并允许使用环境变量覆盖（前缀 workbench_，层级用下划线）。
// workbench_MODE=dev 时读取 configs/config.dev.yaml，否则读取 configs/config.yaml。
func Load() (*Config, error) {
	configPath := "configs/config.yaml"
	if os.Getenv("WORKBENCH_MODE") == "dev" {
		configPath = "configs/config.dev.yaml"
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetEnvPrefix("workbench")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetDefault("ratelimit.globalRPS", 100)
	v.SetDefault("upload.maxSizeMB", 10)
	v.SetDefault("upload.localDir", "uploads")
	v.SetDefault("upload.allowedTypes", []string{"image/jpeg", "image/png", "application/pdf"})
	v.SetDefault("databaseReadonly.sessionVariables", "ob_read_consistency=Weak")

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
	return &cfg, nil
}
