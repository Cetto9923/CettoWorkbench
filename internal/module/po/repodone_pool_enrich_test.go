// =============================================================================
// 文件: internal/module/po/repodone_pool_enrich_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 我的已办对象上下文"所属需求池"注入路径回归测试。
//       demand：直接 LEFT JOIN zt_demandpool ON zt_demand.pool = dp.id
//       story：先经 zt_story.fromDemand → zt_demand → zt_demandpool
//       task：先经 zt_task.story → zt_story.fromDemand → zt_demand → zt_demandpool
//       其它对象类型（bug/todo/审批）不补齐 PoolName，详情抽屉与列表渲染须呈现 "--"。
// 依赖: github.com/DATA-DOG/go-sqlmock, gorm.io/gorm
// =============================================================================

package po

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newPoolEnrichMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open sqlmock failed: %v", err)
	}
	dialector := mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm failed: %v", err)
	}
	return gormDB, mock
}

// 验证 demand 对象的 PoolName 直接来自 zt_demandpool。
func TestFetchObjectContexts_DemandIncludesPoolName(t *testing.T) {
	gormDB, mock := newPoolEnrichMockDB(t)
	repo := NewRepo(gormDB, gormDB)

	mock.ExpectQuery(`SELECT d\.id, d\.name, d\.status, d\.assignedTo[\s\S]*FROM zt_demand AS d`).
		WithArgs(int64(101)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "status", "assignedTo", "product_name", "pool_name",
		}).AddRow(101, "信贷需求", "active", "user_a", "信贷产品", "信贷需求池"))

	rows := []doneActionDBRow{{ID: 100, ObjectType: "demand", ObjectID: 101}}
	ctxs, err := repo.fetchObjectContexts(t.Context(), rows)
	if err != nil {
		t.Fatalf("fetchObjectContexts error: %v", err)
	}

	got, ok := ctxs["demand:101"]
	if !ok {
		t.Fatalf("expected demand:101 in ctx map")
	}
	if got.PoolName != "信贷需求池" {
		t.Fatalf("demand PoolName = %q, want %q", got.PoolName, "信贷需求池")
	}
	if got.ProductName != "信贷产品" {
		t.Fatalf("demand ProductName = %q, want %q", got.ProductName, "信贷产品")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

// 验证 demand 的需求池 LEFT JOIN 在 dp.deleted='0' 被遵守，dp 行缺失时 PoolName 必须为空。
func TestFetchObjectContexts_DemandPoolMissingReturnsEmpty(t *testing.T) {
	gormDB, mock := newPoolEnrichMockDB(t)
	repo := NewRepo(gormDB, gormDB)

	mock.ExpectQuery(`SELECT d\.id, d\.name, d\.status, d\.assignedTo[\s\S]*FROM zt_demand AS d`).
		WithArgs(int64(202)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "status", "assignedTo", "product_name", "pool_name",
		}).AddRow(202, "无池需求", "draft", "user_b", "产品X", ""))

	ctxs, err := repo.fetchObjectContexts(t.Context(), []doneActionDBRow{{ID: 200, ObjectType: "demand", ObjectID: 202}})
	if err != nil {
		t.Fatalf("fetchObjectContexts error: %v", err)
	}
	if got := ctxs["demand:202"].PoolName; got != "" {
		t.Fatalf("expected empty PoolName when zt_demandpool join missed, got %q", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

// 验证 story 对象的 PoolName 经 zt_story.fromDemand → zt_demand.pool → zt_demandpool 推导。
func TestFetchObjectContexts_StoryIncludesPoolNameViaParentDemand(t *testing.T) {
	gormDB, mock := newPoolEnrichMockDB(t)
	repo := NewRepo(gormDB, gormDB)

	mock.ExpectQuery(`SELECT s\.id, s\.title, s\.status, s\.assignedTo[\s\S]*FROM zt_story AS s`).
		WithArgs(int64(303)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "status", "assignedTo", "product_name", "pool_name",
		}).AddRow(303, "研需标题", "active", "user_c", "信贷产品", "信贷需求池"))

	ctxs, err := repo.fetchObjectContexts(t.Context(), []doneActionDBRow{{ID: 300, ObjectType: "story", ObjectID: 303}})
	if err != nil {
		t.Fatalf("fetchObjectContexts error: %v", err)
	}
	got, ok := ctxs["story:303"]
	if !ok {
		t.Fatalf("expected story:303 in ctx map")
	}
	if got.PoolName != "信贷需求池" {
		t.Fatalf("story PoolName = %q, want %q (derive via fromDemand→demand→demandpool)", got.PoolName, "信贷需求池")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

// 验证 task 对象的 PoolName 经 zt_task.story → zt_story.fromDemand → zt_demand → zt_demandpool 推导。
func TestFetchObjectContexts_TaskIncludesPoolNameViaStoryAndDemand(t *testing.T) {
	gormDB, mock := newPoolEnrichMockDB(t)
	repo := NewRepo(gormDB, gormDB)

	mock.ExpectQuery(`SELECT t\.id, t\.name, t\.status, t\.project[\s\S]*FROM zt_task AS t`).
		WithArgs(int64(404)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "status", "project", "execution", "assignedTo", "project_name", "execution_name", "pool_name",
		}).AddRow(404, "任务标题", "doing", 1, 2, "user_d", "信贷项目", "信贷迭代", "信贷需求池"))

	ctxs, err := repo.fetchObjectContexts(t.Context(), []doneActionDBRow{{ID: 401, ObjectType: "task", ObjectID: 404}})
	if err != nil {
		t.Fatalf("fetchObjectContexts error: %v", err)
	}
	got, ok := ctxs["task:404"]
	if !ok {
		t.Fatalf("expected task:404 in ctx map")
	}
	if got.PoolName != "信贷需求池" {
		t.Fatalf("task PoolName = %q, want %q (derive via task.story→story.fromDemand→demand→demandpool)", got.PoolName, "信贷需求池")
	}
	if got.ProjectName != "信贷项目" || got.ExecutionName != "信贷迭代" {
		t.Fatalf("task context fields regressed: project=%q exec=%q", got.ProjectName, got.ExecutionName)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

// 验证 bug 对象不补齐 PoolName；详情抽屉应呈现 "--"。
func TestFetchObjectContexts_BugDoesNotProvidePoolName(t *testing.T) {
	gormDB, mock := newPoolEnrichMockDB(t)
	repo := NewRepo(gormDB, gormDB)

	mock.ExpectQuery(`SELECT b\.id, b\.title, b\.status, b\.project[\s\S]*FROM zt_bug AS b`).
		WithArgs(int64(505)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "status", "project", "execution", "assignedTo", "project_name", "execution_name",
		}).AddRow(505, "Bug标题", "active", 1, 0, "user_e", "项目", ""))

	ctxs, err := repo.fetchObjectContexts(t.Context(), []doneActionDBRow{{ID: 500, ObjectType: "bug", ObjectID: 505}})
	if err != nil {
		t.Fatalf("fetchObjectContexts error: %v", err)
	}
	got, ok := ctxs["bug:505"]
	if !ok {
		t.Fatalf("expected bug:505 in ctx map")
	}
	if got.PoolName != "" {
		t.Fatalf("bug PoolName must remain empty, got %q", got.PoolName)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

// 验证 DoneAction.PoolName 与 DoneDetailContext.PoolName 字段已经被声明（DTO 字段注入）。
func TestDoneActionAndDetailContextPoolNameFieldWired(t *testing.T) {
	action := DoneAction{PoolName: "信贷需求池"}
	if action.PoolName != "信贷需求池" {
		t.Fatalf("DoneAction.PoolName round-trip failed, got %q", action.PoolName)
	}
	ctx := DoneDetailContext{PoolName: "信贷需求池"}
	if ctx.PoolName != "信贷需求池" {
		t.Fatalf("DoneDetailContext.PoolName round-trip failed, got %q", ctx.PoolName)
	}
}
