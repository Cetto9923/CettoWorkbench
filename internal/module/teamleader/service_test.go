// =============================================================================
// 文件: internal/module/teamleader/service_test.go
// 模块: 团队长工作台
// 类型: test
// 职责: 验证权限隔离、多团队身份、跨组成员双重去重等核心业务逻辑。
// =============================================================================

package teamleader

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})
	gdb, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      conn,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return gdb, mock
}

// 1. 验证未登录账号直接拦截
func TestGetTeamHierarchy_Unauthenticated(t *testing.T) {
	db, _ := newMockDB(t)
	svc := NewService(NewRepo(db))

	_, err := svc.GetTeamHierarchy(context.Background(), nil, TeamHierarchyReq{})
	if err == nil {
		t.Fatal("expected error for nil actor, got nil")
	}
}

// 2. 验证越权访问未授权团队返回 403
func TestGetTeamHierarchy_ForbiddenTeam(t *testing.T) {
	db, mock := newMockDB(t)
	svc := NewService(NewRepo(db))

	actor := &model.User{Account: "user01", IsSuperAdmin: false}

	// 模拟查询所有父团队
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs("parent", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(1, "parent", "信贷团队", "leader01", "po01", 0, 1, "enable").
			AddRow(2, "parent", "运营团队", "leader02", "po02", 0, 1, "enable"))

	// 模拟查询当前用户授权团队，只属于团队 1
	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT p.id FROM zt_teamgroup p")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	// 用户尝试强行访问团队 2
	_, err := svc.GetTeamHierarchy(context.Background(), actor, TeamHierarchyReq{TeamID: 2})
	if err == nil {
		t.Fatal("expected forbidden error for unauthorized teamId, got nil")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) || bizErr.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected ErrCodeForbidden, got %v", err)
	}
}

// 3. 验证无关联团队返回空状态而非报错
func TestGetTeamHierarchy_NoAssociatedTeam(t *testing.T) {
	db, mock := newMockDB(t)
	svc := NewService(NewRepo(db))

	actor := &model.User{Account: "outsider", IsSuperAdmin: false}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs("parent", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(1, "parent", "信贷团队", "leader01", "po01", 0, 1, "enable"))

	// 用户不属于任何团队
	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT p.id FROM zt_teamgroup p")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	resp, err := svc.GetTeamHierarchy(context.Background(), actor, TeamHierarchyReq{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CurrentTeam != nil {
		t.Fatalf("expected nil CurrentTeam for no associated team, got %+v", resp.CurrentTeam)
	}
	if len(resp.AuthorizedTeams) != 0 {
		t.Fatalf("expected empty AuthorizedTeams, got %d", len(resp.AuthorizedTeams))
	}
	if resp.MyRole != "guest" {
		t.Fatalf("expected role guest, got %s", resp.MyRole)
	}
}

