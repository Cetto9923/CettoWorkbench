// =============================================================================
// 文件: internal/pkg/database/database.go
// 模块: 基础设施
// 类型: infra
// 职责: 初始化数据库连接并提供关闭能力。
// 依赖: internal/config
// =============================================================================

package database

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"go.uber.org/zap"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"workbrench/internal/config"
)

const (
	maxOpenConns    = 50
	maxIdleConns    = 10
	connMaxLifetime = time.Hour
	slowThreshold   = 200 * time.Millisecond
)

// New 根据配置初始化 GORM MySQL 连接。
func New(cfg *config.Config, logger *zap.Logger) (*gorm.DB, error) {
	dsn := buildDSN(cfg)
	gormCfg := &gorm.Config{
		Logger: newGormZapLogger(logger),
	}

	db, err := gorm.Open(gormmysql.Open(dsn), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return db, nil
}

// Close 关闭底层数据库连接。
func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func buildDSN(cfg *config.Config) string {
	parseTime := "false"
	if cfg.Database.ParseTime {
		parseTime = "true"
	}
	loc := url.QueryEscape(cfg.Database.Loc)
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%s&loc=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
		cfg.Database.Charset,
		parseTime,
		loc,
	)
}

type gormZapLogger struct {
	logger *zap.Logger
	level  gormlogger.LogLevel
}

func newGormZapLogger(logger *zap.Logger) gormlogger.Interface {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &gormZapLogger{
		logger: logger,
		level:  gormlogger.Warn,
	}
}

func (l *gormZapLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return &gormZapLogger{
		logger: l.logger,
		level:  level,
	}
}

func (l *gormZapLogger) Info(context.Context, string, ...interface{}) {}

func (l *gormZapLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.level < gormlogger.Warn {
		return
	}
	l.logger.Warn(fmt.Sprintf(msg, data...))
}

func (l *gormZapLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.level < gormlogger.Error {
		return
	}
	l.logger.Error(fmt.Sprintf(msg, data...))
}

func (l *gormZapLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level == gormlogger.Silent {
		return
	}
	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && l.level >= gormlogger.Error:
		l.logger.Error(
			"gorm query error",
			zap.Error(err),
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
		)
	case elapsed > slowThreshold && l.level >= gormlogger.Warn:
		l.logger.Warn(
			"gorm slow query",
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
		)
	}
}
