// =============================================================================
// 文件: internal/module/po/service_detail_review_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证详情页待评审操作权限摘要。
// =============================================================================

package po

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"go.uber.org/zap"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
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


func TestCanSubmitDemandReview(t *testing.T) {
	cases := []struct {
		name      string
		actor     *model.User
		createdBy string
		want      bool
	}{
		{
			name:      "creator_can_submit",
			actor:     &model.User{Account: "003030", IsSuperAdmin: false},
			createdBy: "003030",
			want:      true,
		},
		{
			name:      "superadmin_can_submit",
			actor:     &model.User{Account: "admin", IsSuperAdmin: true},
			createdBy: "003030",
			want:      true,
		},
		{
			name:      "ordinary_assignee_cannot_submit",
			actor:     &model.User{Account: "002940", IsSuperAdmin: false},
			createdBy: "003030",
			want:      false,
		},
		{
			name:      "nil_actor_cannot_submit",
			actor:     nil,
			createdBy: "003030",
			want:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := canSubmitDemandReview(tc.actor, tc.createdBy)
			if got != tc.want {
				t.Fatalf("canSubmitDemandReview()=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestSubmitReview_ServiceAuthorization_ThreeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("ordinary_assignee_cannot_submit", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := NewRepo(gormDB, gormDB)
		svc := NewService(repo, nil, nil, zap.NewNop())

		mock.ExpectQuery("SELECT id, status, deleted, createdBy, assignedTo, reviewedBy, reviewer, mailto, isNeedFocus, product FROM `zt_demand` WHERE id = ?").
			WithArgs(int64(100), 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "status", "deleted", "createdBy", "assignedTo", "reviewedBy", "reviewer", "mailto", "isNeedFocus", "product"}).
				AddRow(100, "draft", "0", "user_creator", "user_assignee", "", "", "", "0", 1))

		actor := &model.User{Account: "user_assignee", IsSuperAdmin: false}
		_, err := svc.GetDemandReviewCandidates(ctx, actor, 100)
		if err == nil {
			t.Fatal("ordinary assignee must NOT be allowed to get review candidates")
		}
		if biz, ok := errorx.IsBizError(err); !ok || biz.Code != errorx.ErrCodeForbidden {
			t.Fatalf("expected ErrCodeForbidden, got: %v", err)
		}

		mock.ExpectQuery("SELECT id, status, deleted, createdBy, assignedTo, reviewedBy, reviewer, mailto, isNeedFocus, product FROM `zt_demand` WHERE id = ?").
			WithArgs(int64(100), 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "status", "deleted", "createdBy", "assignedTo", "reviewedBy", "reviewer", "mailto", "isNeedFocus", "product"}).
				AddRow(100, "draft", "0", "user_creator", "user_assignee", "", "", "", "0", 1))

		err = svc.SubmitDemandReview(ctx, actor, SubmitDemandReviewReq{ID: 100, Reviewer: []string{"reviewer1"}})
		if err == nil {
			t.Fatal("ordinary assignee must NOT be allowed to submit demand review")
		}
		if biz, ok := errorx.IsBizError(err); !ok || biz.Code != errorx.ErrCodeForbidden {
			t.Fatalf("expected ErrCodeForbidden, got: %v", err)
		}
	})
}
