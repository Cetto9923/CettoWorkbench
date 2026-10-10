// =============================================================================
// 文件: internal/module/teamleader/service_tasks_test.go
// 模块: 团队长工作台
// 类型: test
// 职责: 验证小组研发工作看板对象级权限隔离（团队长/小组长/普通成员/外部人员）、
//       成员筛选鉴权、真实统计与分页对账、待核查任务隔离。
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

// 1. 验证未登录或未传必填参数拦截
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

	_, err = svc.ListGroupTasks(context.Background(), actor, ListGroupTasksReq{TeamID: 1, GroupID: 0})
	if err == nil {
		t.Fatal("expected validation error for zero groupId, got nil")
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

// 3. 负向测试：普通成员越权访问兄弟小组返回 403 Forbidden
func TestListGroupTasks_NormalMemberCannotAccessSiblingGroup(t *testing.T) {
	db, mock := newMockDB(t)
	svc := NewService(NewRepo(db))

	// actor dev01 是小组 11 的普通成员，非团队长，非小组长
	actor := &model.User{Account: "dev01", IsSuperAdmin: false}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs("parent", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(1, "parent", "信贷团队", "leader01", "po01", 0, 1, "enable"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT p.id FROM zt_teamgroup p")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	// FindParentTeamByID
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE id = ? AND type = ? AND deleted = ? LIMIT ?")).
		WithArgs(uint(1), "parent", "0", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(1, "parent", "信贷团队", "leader01", "po01", 0, 1, "enable"))

	// FindSubGroupsByParentID
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE parent = ? AND type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs(uint(1), "child", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(11, "child", "对公1组", "manager11", "po11", 1, 1, "enable").
			AddRow(12, "child", "对公2组(兄弟组)", "manager12", "po12", 1, 1, "enable"))

	// FindMembersByGroupIDs for Group 12 (兄弟组中只有 dev02, 没有 dev01)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT root AS group_id, account, COALESCE(role, '') AS role FROM `zt_team` WHERE type = ? AND root IN (?) ORDER BY root ASC, account ASC")).
		WithArgs("teamgroup", uint(12)).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "account", "role"}).
			AddRow(12, "dev02", "研发"))

	_, err := svc.ListGroupTasks(context.Background(), actor, ListGroupTasksReq{
		TeamID:  1,
		GroupID: 12, // 尝试访问兄弟组
	})
	if err == nil {
		t.Fatal("expected forbidden error for accessing sibling group, got nil")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) || bizErr.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected ErrCodeForbidden, got: %v", err)
	}
}

// 4. 负向测试：小组长越权访问未负责的兄弟小组返回 403 Forbidden
func TestListGroupTasks_GroupLeaderCannotAccessSiblingGroup(t *testing.T) {
	db, mock := newMockDB(t)
	svc := NewService(NewRepo(db))

	// manager11 只是小组 11 的组长，非团队长
	actor := &model.User{Account: "manager11", IsSuperAdmin: false}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs("parent", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(1, "parent", "信贷团队", "leader01", "po01", 0, 1, "enable"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT p.id FROM zt_teamgroup p")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE id = ? AND type = ? AND deleted = ? LIMIT ?")).
		WithArgs(uint(1), "parent", "0", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(1, "parent", "信贷团队", "leader01", "po01", 0, 1, "enable"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE parent = ? AND type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs(uint(1), "child", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(11, "child", "对公1组", "manager11", "po11", 1, 1, "enable").
			AddRow(12, "child", "对公2组(兄弟组)", "manager12", "po12", 1, 1, "enable"))

	// 小组 12 成员只有 dev02
	mock.ExpectQuery(regexp.QuoteMeta("SELECT root AS group_id, account, COALESCE(role, '') AS role FROM `zt_team` WHERE type = ? AND root IN (?) ORDER BY root ASC, account ASC")).
		WithArgs("teamgroup", uint(12)).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "account", "role"}).
			AddRow(12, "dev02", "研发"))

	_, err := svc.ListGroupTasks(context.Background(), actor, ListGroupTasksReq{
		TeamID:  1,
		GroupID: 12,
	})
	if err == nil {
		t.Fatal("expected forbidden error for group leader accessing sibling group, got nil")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) || bizErr.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected ErrCodeForbidden, got: %v", err)
	}
}

