// =============================================================================
// 文件: internal/module/po/servicefollow_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证 Service.Demands 的 objectType=story / objectType=demand 工具栏筛选
//       在 status=all, focus=all（首页默认）和 stage=schedule/acceptanced 两条
//       路径下都生效。Bug 复现：listAllStageDemands / listMySQLDemands 不读
//       req.ObjectType，把对象类型当装饰而不当作过滤条件，导致 Story 按钮
//       选中后仍然混入业务需求。
// 依赖: github.com/DATA-DOG/go-sqlmock, gorm.io/gorm
// =============================================================================

package po

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"go.uber.org/zap"

	"workbench/internal/model"
)

// TestDemands_StoryOnly_NoDemandSQL 验证 status=all & focus=all（首页默认
// “全部”）+ objectType=story 时：listAllStageDemands 不会触达 zt_demand。
//
// 实现要点：sqlmock 仅声明故事相关查询；任何对 zt_demand 表的 SQL 都会触发
// “unexpected query” 错误。业需候选集一旦被并进 FindAllStageRefsPaged 的
// UNION ALL，参数个数与内容都会偏离下面的断言，测试立即失败。
func TestDemands_StoryOnly_NoDemandSQL(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	// 只预期 zt_story 相关查询；zt_demand 一旦被命中，sqlmock 会报错。
	//
	// FindAllStageRefsPaged 的计数查询：schedule(stage_index=3) 与
	// acceptanced(stage_index=7) 两段 zt_story 子查询 UNION ALL 后包一层
	// count(*)。参数按占位符出现顺序断言，业需一旦被并进来参数个数与内容都会
	// 变化，测试立即失败。
	mock.ExpectQuery("(?s)^SELECT count\\(\\*\\) FROM \\(SELECT kind, id, MIN\\(stage_index\\).*`zt_story`.*UNION ALL.*`zt_story`.*\\) AS all_stages$").
		WithArgs(
			3, "0", "demandpool", "story", "alice",
			7, "0", "demandpool", "story", "alice", sqlmock.AnyArg(),
		).
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(2))

	// FindAllStageRefsPaged 的取数查询：同一个 UNION ALL 候选集 + 排序分页，
	// 因此参数与 count 查询完全一致，末尾多一个 LIMIT 占位符。
	mock.ExpectQuery("(?s)^SELECT id, kind, stage_index FROM \\(SELECT kind, id, MIN\\(stage_index\\).*`zt_story`.*UNION ALL.*`zt_story`.*\\) AS all_stages ORDER BY stage_index ASC, kind_rank ASC, id DESC LIMIT \\?$").
		WithArgs(
			3, "0", "demandpool", "story", "alice",
			7, "0", "demandpool", "story", "alice", sqlmock.AnyArg(),
			15,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "stage_index"}).
			AddRow(101, "story", 3).
			AddRow(202, "story", 7))

	// populateWorkItems → FindStoriesByIDs：SELECT `id`, `title`, `pri`, `status` FROM zt_story ...
	mock.ExpectQuery("SELECT .* FROM `zt_story` WHERE id IN .*").
		WithArgs(101, 202).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "pri", "status"}).
			AddRow(101, "Story A", 1, "developing").
			AddRow(202, "Story B", 2, "developing"))

	// DeriveStoryPrimaryActions → CountStoryTestTasks（单 SQL，子查询先读 zt_story id IN ?）
	mock.ExpectQuery("(?s)SELECT s\\.id AS story.*FROM zt_story").
		WithArgs(101, 202, 101, 202).
		WillReturnRows(sqlmock.NewRows([]string{"story", "count", "first_id"}).
			AddRow(101, 0, 0).
			AddRow(202, 0, 0))

	// DeriveStoryPrimaryActions → FindStoryMetaForAction
	mock.ExpectQuery("SELECT id, status, stage FROM `zt_story`").
		WithArgs(101, 202, "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "stage"}).
			AddRow(101, "developing", "").
			AddRow(202, "developing", ""))

	resp, err := svc.Demands(context.Background(), &model.User{Account: "alice"}, DemandsReq{
		Status:     "all",
		Focus:      "all",
		ObjectType: "story",
		Page:       1,
		PageSize:   15,
	})
	if err != nil {
		t.Fatalf("Demands: %v", err)
	}
	// 防止 mock 全部返回空导致 kind 断言空转。
	if len(resp.Items) != 2 || resp.Total != 2 {
		t.Fatalf("expect 2 story items, got items=%d total=%d", len(resp.Items), resp.Total)
	}
	for _, item := range resp.Items {
		if item.Kind != "story" {
			t.Fatalf("objectType=story returned kind=%q item=%+v", item.Kind, item)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expects unmet (likely a stray zt_demand query): %v", err)
	}
}

