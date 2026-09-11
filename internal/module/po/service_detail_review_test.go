// =============================================================================
// 文件: internal/module/po/service_detail_review_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证详情页待评审操作权限摘要。
// =============================================================================

package po

import (
	"testing"

	"workbench/internal/model"
)

func TestCanWithdrawReviewForDetail(t *testing.T) {
	cases := []struct {
		name  string
		actor *model.User
		row   *DemandDetailRow
		want  bool
	}{
		{
			name:  "creator_wait_status",
			actor: &model.User{Account: "003030"},
			row:   &DemandDetailRow{Status: "wait", CreatedBy: "003030"},
			want:  true,
		},
		{
			name:  "superadmin_wait_status",
			actor: &model.User{Account: "admin", IsSuperAdmin: true},
			row:   &DemandDetailRow{Status: "wait", CreatedBy: "003030"},
			want:  true,
		},
		{
			name:  "creator_non_wait_status",
			actor: &model.User{Account: "003030"},
			row:   &DemandDetailRow{Status: "draft", CreatedBy: "003030"},
			want:  false,
		},
		{
			name:  "other_user_wait_status",
			actor: &model.User{Account: "771277"},
			row:   &DemandDetailRow{Status: "wait", CreatedBy: "003030"},
			want:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := canWithdrawReviewForDetail(tc.actor, tc.row)
			if got != tc.want {
				t.Fatalf("canWithdrawReviewForDetail()=%v, want %v", got, tc.want)
			}
		})
	}
}
