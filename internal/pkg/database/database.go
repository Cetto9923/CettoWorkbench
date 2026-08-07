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
	"strings"
	"time"

	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"workbench/internal/config"
	"workbench/internal/pkg/sqllog"
)

const (
	maxOpenConns    = 50
	maxIdleConns    = 10
	connMaxLifetime = time.Hour
)

// New 根据主库配置初始化 GORM MySQL 连接。
func New(cfg *config.Config) (*gorm.DB, error) {
	return Open(cfg.Database)
}

// Open 按给定 Database 配置打开 GORM 连接（主库 / 只读备库均可）。
func Open(dbCfg config.Database) (*gorm.DB, error) {
	dsn := buildDSN(dbCfg)
	gormCfg := &gorm.Config{
		Logger: newGormSQLLogger(),
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
		_ = sqlDB.Close()
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

func buildDSN(dbCfg config.Database) string {
	parseTime := "false"
	if dbCfg.ParseTime {
		parseTime = "true"
	}
	loc := url.QueryEscape(dbCfg.Loc)
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%s&loc=%s",
		dbCfg.User,
		dbCfg.Password,
		dbCfg.Host,
		dbCfg.Port,
		dbCfg.DBName,
		dbCfg.Charset,
		parseTime,
		loc,
	)
	// 配置可写 JDBC 风格 "ob_read_consistency=Weak"；Go mysql 驱动无 sessionVariables 参数，
	// 需拆成系统变量 DSN：&ob_read_consistency=%27Weak%27 → SET ob_read_consistency = 'Weak'
	return dsn + appendSessionVarParams(dbCfg.SessionVariables)
}

// appendSessionVarParams 将 sessionVariables 转为 go-sql-driver 可识别的系统变量查询参数。
func appendSessionVarParams(sessionVariables string) string {
	sv := strings.TrimSpace(sessionVariables)
	if sv == "" {
		return ""
	}
	var b strings.Builder
	for _, pair := range strings.Split(sv, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		key, val, ok := strings.Cut(pair, "=")
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if !ok || key == "" {
			continue
		}
		// 非数字/布尔的字符串变量需加单引号，否则 OceanBase 会把右侧当成列名
		if !isBareSQLLiteral(val) {
			val = "'" + strings.Trim(val, "'") + "'"
		}
		b.WriteByte('&')
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(url.QueryEscape(val))
	}
	return b.String()
}

func isBareSQLLiteral(val string) bool {
	if val == "" {
		return true
	}
	if strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'") {
		return true
	}
	switch strings.ToLower(val) {
	case "0", "1", "true", "false", "on", "off":
		return true
	}
	for _, r := range val {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

type gormSQLLogger struct {
	level gormlogger.LogLevel
}

func newGormSQLLogger() gormlogger.Interface {
	return &gormSQLLogger{level: gormlogger.Info}
}

func (l *gormSQLLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return &gormSQLLogger{level: level}
}

func (l *gormSQLLogger) Info(context.Context, string, ...interface{}) {}

func (l *gormSQLLogger) Warn(context.Context, string, ...interface{}) {}

func (l *gormSQLLogger) Error(context.Context, string, ...interface{}) {}

func (l *gormSQLLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level == gormlogger.Silent {
		return
	}
	elapsed := time.Since(begin)
	sql, rows := fc()
	sqllog.LogQuery(ctx, sql, elapsed, rows, err)
}