// TestListMySQLDemands_ScheduleDemandOnly 验证 stage=schedule + objectType=demand
// 时 listMySQLDemands 只加载业需，不查 zt_story。
func TestListMySQLDemands_ScheduleDemandOnly(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	scheduleFilter := mysqlStageFilters["schedule"]

	// 只预期 FindRoleDemandIDs，不预期 FindScheduleStoryIDs / FindDeliverStoryIDs
	mock.ExpectQuery("SELECT .* FROM `zt_demand`").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9001))

	// populateWorkItems → FindRoleDemandsByIDs
	mock.ExpectQuery("SELECT .* FROM `zt_demand`").
		WithArgs(9001).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "pri", "status", "hang", "assignedTo", "QD", "RD", "BRA", "pm"}).
			AddRow(9001, "Dem X", 1, "clarified", "0", "alice", "", "", "", ""))

	// DeriveDemandPrimaryActions 的 SQL 链
	mock.ExpectQuery("SELECT .*FROM zt_demand WHERE id IN .*deleted = .*").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "stage", "status", "assignedTo", "accepter"}).
			AddRow(9001, "", "clarified", "alice", ""))

	mock.ExpectQuery("(?s)SELECT d\\.id AS demand_id.*FROM zt_demand.*fromDemand IN").
		WithArgs(9001, 9001).
		WillReturnRows(sqlmock.NewRows([]string{"demand_id", "count", "first_id"}).AddRow(9001, 0, 0))

	mock.ExpectQuery("(?s)SELECT d\\.id AS demand_id.*zt_demandappraise").
		WithArgs("alice", 9001).
		WillReturnRows(sqlmock.NewRows([]string{"demand_id", "has_pending", "has_any"}).AddRow(9001, false, false))

	// enrichDemandCanReview：当前 status="clarified"，不是 wait，所以不应触发查询。
	_, err := svc.listMySQLDemands(context.Background(), &model.User{Account: "alice"}, "schedule", scheduleFilter, DemandsReq{
		Status:     "schedule",
		ObjectType: "demand",
		Page:       1,
		PageSize:   15,
	}, map[string]string{})
	if err != nil {
		t.Fatalf("listMySQLDemands: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expects unmet (likely a stray zt_story query): %v", err)
	}
}

// TestListMySQLDemands_ScheduleStoryOnly 验证 stage=schedule + objectType=story
// 时 listMySQLDemands 只加载研需，不查业需。这是 bug 修复的单元目标。
func TestListMySQLDemands_ScheduleStoryOnly(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	scheduleFilter := mysqlStageFilters["schedule"]

	// 只预期 FindScheduleStoryIDs，不预期 FindRoleDemandIDs
	mock.ExpectQuery("SELECT .* FROM `zt_story`").
		WithArgs("0", "demandpool", "story", "alice").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(501))

	// populateWorkItems → FindStoriesByIDs
	mock.ExpectQuery("SELECT .* FROM `zt_story` WHERE id IN .*").
		WithArgs(501).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "pri", "status"}).
			AddRow(501, "Story X", 2, "clarified"))

	// DeriveStoryPrimaryActions
	mock.ExpectQuery("(?s)SELECT s\\.id AS story.*FROM zt_story").
		WithArgs(501, 501).
		WillReturnRows(sqlmock.NewRows([]string{"story", "count", "first_id"}).AddRow(501, 0, 0))

	mock.ExpectQuery("SELECT id, status, stage FROM `zt_story`").
		WithArgs(501, "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "stage"}).AddRow(501, "clarified", ""))

	_, err := svc.listMySQLDemands(context.Background(), &model.User{Account: "alice"}, "schedule", scheduleFilter, DemandsReq{
		Status:     "schedule",
		ObjectType: "story",
		Page:       1,
		PageSize:   15,
	}, map[string]string{})
	if err != nil {
		t.Fatalf("listMySQLDemands: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expects unmet: %v", err)
	}
}

// TestListMySQLDemands_DevelopingStoryOnlyReturnsEmpty 验证纯业需阶段
// （developing / testing 等没有研需的阶段）+ objectType=story 时直接返回空，
// 不触发任何 SQL。bug 修复前会走 CountRoleDemands / FindRoleDemandsPaged。
func TestListMySQLDemands_DevelopingStoryOnlyReturnsEmpty(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	devFilter := mysqlStageFilters["developing"]

	// 不设置任何 mock.ExpectQuery：修复后应直接返回 Empty，无 SQL 调用。
	resp, err := svc.listMySQLDemands(context.Background(), &model.User{Account: "alice"}, "developing", devFilter, DemandsReq{
		Status:     "developing",
		ObjectType: "story",
		Page:       1,
		PageSize:   15,
	}, map[string]string{})
	if err != nil {
		t.Fatalf("listMySQLDemands: %v", err)
	}
	if len(resp.Items) != 0 || resp.Total != 0 {
		t.Fatalf("objectType=story on demand-only stage must return empty, got items=%d total=%d", len(resp.Items), resp.Total)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected SQL queries were issued: %v", err)
	}
}
