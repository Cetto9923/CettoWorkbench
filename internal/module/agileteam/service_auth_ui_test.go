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

func TestCanViewTeamgroupLeadScopeAllowsManagedChildOnly(t *testing.T) {
	svc, mock := newTestService(t)
	mock.ExpectQuery("(?s)FROM zt_teamgroup managed.*REGEXP \\?").
		WithArgs("(^|[[:space:],;])coach1([[:space:],;]|$)").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(11)))
	allowed, err := svc.CanViewTeamgroupLeadScope(context.Background(), &model.User{Account: "coach1"}, 11)
	if err != nil || !allowed {
		t.Fatalf("managed child access = (%v, %v), want (true, nil)", allowed, err)
	}

	mock.ExpectQuery("(?s)FROM zt_teamgroup managed.*REGEXP \\?").
		WithArgs("(^|[[:space:],;])coach1([[:space:],;]|$)").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(11)))
	mock.ExpectQuery("(?s)SELECT d\\.id, COALESCE\\(d\\.path, ''\\) AS path FROM zt_dept d.*LEFT JOIN zt_wb_dept_manager_override.*REGEXP \\?").
		WithArgs(append(techHQArgs(), "(^|[[:space:],;])coach1([[:space:],;]|$)")...).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}))
	allowed, err = svc.CanViewTeamgroupLeadScope(context.Background(), &model.User{Account: "coach1"}, 12)
	if err != nil || allowed {
		t.Fatalf("sibling access = (%v, %v), want (false, nil)", allowed, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}

func TestCanEnterLeadViewRequiresCoachOrDeptManager(t *testing.T) {
	svc, mock := newTestService(t)
	mock.ExpectQuery("(?s)FROM zt_teamgroup managed.*REGEXP \\?").
		WithArgs("(^|[[:space:],;])coach1([[:space:],;]|$)").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(11)))
	allowed, err := svc.CanEnterLeadView(context.Background(), &model.User{Account: "coach1"})
	if err != nil || !allowed {
		t.Fatalf("coach access = (%v, %v), want (true, nil)", allowed, err)
	}

	mock.ExpectQuery("(?s)FROM zt_teamgroup managed.*REGEXP \\?").
		WithArgs("(^|[[:space:],;])member1([[:space:],;]|$)").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery("(?s)SELECT d\\.id, COALESCE\\(d\\.path, ''\\) AS path FROM zt_dept d.*LEFT JOIN zt_wb_dept_manager_override.*REGEXP \\?").
		WithArgs(append(techHQArgs(), "(^|[[:space:],;])member1([[:space:],;]|$)")...).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}))
	allowed, err = svc.CanEnterLeadView(context.Background(), &model.User{Account: "member1"})
	if err != nil || allowed {
		t.Fatalf("member access = (%v, %v), want (false, nil)", allowed, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}

func TestCanViewTeamgroupMemberDetailsDoesNotInferFromDepartmentManagement(t *testing.T) {
	svc, mock := newTestService(t)
	mock.ExpectQuery("(?s)SELECT d\\.id, COALESCE\\(d\\.path, ''\\) AS path FROM zt_dept d.*LEFT JOIN zt_wb_dept_manager_override.*REGEXP \\?").
		WithArgs(append(techHQArgs(), "(^|[[:space:],;])manager1([[:space:],;]|$)")...).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}).AddRow(uint(20), ",1,20,"))
	mock.ExpectQuery("(?s)SELECT DISTINCT id FROM zt_dept WHERE id IN \\(\\?\\) OR path LIKE \\?").
		WithArgs(uint(20), ",1,20,%").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(20)))
	allowed, err := svc.CanViewTeamgroupMemberDetails(context.Background(), &model.User{Account: "manager1"}, 11)
	if err != nil || allowed {
		t.Fatalf("department manager details = (%v, %v), want (false, nil)", allowed, err)
	}

	mock.ExpectQuery("(?s)SELECT d\\.id, COALESCE\\(d\\.path, ''\\) AS path FROM zt_dept d.*LEFT JOIN zt_wb_dept_manager_override.*REGEXP \\?").
		WithArgs(append(techHQArgs(), "(^|[[:space:],;])coach1([[:space:],;]|$)")...).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}))
	mock.ExpectQuery("(?s)FROM zt_teamgroup managed.*REGEXP \\?").
		WithArgs("(^|[[:space:],;])coach1([[:space:],;]|$)").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(11)))
	allowed, err = svc.CanViewTeamgroupMemberDetails(context.Background(), &model.User{Account: "coach1"}, 11)
	if err != nil || !allowed {
		t.Fatalf("coach details = (%v, %v), want (true, nil)", allowed, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestRedactTeamgroupMemberDetailsKeepsOnlyAggregateCounts(t *testing.T) {
	resp := &DetailResp{
		FormalCount: 3,
		Formal:      []MemberItem{{Account: "alice", Name: "Alice"}},
		PendingJoin: []MemberItem{{Account: "bob", Name: "Bob"}},
		History:     []HistoryItem{{Actor: "alice", ActorName: "Alice"}},
		Pending:     &PendingSummary{SubmittedBy: "Alice", AddCount: 1, AddNames: []string{"Bob"}},
	}
	redactTeamgroupMemberDetails(resp)
	if resp.FormalCount != 3 || len(resp.Formal) != 0 || len(resp.PendingJoin) != 0 || len(resp.History) != 0 {
		t.Fatalf("personal data not redacted or aggregate count lost: %+v", resp)
	}
	if resp.Pending.SubmittedBy != "" || len(resp.Pending.AddNames) != 0 || resp.Pending.AddCount != 1 {
		t.Fatalf("pending details not redacted or aggregate count lost: %+v", resp.Pending)
	}
}
