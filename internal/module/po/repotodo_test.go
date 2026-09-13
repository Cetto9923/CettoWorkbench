// =============================================================================
// 文件: internal/module/po/repotodo_test.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的待办数据转换回归测试。
// 依赖: 无
// =============================================================================

package po

import (
	"testing"
	"time"
)

func TestFormatTodoDeadline(t *testing.T) {
	zero := time.Time{}
	if got := formatTodoDeadline(&zero); got != "" {
		t.Fatalf("zero deadline = %q, want empty", got)
	}

	deadline := time.Date(2026, time.September, 3, 0, 0, 0, 0, time.Local)
	if got := formatTodoDeadline(&deadline); got != "2026-09-03" {
		t.Fatalf("deadline = %q, want 2026-09-03", got)
	}
}

func TestIssueRiskPriLabel(t *testing.T) {
	// zt_issue/zt_risk.pri 为 char(30)，混存数字串与 low/middle/high/urgent，须统一映射到 P1..P4。
	cases := map[string]string{
		"1": "P1", "urgent": "P1", "immediate": "P1", "URGENT": "P1",
		"2": "P2", "high": "P2",
		"3": "P3", "middle": "P3", "medium": "P3",
		"4": "P4", "low": "P4",
		"": "", "5": "", "  high ": "P2",
	}
	for in, want := range cases {
		if got := issueRiskPriLabel(in); got != want {
			t.Fatalf("issueRiskPriLabel(%q) = %q, want %q", in, got, want)
		}
	}
}
