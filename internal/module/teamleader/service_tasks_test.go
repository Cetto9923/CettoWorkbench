// =============================================================================
// 文件: internal/module/teamleader/service_tasks_test.go
// 模块: 团队长工作台
// 类型: test
// 职责: 验证小组研发工作看板权限隔离、任务真实归属与待核查任务分组计算。
// =============================================================================

package teamleader

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

// 1. 验证未登录或未传参数拦截
func TestListGroupTasks_Validation(t *testing.T) {
	db, _ := newMockDB(t)
	svc := NewService(NewRepo(db))

	_, err := svc.ListGroupTasks(context.Background(), nil, ListGroupTasksReq{})
	if err == nil {
		t.Fatal("expected error for nil actor, got nil")
	}

	actor := &model.User{Account: "u1"}
	_, err = svc.ListGroupTasks(context.Background(), actor, ListGroupTasksReq{TeamID: 0, GroupID: 10})
	if err == nil {
		t.Fatal("expected validation error for zero teamId, got nil")
	}
}

// 2. 验证越权访问非授权团队返回 403
func TestListGroupTasks_ForbiddenTeam(t *testing.T) {
	db, mock := newMockDB(t)
	svc := NewService(NewRepo(db))

	actor := &model.User{Account: "leader01", IsSuperAdmin: false}

	// mock resolveAuthorizedTeams：当前用户仅属于团队 1
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs("parent", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(1, "parent", "信贷团队", "leader01", "po01", 0, 1, "enable").
			AddRow(2, "parent", "运营团队", "leader02", "po02", 0, 1, "enable"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT p.id FROM zt_teamgroup p")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	// 试图查询未授权的团队 2
	_, err := svc.ListGroupTasks(context.Background(), actor, ListGroupTasksReq{
		TeamID:  2,
		GroupID: 10,
	})
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) || bizErr.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected ErrCodeForbidden, got: %v", err)
	}
}

// 3. 验证小组正式任务与待核查候选任务正确分组与统计
func TestListGroupTasks_ConfirmedAndPendingTasks(t *testing.T) {
	db, mock := newMockDB(t)
	svc := NewService(NewRepo(db))

	actor := &model.User{Account: "leader01", IsSuperAdmin: false}

	// 1. resolveAuthorizedTeams
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs("parent", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(1, "parent", "信贷团队", "leader01", "po01", 0, 1, "enable"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT p.id FROM zt_teamgroup p")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	// 2. FindSubGroupsByParentID (团队 1 下有小组 11)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE parent = ? AND type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs(uint(1), "child", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(11, "child", "对公1组", "coach01", "po01", 1, 1, "enable"))

	// 3. FindMembersByGroupIDs (小组 11 成员)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT root AS group_id, account, COALESCE(role, '') AS role FROM `zt_team` WHERE type = ? AND root IN (?) ORDER BY root ASC, account ASC")).
		WithArgs("teamgroup", uint(11)).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "account", "role"}).
			AddRow(11, "dev01", "研发").
			AddRow(11, "dev02", "测试"))

	// 4. FindGroupTasksByDemand (查询需求归属为小组 11 的正式任务)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT t.id, t.name, t.type, t.status, t.story, t.assignedTo, t.finishedBy, t.deadline,")).
		WithArgs("11", 200).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "type", "status", "story", "assignedTo", "finishedBy", "deadline", "storyTitle", "demandTeamGroup"}).
			AddRow(101, "对公开发任务A", "devel", "doing", 501, "dev01", "", time.Now().Add(24*time.Hour), "对公需求A", "11").
			AddRow(102, "对公测试任务B", "test", "wait", 501, "dev02", "", time.Now().Add(-24*time.Hour), "对公需求A", "11"))

	// 5. FindPendingTasksByAccounts (查询成员 dev01, dev02 名下无明确需求的任务)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT t.id, t.name, t.type, t.status, t.story, t.assignedTo, t.finishedBy, t.deadline,")).
		WithArgs("dev01", "dev02", 200).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "type", "status", "story", "assignedTo", "finishedBy", "deadline", "storyTitle", "demandTeamGroup"}).
			AddRow(201, "慢SQL优化", "misc", "doing", 0, "dev01", "", nil, "", ""))

	// 6. ResolveRealnames
	mock.ExpectQuery(regexp.QuoteMeta("SELECT account, realname FROM `zt_user`")).
		WillReturnRows(sqlmock.NewRows([]string{"account", "realname"}).
			AddRow("dev01", "张开发").
			AddRow("dev02", "李测试"))

	resp, err := svc.ListGroupTasks(context.Background(), actor, ListGroupTasksReq{
		TeamID:  1,
		GroupID: 11,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GroupID != 11 || resp.GroupName != "对公1组" {
		t.Fatalf("unexpected group info: %v", resp)
	}

	// 验证正式指标：正式任务 2 条（1 doing, 1 wait），逾期 1 条（任务 102 截止时间在昨天）
	if resp.Summary.ConfirmedTotal != 2 {
		t.Errorf("expected confirmedTotal 2, got %d", resp.Summary.ConfirmedTotal)
	}
	if resp.Summary.ConfirmedDoing != 1 {
		t.Errorf("expected confirmedDoing 1, got %d", resp.Summary.ConfirmedDoing)
	}
	if resp.Summary.ConfirmedWait != 1 {
		t.Errorf("expected confirmedWait 1, got %d", resp.Summary.ConfirmedWait)
	}
	if resp.Summary.ConfirmedOverdue != 1 {
		t.Errorf("expected confirmedOverdue 1, got %d", resp.Summary.ConfirmedOverdue)
	}

	// 验证待核查任务指标：1 条，且待核查任务绝不计入 ConfirmedTotal！
	if resp.Summary.PendingReviewTotal != 1 {
		t.Errorf("expected pendingReviewTotal 1, got %d", resp.Summary.PendingReviewTotal)
	}
	if len(resp.PendingReviewTasks) != 1 {
		t.Fatalf("expected 1 pending review task, got %d", len(resp.PendingReviewTasks))
	}
	if !resp.PendingReviewTasks[0].IsPendingReview {
		t.Errorf("expected IsPendingReview to be true")
	}
	if resp.PendingReviewTasks[0].Title != "慢SQL优化" {
		t.Errorf("unexpected task title: %s", resp.PendingReviewTasks[0].Title)
	}
}
