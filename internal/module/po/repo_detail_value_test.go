// =============================================================================
// 文件: internal/module/po/repo_detail_value_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证价值模型总开关默认打开（无 enabled 项即开启）与显式配置覆盖规则。
// =============================================================================

package po

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// expectDemandValueConfig 铺设 LoadDemandValueConfig 的两条查询预期：
// demandvalue 配置项 + noAiCategory 名单。
func expectDemandValueConfig(mock sqlmock.Sqlmock, rows *sqlmock.Rows) {
	mock.ExpectQuery(`SELECT[\s\S]*FROM.*zt_config.*WHERE owner = \? AND module = \? AND section = \?`).
		WithArgs("system", "demand", "demandvalue").
		WillReturnRows(rows)
	mock.ExpectQuery(`SELECT[\s\S]*FROM.*zt_config.*WHERE owner = \? AND module = \? AND section = \? AND.*key.*= \?`).
		WithArgs("system", "custom", "clarifyCategoryAIConfig", "noAiCategory").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`["datachange"]`))
}

// 1. 缺省：无 enabled 项 → 总开关开启（默认打开）。
func TestLoadDemandValueConfig_EnabledDefaultsOn(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)

	expectDemandValueConfig(mock, sqlmock.NewRows([]string{"key", "value"}).
		AddRow("costPerMonth", "21750").
		AddRow("intervalMethod", "fixed"))

	cfg, err := repo.LoadDemandValueConfig(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Enabled {
		t.Fatal("缺省无 enabled 项时总开关应默认为开启")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足的 SQL 预期: %v", err)
	}
}

// 2. enabled = "1" → 开启。
func TestLoadDemandValueConfig_EnabledOn(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)

	expectDemandValueConfig(mock, sqlmock.NewRows([]string{"key", "value"}).
		AddRow("enabled", "1"))

	cfg, err := repo.LoadDemandValueConfig(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Enabled {
		t.Fatal(`enabled = "1" 时总开关应开启`)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足的 SQL 预期: %v", err)
	}
}

// 3. enabled = "0" → 关闭。
func TestLoadDemandValueConfig_EnabledOff(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)

	expectDemandValueConfig(mock, sqlmock.NewRows([]string{"key", "value"}).
		AddRow("enabled", "0"))

	cfg, err := repo.LoadDemandValueConfig(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Enabled {
		t.Fatal(`enabled = "0" 时总开关应关闭`)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足的 SQL 预期: %v", err)
	}
}
