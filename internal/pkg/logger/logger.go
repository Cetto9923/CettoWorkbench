// =============================================================================
// 文件: internal/pkg/logger/logger.go
// 模块: 基础设施
// 类型: infra
// 职责: 初始化并配置全局日志组件。
// 依赖: internal/config
// =============================================================================

package logger

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"workbench/internal/config"
)

// Init 根据配置初始化 Zap：dev 为 Development（可读、带颜色），prod 为 Production（JSON）。
func Init(cfg *config.Config) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(cfg.Log.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	isProd := strings.EqualFold(cfg.App.Env, "prod")
	if isProd {
		zcfg := zap.NewProductionConfig()
		zcfg.Level = zap.NewAtomicLevelAt(level)
		return zcfg.Build()
	}

	zcfg := zap.NewDevelopmentConfig()
	zcfg.Level = zap.NewAtomicLevelAt(level)
	return zcfg.Build()
}
