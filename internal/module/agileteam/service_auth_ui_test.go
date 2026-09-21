// =============================================================================
// 文件: internal/module/agileteam/service_auth_ui_test.go
// 模块: 敏捷小组治理
// 类型: security regression
// 职责: Detail CanEdit 必须与写接口对象级授权一致。
// =============================================================================

package agileteam

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"workbench/internal/model"
)

func TestCanEditTeamgroupObjectRequiresCoarsePermission(t *testing.T) {
	svc, mock := newTestService(t)
	got, err := svc.CanEditTeamgroupObject(
		context.Background(), &model.User{Account: "po1"}, 7, false, false,
	)
	if err != nil {
		t.Fatalf("object edit check: %v", err)
	}
	if got {
		t.Fatal("coarse permission=false must keep detail read-only")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected database access: %v", err)
	}
}

func TestCanEditTeamgroupObjectRejectsUnrelatedUser(t *testing.T) {
	svc, mock := newTestService(t)
	mock.ExpectQuery("(?s)FROM zt_teamgroup tg.*WHERE tg.id = \\?.*LIMIT 1").
		WithArgs(uint(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "parent", "parent_name", "type", "grade", "path", "PO", "manager",
			"slogan", "declaration", "logo", "status", "createdDate",
		}).AddRow(uint(7), "团队七", uint(0), "", "parent", 1, ",7,", "po-owner", "sm-owner", "", "", "", "enable", "2026-01-01"))

	got, err := svc.CanEditTeamgroupObject(
		context.Background(), &model.User{Account: "po-other"}, 7, true, false,
	)
	if err != nil {
		t.Fatalf("object edit check: %v", err)
	}
	if got {
		t.Fatal("unrelated workbench user must not receive CanEdit=true")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
