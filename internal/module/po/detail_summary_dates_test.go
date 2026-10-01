// =============================================================================
// 文件: internal/module/po/detail_summary_dates_test.go
// 模块: 需求详情
// 类型: test
// 职责: 验证数据库零日期不会作为实际日期显示，正常日期仍按原格式展示。
// =============================================================================

package po

import (
	"testing"
	"time"
)

func TestDemandSummaryZeroDatesAreUnset(t *testing.T) {
	zero := time.Time{}
	date := time.Date(2026, 9, 18, 10, 30, 0, 0, time.UTC)
	svc := &DetailService{}
	row := &DemandDetailRow{CreatedDate: &date, EditedDate: &zero,
		EstimateLaunch: &zero, DevelopFinish: &zero, TestFinish: &zero, VerifyFinish: &zero}
	summary := svc.buildSummary(row)
	for _, value := range []string{summary.EstimateLaunch, summary.DevelopFinish, summary.TestFinish, summary.VerifyFinish} {
		if value != "—" {
			t.Fatalf("unset planned date displayed as %q", value)
		}
	}
	if summary.CreatedDate != "2026-09-18 10:30" || summary.EditedDate != summary.CreatedDate {
		t.Fatalf("created date or existing edit fallback changed: %+v", summary)
	}
	row.EstimateLaunch, row.EditedDate = &date, &date
	summary = svc.buildSummary(row)
	if summary.EstimateLaunch != "2026-09-18" || summary.EditedDate != "2026-09-18 10:30" {
		t.Fatalf("valid dates lost their original display format: %+v", summary)
	}
}
