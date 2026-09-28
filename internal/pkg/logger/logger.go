// =============================================================================
// 文件: internal/pkg/logger/logger.go
// 模块: 基础设施
// 类型: infra
// 职责: 初始化并配置全局日志组件。
// 依赖: internal/config
// =============================================================================

package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"workbench/internal/config"
)

// Init 根据配置初始化应用日志。
// dev：控制台（可读、带颜色）+ app-YYYY-MM-DD.log（可读、无颜色）；prod：app-YYYY-MM-DD.log（JSON）。
// log.dir 为空时仅输出到控制台。
func Init(cfg *config.Config) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(cfg.Log.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}
	atomLevel := zap.NewAtomicLevelAt(level)

	isProd := strings.EqualFold(cfg.App.Env, "prod")
	cores := make([]zapcore.Core, 0, 2)

	if !isProd {
		devCfg := zap.NewDevelopmentEncoderConfig()
		consoleEncoder := zapcore.NewConsoleEncoder(devCfg)
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stderr), atomLevel))
	}

	if dir := strings.TrimSpace(cfg.Log.Dir); dir != "" {
		fileCore, err := newDailyFileCore(dir, isProd, atomLevel)
		if err != nil {
			return nil, err
		}
		cores = append(cores, fileCore)
	}

	if len(cores) == 0 {
		encoderCfg := zap.NewProductionEncoderConfig()
		encoder := zapcore.NewJSONEncoder(encoderCfg)
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stderr), atomLevel))
	}

	core := zapcore.NewTee(cores...)
	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)), nil
}

func newDailyFileCore(dir string, isProd bool, level zap.AtomicLevel) (zapcore.Core, error) {
	syncer, err := newDailyWriteSyncer(dir, "app-", ".log")
	if err != nil {
		return nil, err
	}

	var encoder zapcore.Encoder
	if isProd {
		encoder = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	} else {
		devCfg := zap.NewDevelopmentEncoderConfig()
		devCfg.EncodeLevel = zapcore.CapitalLevelEncoder
		encoder = zapcore.NewConsoleEncoder(devCfg)
	}

	return zapcore.NewCore(encoder, zapcore.AddSync(syncer), level), nil
}
