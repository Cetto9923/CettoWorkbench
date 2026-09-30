// =============================================================================
// 文件: internal/module/po/repo_primaryaction_rel_sqlmock_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 用 sqlmock 覆盖关系批量的非超管路径，证明整批只发固定条数 SQL（无 N+1）。
// =============================================================================

package po

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"workbench/internal/model"
)

// TestBatchDemandRelations_LeaderPath 覆盖「非超管、有管辖」路径：
// 期望固定 3 条 SQL（个人关系批量 / ListLeaderDeptTreeIDs / 管辖批量），
// 与批内需求条数无关，证明没有 N+1。
func TestBatchDemandRelations_LeaderPath(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	ids := []uint{1, 2, 3, 4, 5}

	// 1. 个人干系人批量：一条 IN 查询覆盖整批
	//    （actor.ID 为 0 时 IsPMORole 会被跳过，故此处不发该 SQL）
	mock.ExpectQuery(`(?s)SELECT d\.id FROM zt_demand d.*d\.id IN \(.*d\.originator`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2))

	// 2. 管辖部门树：ListLeaderDeptTreeIDs 内部两条
	//    2a. 扫出管辖/下级部门
	mock.ExpectQuery(`(?s)SELECT d\.id, COALESCE\(d\.path, ''\) AS path FROM zt_dept`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}).AddRow(14, ",14,"))
	//    2b. 展开部门树
	mock.ExpectQuery(`(?s)SELECT DISTINCT .id. FROM .zt_dept. WHERE id IN`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(14))

	// 3. 管辖批量：一条 IN 查询
	mock.ExpectQuery(`(?s)SELECT DISTINCT d\.id FROM zt_demand d.*u\.dept IN \(.*u\.account = d\.originator`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(3))

	batch, err := repo.BatchDemandRelations(t.Context(),
		&model.User{Account: "leader_a", IsSuperAdmin: false}, ids)
	if err != nil {
		t.Fatalf("BatchDemandRelations 失败: %v", err)
	}

	for _, want := range []uint{1, 2, 3} {
		if !batch.HasRelated(want) {
			t.Fatalf("需求 %d 应判定为有关系", want)
		}
	}
	for _, unwanted := range []uint{4, 5} {
		if batch.HasRelated(unwanted) {
			t.Fatalf("需求 %d 应判定为无关系", unwanted)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足的 SQL 预期（说明发出的 SQL 条数或形态与设计不符）: %v", err)
	}
}

// TestBatchDemandRelations_PMORole 覆盖 PMO 角色：判定为全量有关系后立即返回，不再往下查。
func TestBatchDemandRelations_PMORole(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	ids := []uint{1, 2}

	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\) FROM zt_gf_user_roles.*r\.code = \?`).
		WithArgs(int64(15864), "pmo").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	batch, err := repo.BatchDemandRelations(t.Context(),
		&model.User{ID: 15864, Account: "pmo_a", IsSuperAdmin: false}, ids)
	if err != nil {
		t.Fatalf("BatchDemandRelations 失败: %v", err)
	}
	for _, id := range ids {
		if !batch.HasRelated(id) {
			t.Fatalf("PMO 对需求 %d 应恒为有关系", id)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("PMO 分支不应再发出后续 SQL: %v", err)
	}
}

// TestBatchDemandRelations_SuperAdminShortCircuit 覆盖超管短路：0 条 SQL、全量有关系。
func TestBatchDemandRelations_SuperAdminShortCircuit(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	ids := []uint{7, 8, 9}

	batch, err := repo.BatchDemandRelations(t.Context(),
		&model.User{Account: "root", IsSuperAdmin: true}, ids)
	if err != nil {
		t.Fatalf("BatchDemandRelations 失败: %v", err)
	}
	for _, id := range ids {
		if !batch.HasRelated(id) {
			t.Fatalf("超管对需求 %d 应恒为有关系", id)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("超管短路不应发出任何 SQL，但有未预期调用: %v", err)
	}
}

// TestBatchDemandRelations_NilSafe 覆盖空批与 nil actor。
func TestBatchDemandRelations_NilSafe(t *testing.T) {
	gormDB, _ := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)

	b, err := repo.BatchDemandRelations(t.Context(), nil, []uint{1, 2})
	if err != nil || b.HasRelated(1) {
		t.Fatalf("nil actor 应返回全 false 且无错误，得到 err=%v", err)
	}
	b, err = repo.BatchDemandRelations(t.Context(), &model.User{Account: "x"}, nil)
	if err != nil || b.HasRelated(1) {
		t.Fatalf("空批应返回全 false 且无错误，得到 err=%v", err)
	}
	var nilRepo *Repo
	b, err = nilRepo.BatchDemandRelations(t.Context(), &model.User{Account: "x"}, []uint{1})
	if err != nil {
		t.Fatalf("nil repo 不应报错，得到 %v", err)
	}
	if b.HasRelated(1) {
		t.Fatal("nil repo 不应判定为有关系")
	}
}
