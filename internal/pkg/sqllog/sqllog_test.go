package sqllog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"workbench/internal/config"
)

func initTestLog(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{
		Log: config.Log{
			Dir: dir,
		},
	}
	if err := Init(cfg); err != nil {
		t.Fatalf("Init sqllog failed: %v", err)
	}
	return filepath.Join(dir, "sql.log")
}

// TestSQLLog_SQLCanaryAbsent 验证 synthetic 密码/email/token/正文无论在 Raw 还是 GORM 形式下均不在日志中出现。
func TestSQLLog_SQLCanaryAbsent(t *testing.T) {
	logPath := initTestLog(t)

	canaries := []string{
		"CANARY_SECRET_PASSWORD_12345",
		"canary.user@confidential-corp.internal",
		"tok_canary_live_abcdef9876543210",
		"Confidential requirement text CANARY_BODY_XYZ",
		"CANARY_COMMENT_TOKEN_IN_SQL",
	}

	queries := []string{
		fmt.Sprintf("SELECT id, account FROM `zt_user` WHERE `account` = 'admin' AND `password` = '%s'", canaries[0]),
		fmt.Sprintf("INSERT INTO `zt_user` (`account`, `email`, `token`) VALUES ('alice', '%s', '%s')", canaries[1], canaries[2]),
		fmt.Sprintf("UPDATE `zt_demand` SET `content` = '%s' WHERE `id` = 123", canaries[3]),
		fmt.Sprintf("/* caller comment: %s */ SELECT 1 FROM `zt_user` WHERE `id` = 99", canaries[4]),
	}

	ctx := context.Background()
	for _, q := range queries {
		LogQuery(ctx, q, 10*time.Millisecond, 1, nil)
	}

	if err := Sync(); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log failed: %v", err)
	}
	content := string(data)

	for _, canary := range canaries {
		if strings.Contains(content, canary) {
			t.Fatalf("SQL canary leaked in log: %q", canary)
		}
	}

	// Raw SQL with literals is intentionally fingerprint-only, never parsed heuristically.
	if strings.Count(content, "SQL fingerprint sha256:") != len(queries) {
		t.Fatal("raw literal queries must retain only fingerprints")
	}
}

// TestSQLLog_ErrorCanaryAbsent 验证数据库驱动错误中包含的参数/输入在日志中被安全分类，不泄露 canary。
func TestSQLLog_ErrorCanaryAbsent(t *testing.T) {
	logPath := initTestLog(t)

	errCanary := "CANARY_USER_INPUT_FOR_DUPLICATE_KEY_999"
	driverErr := fmt.Errorf("Error 1062 (23000): Duplicate entry '%s' for key 'zt_user.account'", errCanary)

	ctx := context.Background()
	LogQuery(ctx, "INSERT INTO `zt_user` (`account`) VALUES (?)", 5*time.Millisecond, 0, driverErr)

	if err := Sync(); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log failed: %v", err)
	}
	content := string(data)

	if strings.Contains(content, errCanary) {
		t.Fatalf("error canary leaked in log: %q", errCanary)
	}
	if !strings.Contains(content, "duplicate entry") {
		t.Fatalf("expected safe error category 'duplicate entry' not found in log: %s", content)
	}
}

// TestSQLLog_MetricsPreserved 验证普通查询记录时 count/time/request_id/seq/rows/file 等性能度量完整保留。
func TestSQLLog_MetricsPreserved(t *testing.T) {
	logPath := initTestLog(t)

	reqID := "req_test_123456"
	state := &RequestState{
		RequestID: reqID,
		Method:    "GET",
	}
	ctx := WithRequestState(context.Background(), state)

	LogQuery(ctx, "SELECT id, name FROM `zt_project` WHERE `id` = 10", 15*time.Millisecond, 5, nil)
	LogRequestSummary(state, "/home", 25*time.Millisecond)

	if err := Sync(); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log failed: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, reqID) {
		t.Fatalf("request_id missing from log: %s", content)
	}
	if !strings.Contains(content, `"rows":5`) {
		t.Fatalf("rows metric missing from log: %s", content)
	}
	if !strings.Contains(content, `"seq":1`) {
		t.Fatalf("seq metric missing from log: %s", content)
	}
	if !strings.Contains(content, `"sql_count":1`) {
		t.Fatalf("sql_count missing from summary in log: %s", content)
	}
	if !strings.Contains(content, `"route":"/home"`) {
		t.Fatalf("route missing from summary in log: %s", content)
	}
}
