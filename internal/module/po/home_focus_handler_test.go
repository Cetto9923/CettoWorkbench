// =============================================================================
// 文件: internal/module/po/home_focus_handler_test.go
// 模块: PO 工作台
// 职责: 新办理人口径与评审预查询、关键词占位回归。
// =============================================================================
package po

import (
	"strings"
	"testing"
)

func TestDevelopingHandlerSQLFollowsReportRule(t *testing.T) {
	if strings.Contains(developingHandlerSQL, "developFinish") {
		t.Fatal("提测不看日期")
	}
	if !strings.Contains(developingHandlerSQL, "IFNULL(BRA, '') = '' AND assignedTo = ?") {
		t.Fatal("必须 BRA 优先")
	}
}
func TestDevelopingHandlerSQLWiredIntoEveryMyActionWhere(t *testing.T) {
	for _, reviews := range [][]int{nil, {}, {101, 102}} {
		where, args := currentHandlerDemandWhereWithReviews("alice", reviews)
		if strings.Count(where, "?") != len(args) {
			t.Fatal("占位参数不一致")
		}
		for _, want := range []string{developingHandlerSQL, acceptanceOwnerSQL, demandTestHandlerSQL, latestAcceptanceActorSQL, "createdBy = ? OR"} {
			if !strings.Contains(where, want) {
				t.Fatalf("missing %s", want)
			}
		}
		for _, forbidden := range []string{"zt_demandclarify", "waitdeliver", "developFinish"} {
			if strings.Contains(where, forbidden) {
				t.Fatalf("obsolete %s", forbidden)
			}
		}
	}
}
func TestDevelopingKeywordWhereFollowsHandlerRule(t *testing.T) {
	where, args := currentHandlerDemandWhere("alice")
	keyword := currentHandlerDemandKeywordWhere()
	if strings.Count(keyword, "?") != len(args) || strings.Count(where, "?") != len(args) {
		t.Fatal("keyword argument mismatch")
	}
	if strings.Contains(keyword, "developFinish") || strings.Contains(keyword, "PM") {
		t.Fatal("obsolete keyword rule")
	}
}
