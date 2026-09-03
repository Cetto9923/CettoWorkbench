// =============================================================================
// 文件: internal/module/po/repodone_test.go
// 模块: PO 工作台
// 类型: action
// 职责: 验证我的已办正式动作注册口径。
// 依赖: 无
// =============================================================================

package po

import (
	"strings"
	"testing"
)

func TestFormalDoneActionsExcludeNoise(t *testing.T) {
	for _, key := range []string{"user:login", "demand:edit", "task:comment"} {
		if _, ok := formalDoneActions[key]; ok {
			t.Fatalf("noise action %q must not enter formal done", key)
		}
	}
	for _, key := range []string{"demand:clarify", "task:finished", "bug:resolved"} {
		if _, ok := formalDoneActions[key]; !ok {
			t.Fatalf("formal action %q missing", key)
		}
	}
}

func TestDoneListReqAllowsApprovalTab(t *testing.T) {
	req := DoneListReq{Tab: DoneTabApproval}
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("approval tab should be valid: %#v", errs)
	}
	if req.Tab != DoneTabApproval {
		t.Fatalf("unexpected tab after validation: %q", req.Tab)
	}
}

func TestApprovalDoneScopeContainsOnlyReviewActions(t *testing.T) {
	scope := buildApprovalDoneScopeSQL()
	for _, action := range []string{"reviewed", "reviewpassed", "reviewrejected"} {
		if !strings.Contains(scope, action) {
			t.Fatalf("approval scope missing %q", action)
		}
	}
	if strings.Contains(scope, "demand:edit") {
		t.Fatal("approval scope must not include ordinary edits")
	}
}