// 5. 负向测试：普通成员试图查询组内其他成员个人任务返回 403 Forbidden
func TestListGroupTasks_NormalMemberCannotQueryOtherMember(t *testing.T) {
	db, mock := newMockDB(t)
	svc := NewService(NewRepo(db))

	actor := &model.User{Account: "dev01", IsSuperAdmin: false}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs("parent", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(1, "parent", "信贷团队", "leader01", "po01", 0, 1, "enable"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT p.id FROM zt_teamgroup p")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE id = ? AND type = ? AND deleted = ? LIMIT ?")).
		WithArgs(uint(1), "parent", "0", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(1, "parent", "信贷团队", "leader01", "po01", 0, 1, "enable"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE parent = ? AND type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs(uint(1), "child", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(11, "child", "对公1组", "manager11", "po11", 1, 1, "enable"))

	// 小组 11 成员包含 dev01 与 dev02
	mock.ExpectQuery(regexp.QuoteMeta("SELECT root AS group_id, account, COALESCE(role, '') AS role FROM `zt_team` WHERE type = ? AND root IN (?) ORDER BY root ASC, account ASC")).
		WithArgs("teamgroup", uint(11)).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "account", "role"}).
			AddRow(11, "dev01", "研发").
			AddRow(11, "dev02", "测试"))

	// dev01 试图通过 account=dev02 探测他人个人任务
	_, err := svc.ListGroupTasks(context.Background(), actor, ListGroupTasksReq{
		TeamID:  1,
		GroupID: 11,
		Account: "dev02",
	})
	if err == nil {
		t.Fatal("expected forbidden error when member queries other member, got nil")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) || bizErr.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected ErrCodeForbidden, got: %v", err)
	}
}

// 6. 负向测试：传入组外未知账号返回 403 Forbidden
func TestListGroupTasks_QueryNonMemberAccountForbidden(t *testing.T) {
	db, mock := newMockDB(t)
	svc := NewService(NewRepo(db))

	// leader01 是团队长，但在小组 11 筛选时传入了不属于该小组的外部人员 attacker
	actor := &model.User{Account: "leader01", IsSuperAdmin: false}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs("parent", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(1, "parent", "信贷团队", "leader01", "po01", 0, 1, "enable"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT p.id FROM zt_teamgroup p")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE id = ? AND type = ? AND deleted = ? LIMIT ?")).
		WithArgs(uint(1), "parent", "0", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(1, "parent", "信贷团队", "leader01", "po01", 0, 1, "enable"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE parent = ? AND type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs(uint(1), "child", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(11, "child", "对公1组", "manager11", "po11", 1, 1, "enable"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT root AS group_id, account, COALESCE(role, '') AS role FROM `zt_team` WHERE type = ? AND root IN (?) ORDER BY root ASC, account ASC")).
		WithArgs("teamgroup", uint(11)).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "account", "role"}).
			AddRow(11, "dev01", "研发"))

	_, err := svc.ListGroupTasks(context.Background(), actor, ListGroupTasksReq{
		TeamID:  1,
		GroupID: 11,
		Account: "external_attacker",
	})
	if err == nil {
		t.Fatal("expected forbidden error for external account, got nil")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) || bizErr.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected ErrCodeForbidden, got: %v", err)
	}
}