// 4. 验证双重集合去重与跨组成员识别
func TestGetTeamHierarchy_DeduplicationAndCrossGroup(t *testing.T) {
	db, mock := newMockDB(t)
	svc := NewService(NewRepo(db))

	actor := &model.User{Account: "leader01", IsSuperAdmin: false}

	// 1. 父团队列表
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs("parent", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(1, "parent", "信贷团队", "leader01", "po01", 0, 1, "enable"))

	// 2. 授权团队列表（命中团队 1）
	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT p.id FROM zt_teamgroup p")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	// 3. 查询团队 1 详情
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE id = ? AND type = ? AND deleted = ? LIMIT ?")).
		WithArgs(1, "parent", "0", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(1, "parent", "信贷团队", "leader01", "po01", 0, 1, "enable"))

	// 4. 查询团队 1 下属子小组（2 个小组）
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE parent = ? AND type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs(1, "child", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(11, "child", "对公1组", "coach11", "po11", 1, 2, "enable").
			AddRow(12, "child", "对公2组", "coach12", "po12", 1, 2, "enable"))

	// 5. 批量查询子小组成员（注意：member01 同时在 11 组和 12 组；且 11 组有重复写入记录）
	mock.ExpectQuery(regexp.QuoteMeta("SELECT root AS group_id, account, COALESCE(role, '') AS role FROM `zt_team` WHERE type = ? AND root IN (?,?) ORDER BY root ASC, account ASC")).
		WithArgs("teamgroup", 11, 12).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "account", "role"}).
			AddRow(11, "member01", "dev").
			AddRow(11, "member01", "dev"). // 重复行，应被 (groupId, account) 过滤
			AddRow(11, "member02", "qa").
			AddRow(12, "member01", "dev"). // 跨组行
			AddRow(12, "member03", "dev"))

	// 6. 解析真实姓名
	mock.ExpectQuery(regexp.QuoteMeta("SELECT account, realname FROM `zt_user` WHERE account IN (?,?,?,?,?,?,?,?,?) AND deleted = ?")).
		WillReturnRows(sqlmock.NewRows([]string{"account", "realname"}).
			AddRow("leader01", "张团队长").
			AddRow("po01", "李总监").
			AddRow("coach11", "王组长1").
			AddRow("po11", "赵产品1").
			AddRow("coach12", "吴组长2").
			AddRow("po12", "郑产品2").
			AddRow("member01", "钱跨组").
			AddRow("member02", "孙测试").
			AddRow("member03", "周开发"))

	resp, err := svc.GetTeamHierarchy(context.Background(), actor, TeamHierarchyReq{TeamID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证团队总人数去重：一共 3 个独立成员 (member01, member02, member03)，不能为 4
	if resp.CurrentTeam.TotalUniqueMembers != 3 {
		t.Fatalf("expected TotalUniqueMembers=3, got %d", resp.CurrentTeam.TotalUniqueMembers)
	}

	// 验证子小组数量
	if len(resp.SubGroups) != 2 {
		t.Fatalf("expected 2 subGroups, got %d", len(resp.SubGroups))
	}

	// 验证 11 组有效成员数 (member01 去重后 + member02 = 2 人)
	if resp.SubGroups[0].MemberCount != 2 {
		t.Fatalf("expected 11 group MemberCount=2, got %d", resp.SubGroups[0].MemberCount)
	}

	// 验证 member01 跨组标记
	m1 := resp.SubGroups[0].Members[0]
	if !m1.IsMultiGroup {
		t.Fatalf("expected member01 IsMultiGroup=true, got false")
	}
	if len(m1.OtherGroupNames) != 1 || m1.OtherGroupNames[0] != "对公2组" {
		t.Fatalf("expected otherGroupNames [对公2组], got %v", m1.OtherGroupNames)
	}

	// 验证 member02 未跨组
	m2 := resp.SubGroups[0].Members[1]
	if m2.IsMultiGroup {
		t.Fatalf("expected member02 IsMultiGroup=false, got true")
	}

	// 验证敏捷教练与产品经理正确映射自 zt_teamgroup 的 manager 与 PO
	if resp.SubGroups[0].ScrumMaster.Name != "王组长1" {
		t.Fatalf("expected ScrumMaster '王组长1', got %s", resp.SubGroups[0].ScrumMaster.Name)
	}
	if resp.SubGroups[0].PO.Name != "赵产品1" {
		t.Fatalf("expected PO '赵产品1', got %s", resp.SubGroups[0].PO.Name)
	}

	// 验证当前角色为团队长
	if resp.MyRole != "team_leader" {
		t.Fatalf("expected role team_leader, got %s", resp.MyRole)
	}
}

// 5. 验证普通组长/组员不可越权查看非所属小组的成员明细
func TestGetTeamHierarchy_SubgroupDetailPermissionMasking(t *testing.T) {
	db, mock := newMockDB(t)
	svc := NewService(NewRepo(db))

	// coach11 仅为 11 组组长，未参与 12 组，且非团队长
	actor := &model.User{Account: "coach11", IsSuperAdmin: false}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs("parent", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(1, "parent", "信贷团队", "leader01", "po01", 0, 1, "enable"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT p.id FROM zt_teamgroup p")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE id = ? AND type = ? AND deleted = ? LIMIT ?")).
		WithArgs(1, "parent", "0", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(1, "parent", "信贷团队", "leader01", "po01", 0, 1, "enable"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, name, manager, PO, parent, grade, status FROM `zt_teamgroup` WHERE parent = ? AND type = ? AND deleted = ? ORDER BY id ASC")).
		WithArgs(1, "child", "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "name", "manager", "PO", "parent", "grade", "status"}).
			AddRow(11, "child", "对公1组", "coach11", "po11", 1, 2, "enable").
			AddRow(12, "child", "对公2组", "coach12", "po12", 1, 2, "enable"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT root AS group_id, account, COALESCE(role, '') AS role FROM `zt_team` WHERE type = ? AND root IN (?,?) ORDER BY root ASC, account ASC")).
		WithArgs("teamgroup", 11, 12).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "account", "role"}).
			AddRow(11, "member01", "dev").
			AddRow(12, "member03", "dev"))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT account, realname FROM `zt_user` WHERE account IN (?,?,?,?,?,?,?,?) AND deleted = ?")).
		WillReturnRows(sqlmock.NewRows([]string{"account", "realname"}).
			AddRow("leader01", "张团队长").
			AddRow("po01", "李总监").
			AddRow("coach11", "王组长1").
			AddRow("po11", "赵产品1").
			AddRow("coach12", "吴组长2").
			AddRow("po12", "郑产品2").
			AddRow("member01", "钱开发").
			AddRow("member03", "周开发"))

	resp, err := svc.GetTeamHierarchy(context.Background(), actor, TeamHierarchyReq{TeamID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.MyRole != "group_leader" {
		t.Fatalf("expected role group_leader, got %s", resp.MyRole)
	}

	// 11 组是其管辖组，应该能看到 11 组成员明细
	if len(resp.SubGroups[0].Members) != 1 {
		t.Fatalf("expected group 11 members visible (1), got %d", len(resp.SubGroups[0].Members))
	}

	// 12 组非其管辖组，成员明细必须被屏蔽清空，但人数统计 MemberCount 依然准确保留
	if resp.SubGroups[1].MemberCount != 1 {
		t.Fatalf("expected group 12 MemberCount=1, got %d", resp.SubGroups[1].MemberCount)
	}
	if len(resp.SubGroups[1].Members) != 0 {
		t.Fatalf("expected group 12 members masked to empty, got %d", len(resp.SubGroups[1].Members))
	}
}

