// =============================================================================
// 文件: internal/module/po/detail_created_name_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 回归详情"创建人"必须返回 中文名(工号)，而不是裸工号。
// =============================================================================

package po

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestBuildSummaryCreatedNameUsesRealnameWithAccount(t *testing.T) {
	svc := &DetailService{}

	row := &DemandDetailRow{ID: 1723, CreatedBy: "004481", CreatedByName: "胡昶"}
	summary := svc.buildSummary(row)
	if summary.CreatedName != "胡昶(004481)" {
		t.Fatalf("CreatedName = %q, want %q", summary.CreatedName, "胡昶(004481)")
	}
	// 原始工号仍需保留，前端的 isCreator 等判定依赖它。
	if summary.CreatedBy != "004481" {
		t.Fatalf("CreatedBy = %q, want raw account 004481", summary.CreatedBy)
	}

	// zt_user 查不到 realname 时回退裸工号，不能返回空串让前端显示 "—"。
	noName := svc.buildSummary(&DemandDetailRow{ID: 1, CreatedBy: "004481"})
	if noName.CreatedName != "004481" {
		t.Fatalf("CreatedName without realname = %q, want 004481", noName.CreatedName)
	}

	// realname 已自带工号后缀时不得重复拼接。
	dup := svc.buildSummary(&DemandDetailRow{ID: 2, CreatedBy: "003030", CreatedByName: "程统(003030)"})
	if dup.CreatedName != "程统(003030)" {
		t.Fatalf("CreatedName = %q, want no duplicated account suffix", dup.CreatedName)
	}

	// createdBy 为空（历史数据）时保持空，交给前端回退 "—"。
	empty := svc.buildSummary(&DemandDetailRow{ID: 3})
	if empty.CreatedName != "" {
		t.Fatalf("CreatedName for empty account = %q, want empty", empty.CreatedName)
	}
}

func TestFormatModuleNameUsesChineseEmptyState(t *testing.T) {
	for _, value := range []string{"", "0", "—"} {
		if got := formatModuleName(value); got != "未设置" {
			t.Fatalf("formatModuleName(%q) = %q, want 未设置", value, got)
		}
	}
	if got := formatModuleName("客户管理模块"); got != "客户管理模块" {
		t.Fatalf("formatModuleName() = %q, want original module name", got)
	}
}

func TestBuildSummaryUsesActionCreatedDateWhenAvailable(t *testing.T) {
	svc := &DetailService{}
	createdDate := time.Date(2025, 7, 11, 0, 0, 0, 0, time.Local)
	actionDate := time.Date(2025, 7, 11, 14, 35, 22, 0, time.Local)

	row := &DemandDetailRow{
		ID:                63316,
		CreatedDate:       &createdDate,
		ActionCreatedDate: &actionDate,
	}

	summary := svc.buildSummary(row)
	if summary.CreatedDate != "2025-07-11 14:35" {
		t.Fatalf("CreatedDate = %q, want %q", summary.CreatedDate, "2025-07-11 14:35")
	}
	if summary.EditedDate != "2025-07-11 14:35" {
		t.Fatalf("EditedDate = %q, want %q", summary.EditedDate, "2025-07-11 14:35")
	}
}

func TestBuildSummaryFallsBackToCreatedDateWhenActionDateNil(t *testing.T) {
	svc := &DetailService{}
	createdDate := time.Date(2025, 7, 11, 10, 20, 0, 0, time.Local)

	row := &DemandDetailRow{
		ID:          63316,
		CreatedDate: &createdDate,
	}

	summary := svc.buildSummary(row)
	if summary.CreatedDate != "2025-07-11 10:20" {
		t.Fatalf("CreatedDate = %q, want %q", summary.CreatedDate, "2025-07-11 10:20")
	}
}

func TestFindDemandCreationActionDate_QueriesZtAction(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)

	actionTime := time.Date(2025, 7, 11, 16, 42, 10, 0, time.Local)
	mock.ExpectQuery(`SELECT a\.date FROM zt_action a WHERE a\.objectType = 'demand' AND a\.objectID = \?`).
		WithArgs(uint(63316)).
		WillReturnRows(sqlmock.NewRows([]string{"date"}).AddRow(actionTime))

	actDate, err := repo.FindDemandCreationActionDate(t.Context(), 63316)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actDate == nil || *actDate != actionTime {
		t.Fatalf("actDate = %v, want %v", actDate, actionTime)
	}
}
