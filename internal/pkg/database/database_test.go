package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gormlogger "gorm.io/gorm/logger"

	"workbench/internal/config"
	"workbench/internal/pkg/sqllog"
)

func initTestSQLLog(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{
		Log: config.Log{
			Dir: dir,
		},
	}
	if err := sqllog.Init(cfg); err != nil {
		t.Fatalf("Init sqllog failed: %v", err)
	}
	return filepath.Join(dir, "sql.log")
}

// TestDatabase_SQLCanaryAbsent 验证通过 GORM Logger Trace 接口写入的 SQL 中的敏感参数不在日志中出现。
func TestDatabase_SQLCanaryAbsent(t *testing.T) {
	logPath := initTestSQLLog(t)

	logger := newGormSQLLogger()
	canary := "CANARY_GORM_PASSWORD_SECRET_777"
	rawQuery := fmt.Sprintf("SELECT id, account FROM `zt_user` WHERE `password` = '%s'", canary)

	ctx := context.Background()
	logger.Trace(ctx, time.Now().Add(-10*time.Millisecond), func() (string, int64) {
		return rawQuery, 1
	}, nil)

	if err := sqllog.Sync(); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log failed: %v", err)
	}
	content := string(data)

	if strings.Contains(content, canary) {
		t.Fatalf("Trace leaked SQL canary: %q", canary)
	}
	if !strings.Contains(content, "SELECT id, account FROM `zt_user` WHERE `password` = ?") {
		t.Fatalf("expected parameterized query not found in log: %s", content)
	}
}

// TestDatabase_ErrorCanaryAbsent 验证通过 GORM Logger Trace 接口传入的驱动错误参数不泄露到日志。
func TestDatabase_ErrorCanaryAbsent(t *testing.T) {
	logPath := initTestSQLLog(t)

	logger := newGormSQLLogger()
	errCanary := "CANARY_GORM_USER_DUPLICATE_888"
	driverErr := fmt.Errorf("Error 1062: Duplicate entry '%s' for key 'PRIMARY'", errCanary)

	ctx := context.Background()
	logger.Trace(ctx, time.Now().Add(-5*time.Millisecond), func() (string, int64) {
		return "INSERT INTO `zt_user` (`id`) VALUES (1)", 0
	}, driverErr)

	if err := sqllog.Sync(); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log failed: %v", err)
	}
	content := string(data)

	if strings.Contains(content, errCanary) {
		t.Fatalf("Trace leaked error canary: %q", errCanary)
	}
	if !strings.Contains(content, "duplicate entry") {
		t.Fatalf("expected safe error category in log: %s", content)
	}
}

// TestDatabase_MetricsPreserved 验证通过 Trace 写入的性能度量（耗时、受影响行数）被正确记录。
func TestDatabase_MetricsPreserved(t *testing.T) {
	logPath := initTestSQLLog(t)

	logger := newGormSQLLogger()
	reqID := "req_trace_test_999"
	state := &sqllog.RequestState{
		RequestID: reqID,
		Method:    "GET",
	}
	ctx := sqllog.WithRequestState(context.Background(), state)

	start := time.Now().Add(-20 * time.Millisecond)
	logger.Trace(ctx, start, func() (string, int64) {
		return "SELECT id, title FROM `zt_story` WHERE `id` = 42", 3
	}, nil)

	if err := sqllog.Sync(); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log failed: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, reqID) {
		t.Fatalf("request_id missing from Trace log: %s", content)
	}
	if !strings.Contains(content, `"rows":3`) {
		t.Fatalf("rows count missing from Trace log: %s", content)
	}
}

func TestDatabase_SilentLogger(t *testing.T) {
	logPath := initTestSQLLog(t)

	logger := newGormSQLLogger().LogMode(gormlogger.Silent)
	logger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT 1", 1
	}, nil)

	_ = sqllog.Sync()

	data, err := os.ReadFile(logPath)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read log failed: %v", err)
	}
	if len(data) > 0 {
		t.Fatalf("silent logger should not produce log, got: %s", string(data))
	}
}
