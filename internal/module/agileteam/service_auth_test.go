// =============================================================================
// 文件: internal/module/agileteam/service_auth_test.go
// 模块: 敏捷小组治理
// 类型: security regression
// 职责: 锁定敏捷小组写操作对象级授权，防止只靠粗粒度权限造成 IDOR。
// =============================================================================

package agileteam

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

func TestCanEditTeamgroupObject(t *testing.T) {
	row := &TeamgroupRow{ID: 7, PO: "po1", Manager: "sm1，sm2; sm3\nsm4"}
	cases := []struct {
		name        string
		actor       *model.User
		allowGlobal bool
		want        bool
	}{
		{name: "nil actor", actor: nil, want: false},
		{name: "po owns group", actor: &model.User{Account: "po1"}, want: true},
		{name: "first manager", actor: &model.User{Account: "sm1"}, want: true},
		{name: "fullwidth delimiter manager", actor: &model.User{Account: "sm2"}, want: true},
		{name: "semicolon manager", actor: &model.User{Account: "sm3"}, want: true},
		{name: "newline manager", actor: &model.User{Account: "sm4"}, want: true},
		{name: "ordinary member not enough", actor: &model.User{Account: "dev1"}, want: false},
		{name: "pmo global capability", actor: &model.User{Account: "pmo1"}, allowGlobal: true, want: true},
		{name: "super admin", actor: &model.User{Account: "admin", IsSuperAdmin: true}, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := canEditTeamgroupObject(tc.actor, row, tc.allowGlobal); got != tc.want {
				t.Fatalf("canEditTeamgroupObject()=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestRequireTeamgroupObjectEditRejectsCrossGroupWrite(t *testing.T) {
	err := requireTeamgroupObjectEdit(
		&model.User{Account: "po-other"},
		&TeamgroupRow{ID: 9, PO: "po-owner", Manager: "sm-owner"},
		false,
	)
	if err == nil {
		t.Fatal("expected forbidden for unrelated writer")
	}
	biz, ok := errorx.IsBizError(err)
	if !ok || biz.Code != "forbidden" {
		t.Fatalf("want forbidden biz error, got %#v", err)
	}
}

func TestConfirmAndRejectRequireServiceCapability(t *testing.T) {
	svc, _ := newTestService(t)
	actor := &model.User{Account: "po1"}
	if err := svc.ConfirmAdjustment(context.Background(), actor, ConfirmReq{AdjustmentID: 1}, false); err == nil {
		t.Fatal("confirm without PMO capability must fail before database access")
	}
	if err := svc.RejectAdjustment(context.Background(), actor, RejectReq{AdjustmentID: 1, Reason: "x"}, false); err == nil {
		t.Fatal("reject without PMO capability must fail before database access")
	}
}

func TestBlankSuperAdminCannotBypassActorAuth(t *testing.T) {
	svc, mock := newTestService(t)
	blank := &model.User{Account: "   ", IsSuperAdmin: true}
	if err := svc.ConfirmAdjustment(context.Background(), blank, ConfirmReq{AdjustmentID: 1}, true); err == nil {
		t.Fatal("blank superadmin must not confirm")
	}
	if err := svc.RejectAdjustment(context.Background(), blank, RejectReq{AdjustmentID: 1, Reason: "x"}, true); err == nil {
		t.Fatal("blank superadmin must not reject")
	}
	if err := svc.UpdateBasicInfo(context.Background(), blank, UpdateBasicReq{ID: 1, Name: "x"}, true); err == nil {
		t.Fatal("blank superadmin must not update basic info")
	}
	if _, err := svc.GetAdjustment(context.Background(), blank, 1, true); err == nil {
		t.Fatal("blank superadmin must not read adjustment")
	}
	if _, err := svc.SearchCandidates(context.Background(), blank, CandidateSearchReq{Q: "a"}); err == nil {
		t.Fatal("blank superadmin must not search candidates")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("blank actor must fail before database access: %v", err)
	}
}

func TestGetAdjustmentRejectsUnrelatedUser(t *testing.T) {
	svc, mock := newTestService(t)
	now := time.Now()
	mock.ExpectQuery("(?s)SELECT \\* FROM `zt_wb_agileteam_adjustment`.*id = \\?.*LIMIT").
		WithArgs(int64(88), 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "teamgroupId", "adjustNo", "status", "reason",
			"submittedBy", "confirmedBy", "confirmedDate", "rejectedBy", "rejectedDate", "rejectReason",
			"createdBy", "createdDate", "updatedBy", "updatedDate", "deletedAt",
		}).AddRow(int64(88), uint(3), "ADJ-088", StatusPending, "扩编",
			"po1", "", nil, "", nil, "",
			"po1", now, "po1", now, nil))
	mock.ExpectQuery("(?s)FROM zt_teamgroup tg.*WHERE tg.id = \\?.*LIMIT 1").
		WithArgs(uint(3)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "parent", "parent_name", "type", "grade", "path", "PO", "manager",
			"slogan", "declaration", "logo", "status", "createdDate",
		}).AddRow(uint(3), "团队三", uint(0), "", "parent", 1, ",3,", "po1", "sm1", "", "", "", "enable", "2026-01-01"))

	_, err := svc.GetAdjustment(context.Background(), &model.User{Account: "po-other"}, 88, false)
	biz, ok := errorx.IsBizError(err)
	if !ok || biz.Code != "forbidden" {
		t.Fatalf("want forbidden for unrelated reader, got %#v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestValidateBasicWriteBoundary(t *testing.T) {
	valid := UpdateBasicReq{ID: 1, Name: "组织变革团队", Slogan: "稳", Declaration: "信条", Logo: "/static/logo.png"}
	if err := validateBasicWriteBoundary(valid); err != nil {
		t.Fatalf("valid basic request rejected: %v", err)
	}
	badLogo := valid
	badLogo.Logo = "javascript:alert(1)"
	if err := validateBasicWriteBoundary(badLogo); err == nil {
		t.Fatal("javascript logo must be rejected")
	}
}

func TestValidateAdjustmentWriteBoundary(t *testing.T) {
	valid := SubmitAdjustmentReq{
		TeamgroupID: 1,
		Reason:      "正常调整",
		Items: []AdjustItemReq{{
			Account: "dev1", ActionType: ActionAdd, Role: "研发", AvailableHours: 7,
		}},
	}
	if err := validateAdjustmentWriteBoundary(valid); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	badHours := valid
	badHours.Items = []AdjustItemReq{{Account: "dev1", ActionType: ActionAdd, Role: "研发", AvailableHours: 25}}
	if err := validateAdjustmentWriteBoundary(badHours); err == nil {
		t.Fatal("hours > 24 must be rejected")
	}

	badAccount := valid
	badAccount.Items = []AdjustItemReq{{Account: "dev\n1", ActionType: ActionAdd, Role: "研发", AvailableHours: 7}}
	if err := validateAdjustmentWriteBoundary(badAccount); err == nil {
		t.Fatal("control characters in account must be rejected")
	}
}
