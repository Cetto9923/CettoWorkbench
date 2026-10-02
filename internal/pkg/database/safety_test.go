// =============================================================================
// 文件: internal/pkg/database/safety_test.go
// 模块: 数据库
// 类型: test
// 职责: 安装字段与运行期检查一致；SQL 日志不插入凭据值。
// =============================================================================

package database

import (
	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"workbench/internal/config"
	"workbench/internal/pkg/sqllog"
)

func TestSQLLoggerLeavesSensitiveParametersOut(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()
	dir := t.TempDir()
	if err := sqllog.Init(&config.Config{Log: config.Log{Dir: dir}}); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sqllog.Sync() }()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{Logger: newGormSQLLogger()})
	if err != nil {
		t.Fatal(err)
	}
	const credential = "synthetic-secret-value"
	mock.ExpectQuery(`SELECT account FROM zt_user WHERE password = \?`).WithArgs(credential).WillReturnRows(sqlmock.NewRows([]string{"account"}).AddRow("fixture"))
	var rows []struct{ Account string }
	if err := db.Raw("SELECT account FROM zt_user WHERE password = ?", credential).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if err := sqllog.Sync(); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "sql-*.log"))
	if err != nil || len(files) != 1 {
		t.Fatalf("SQL evidence file missing: %v %v", files, err)
	}
	log, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(log), credential) || !strings.Contains(string(log), "?") {
		t.Fatal("SQL log leaked a sensitive parameter")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSchemaCheckUsesInstallColumnsWithoutDDL(t *testing.T) {
	assertInstalledSchema(t, "")
	assertInstalledSchema(t, "actionType")
}

func installColumnRows(source []byte, missing string) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{"table_name", "column_name"})
	for _, table := range regexp.MustCompile("(?s)CREATE TABLE IF NOT EXISTS `([^`]+)` \\((.*?)\\) ENGINE").FindAllStringSubmatch(string(source), -1) {
		for _, column := range regexp.MustCompile("(?m)^\\s*`([^`]+)`").FindAllStringSubmatch(table[2], -1) {
			if column[1] != missing {
				rows.AddRow(table[1], column[1])
			}
		}
	}
	return rows
}

func assertInstalledSchema(t *testing.T, missing string) {
	t.Helper()
	source, err := os.ReadFile("../../../db/install.sql")
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(`SELECT table_name, column_name FROM information_schema.columns`).WillReturnRows(installColumnRows(source, missing))
	err = CheckSchema(db)
	if missing == "" && err != nil {
		t.Fatalf("installer columns must satisfy startup check: %v", err)
	}
	if missing != "" && err == nil {
		t.Fatal("incompatible schema accepted")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