// 7. 正向测试：团队长查看小组研发任务（真实统计对账、分页隔离与时间范围口径）
func TestListGroupTasks_ConfirmedAndPendingTasks(t *testing.T) {
	db, mock := newMockDB(t)
	svc := NewService(NewRepo(db))

	actor := &model.User{Account: "leader01", IsSuperAdmin: false}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs("parent", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(1, "parent", "信贷团队", "leader01", "po01", 0, 1, "enable"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT p.id FROM zt_teamgroup p")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE id = ? AND type = ? AND deleted = ? LIMIT ?")).
		WithArgs(uint(1), "parent", "0", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(1, "parent", "信贷团队", "leader01", "po01", 0, 1, "enable"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE parent = ? AND type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs(uint(1), "child", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(11, "child", "对公1组", "coach01", "po01", 1, 1, "enable"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT root AS group_id, account, COALESCE(role, '') AS role FROM `zt_team` WHERE type = ? AND root IN (?) ORDER BY root ASC, account ASC")).
		WithArgs("teamgroup", uint(11)).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "account", "role"}).
			AddRow(11, "dev01", "研发").
			AddRow(11, "dev02", "测试"))

	// CountGroupConfirmedTasks
	mock.ExpectQuery(regexp.QuoteMeta("COUNT(CASE WHEN t.status = 'wait' THEN 1 END) AS wait_count")).
		WillReturnRows(sqlmock.NewRows([]string{"wait_count", "doing_count", "done_count", "overdue_count"}).
			AddRow(1, 1, 2, 1))

	// CountPendingTasksByAccounts
	mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM zt_task AS t")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// FindGroupConfirmedTasksPaged
	mock.ExpectQuery(regexp.QuoteMeta("SELECT t.id, t.name, t.type, t.status, t.story, t.assignedTo, t.finishedBy, t.deadline, t.finishedDate,")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "type", "status", "story", "assignedTo", "finishedBy", "deadline", "finishedDate", "storyTitle", "demandTeamGroup"}).
			AddRow(101, "对公开发任务A", "devel", "doing", 501, "dev01", "", time.Now().Add(24*time.Hour), nil, "对公需求A", "11").
			AddRow(102, "对公测试任务B", "test", "wait", 501, "dev02", "", time.Now().Add(-24*time.Hour), nil, "对公需求A", "11"))

	// FindPendingTasksByAccountsPaged
	mock.ExpectQuery(regexp.QuoteMeta("SELECT t.id, t.name, t.type, t.status, t.story, t.assignedTo, t.finishedBy, t.deadline, t.finishedDate,")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "type", "status", "story", "assignedTo", "finishedBy", "deadline", "finishedDate", "storyTitle", "demandTeamGroup"}).
			AddRow(201, "慢SQL优化", "misc", "doing", 0, "dev01", "", nil, nil, "", ""))

	// ResolveRealnames
	mock.ExpectQuery(regexp.QuoteMeta("SELECT account, realname FROM `zt_user`")).
		WillReturnRows(sqlmock.NewRows([]string{"account", "realname"}).
			AddRow("dev01", "张开发").
			AddRow("dev02", "李测试"))

	resp, err := svc.ListGroupTasks(context.Background(), actor, ListGroupTasksReq{
		TeamID:   1,
		GroupID:  11,
		Page:     1,
		PageSize: 50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GroupID != 11 || resp.GroupName != "对公1组" {
		t.Fatalf("unexpected group info: %v", resp)
	}

	// 真实统计对账：wait 1, doing 1, done 2 => ConfirmedTotal = 4
	if resp.Summary.ConfirmedTotal != 4 {
		t.Errorf("expected confirmedTotal 4, got %d", resp.Summary.ConfirmedTotal)
	}
	if resp.Summary.ConfirmedWait != 1 {
		t.Errorf("expected confirmedWait 1, got %d", resp.Summary.ConfirmedWait)
	}
	if resp.Summary.ConfirmedDoing != 1 {
		t.Errorf("expected confirmedDoing 1, got %d", resp.Summary.ConfirmedDoing)
	}
	if resp.Summary.ConfirmedDone != 2 {
		t.Errorf("expected confirmedDone 2, got %d", resp.Summary.ConfirmedDone)
	}
	if resp.Summary.ConfirmedOverdue != 1 {
		t.Errorf("expected confirmedOverdue 1, got %d", resp.Summary.ConfirmedOverdue)
	}

	// 待核查候选任务总数 1，且待核查任务不计入 ConfirmedTotal
	if resp.Summary.PendingReviewTotal != 1 {
		t.Errorf("expected pendingReviewTotal 1, got %d", resp.Summary.PendingReviewTotal)
	}
	if resp.ConfirmedPagination.Total != 4 {
		t.Errorf("expected ConfirmedPagination.Total 4, got %d", resp.ConfirmedPagination.Total)
	}
	if resp.PendingPagination.Total != 1 {
		t.Errorf("expected PendingPagination.Total 1, got %d", resp.PendingPagination.Total)
	}
	if resp.TimeRangeLabel == "" {
		t.Errorf("expected non-empty time range label")
	}
}
