// =============================================================================
// 文件: internal/module/po/service_detail_edit_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证业务需求编辑修改权限与锁定规则（对齐禅道原生生命周期）。
// =============================================================================

package po

import (
	"testing"

	"workbench/internal/model"
)

func TestDeriveDemandEditability(t *testing.T) {
	creator := &model.User{Account: "003030"}
	assignee := &model.User{Account: "002940"}
	other := &model.User{Account: "771277"}
	admin := &model.User{Account: "admin", IsSuperAdmin: true}

	cases := []struct {
		name          string
		actor         *model.User
		row           *DemandDetailRow
		reviewedCount int
		wantCanEdit   bool
		wantReasonSub string
	}{
		{
			name:          "wait_creator_zero_reviewed_can_edit",
			actor:         creator,
			row:           &DemandDetailRow{Status: "wait", CreatedBy: "003030", AssignedTo: "002940"},
			reviewedCount: 0,
			wantCanEdit:   true,
		},
		{
			name:          "wait_creator_has_reviewed_locked",
			actor:         creator,
			row:           &DemandDetailRow{Status: "wait", CreatedBy: "003030", AssignedTo: "002940"},
			reviewedCount: 1,
			wantCanEdit:   false,
			wantReasonSub: "已有评审人出具评审意见",
		},
		{
			name:          "wait_other_user_cannot_edit",
			actor:         other,
			row:           &DemandDetailRow{Status: "wait", CreatedBy: "003030", AssignedTo: "002940"},
			reviewedCount: 0,
			wantCanEdit:   false,
			wantReasonSub: "无编辑权限",
		},
		{
			name:          "wait_admin_zero_reviewed_can_edit",
			actor:         admin,
			row:           &DemandDetailRow{Status: "wait", CreatedBy: "003030", AssignedTo: "002940"},
			reviewedCount: 0,
			wantCanEdit:   true,
		},
		{
			name:          "draft_creator_can_edit",
			actor:         creator,
			row:           &DemandDetailRow{Status: "draft", CreatedBy: "003030", AssignedTo: "002940"},
			reviewedCount: 0,
			wantCanEdit:   true,
		},
		{
			name:          "draft_assignee_can_edit",
			actor:         assignee,
			row:           &DemandDetailRow{Status: "draft", CreatedBy: "003030", AssignedTo: "002940"},
			reviewedCount: 0,
			wantCanEdit:   true,
		},
		{
			name:          "refuse_creator_can_edit",
			actor:         creator,
			row:           &DemandDetailRow{Status: "refuse", CreatedBy: "003030", AssignedTo: "002940"},
			reviewedCount: 1,
			wantCanEdit:   true,
		},
		{
			name:          "active_status_cannot_edit",
			actor:         creator,
			row:           &DemandDetailRow{Status: "active", CreatedBy: "003030"},
			reviewedCount: 1,
			wantCanEdit:   false,
			wantReasonSub: "当前阶段状态不支持直接编辑修改",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			canEdit, reason := deriveDemandEditability(tc.actor, tc.row, tc.reviewedCount)
			if canEdit != tc.wantCanEdit {
				t.Fatalf("deriveDemandEditability() canEdit=%v, want %v (reason: %q)", canEdit, tc.wantCanEdit, reason)
			}
			if tc.wantReasonSub != "" && !containsSub(reason, tc.wantReasonSub) {
				t.Fatalf("deriveDemandEditability() reason=%q, want to contain %q", reason, tc.wantReasonSub)
			}
		})
	}
}

func containsSub(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || stringContains(s, sub)))
}

func stringContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
