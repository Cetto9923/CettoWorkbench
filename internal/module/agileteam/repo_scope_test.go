// =============================================================================
// 文件: internal/module/agileteam/repo_scope_test.go
// 模块: 敏捷小组治理
// 类型: security regression
// 职责: 锁定部门负责人只能按本人负责部门及下级确定范围。
// =============================================================================

package agileteam

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"workbench/internal/constants"
)

// techHQArgs 返回管辖范围查询中以 ? 占位符参数化的科技本部口径参数。
func techHQArgs() []driver.Value {
	return []driver.Value{
		constants.DeptTechHQID, constants.DeptTechHQPathPattern(),
		constants.DeptTechHQID, constants.DeptTechHQPathPattern(),
	}
}

func TestListDeptTreeIDsUsesManagedDepartmentsAndDescendants(t *testing.T) {
	repo, mock := newTestRepo(t)
	mock.ExpectQuery("(?s)SELECT d\\.id, COALESCE\\(d\\.path, ''\\) AS path FROM zt_dept d.*LEFT JOIN zt_wb_dept_manager_override.*REGEXP \\?").
		WithArgs(append(techHQArgs(), "(^|[[:space:],;])lead1([[:space:],;]|$)")...).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}).
			AddRow(uint(20), ",1,20,").
			AddRow(uint(40), ",1,40,"))
	mock.ExpectQuery("(?s)SELECT DISTINCT id FROM zt_dept WHERE id IN \\(\\?,\\?\\) OR path LIKE \\? OR path LIKE \\?").
		WithArgs(uint(20), uint(40), ",1,20,%", ",1,40,%").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(uint(20)).
			AddRow(uint(21)).
			AddRow(uint(40)).
			AddRow(uint(41)))

	got, err := repo.ListDeptTreeIDs(context.Background(), " lead1 ")
	if err != nil {
		t.Fatalf("ListDeptTreeIDs() error = %v", err)
	}
	want := []uint{20, 21, 40, 41}
	if len(got) != len(want) {
		t.Fatalf("ListDeptTreeIDs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ListDeptTreeIDs() = %v, want %v", got, want)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}

func TestListDeptTreeIDsReturnsEmptyWhenAccountManagesNoDepartment(t *testing.T) {
	repo, mock := newTestRepo(t)
	mock.ExpectQuery("(?s)SELECT d\\.id, COALESCE\\(d\\.path, ''\\) AS path FROM zt_dept d.*LEFT JOIN zt_wb_dept_manager_override.*REGEXP \\?").
		WithArgs(append(techHQArgs(), "(^|[[:space:],;])member1([[:space:],;]|$)")...).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}))

	got, err := repo.ListDeptTreeIDs(context.Background(), "member1")
	if err != nil {
		t.Fatalf("ListDeptTreeIDs() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ListDeptTreeIDs() = %v, want empty", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}

// TestListDeptTreeIDs_DeptManagerOverridePriority 校验 T5 三种优先级分支：
// case 1: 仅 zt_dept.manager 有值，正确取到
// case 2: 仅补缺表有值，正确取到
// case 3: 两边都有值，补缺表优先生效
func TestListDeptTreeIDs_DeptManagerOverridePriority(t *testing.T) {
	t.Run("case1_zentao_only", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		// 仅 zt_dept.manager = "lead1"，补缺表无记录 -> 命中部门 14
		mock.ExpectQuery("(?s)SELECT d\\.id, COALESCE\\(d\\.path, ''\\) AS path FROM zt_dept d.*LEFT JOIN zt_wb_dept_manager_override.*REGEXP \\?").
			WithArgs(append(techHQArgs(), "(^|[[:space:],;])lead1([[:space:],;]|$)")...).
			WillReturnRows(sqlmock.NewRows([]string{"id", "path"}).AddRow(uint(14), ",53,52,14,"))
		mock.ExpectQuery("(?s)SELECT DISTINCT id FROM zt_dept WHERE id IN \\(\\?\\) OR path LIKE \\?").
			WithArgs(uint(14), ",53,52,14,%").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(14)))

		got, err := repo.ListDeptTreeIDs(context.Background(), "lead1")
		if err != nil {
			t.Fatalf("case 1 error: %v", err)
		}
		if len(got) != 1 || got[0] != 14 {
			t.Fatalf("case 1 got = %v, want [14]", got)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("case 1 expectations: %v", err)
		}
	})

	t.Run("case2_override_only", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		// 仅补缺表有值 account = "lead2" (zt_dept.manager 为空) -> lead2 命中部门 14
		mock.ExpectQuery("(?s)SELECT d\\.id, COALESCE\\(d\\.path, ''\\) AS path FROM zt_dept d.*LEFT JOIN zt_wb_dept_manager_override.*REGEXP \\?").
			WithArgs(append(techHQArgs(), "(^|[[:space:],;])lead2([[:space:],;]|$)")...).
			WillReturnRows(sqlmock.NewRows([]string{"id", "path"}).AddRow(uint(14), ",53,52,14,"))
		mock.ExpectQuery("(?s)SELECT DISTINCT id FROM zt_dept WHERE id IN \\(\\?\\) OR path LIKE \\?").
			WithArgs(uint(14), ",53,52,14,%").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(14)))

		got, err := repo.ListDeptTreeIDs(context.Background(), "lead2")
		if err != nil {
			t.Fatalf("case 2 error: %v", err)
		}
		if len(got) != 1 || got[0] != 14 {
			t.Fatalf("case 2 got = %v, want [14]", got)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("case 2 expectations: %v", err)
		}
	})

	t.Run("case3_override_over_zentao", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		// zt_dept.manager 为 "lead1"，但补缺表覆盖为 "lead3"
		// 查询 lead3: 补缺表优先生效，命中部门 14
		mock.ExpectQuery("(?s)SELECT d\\.id, COALESCE\\(d\\.path, ''\\) AS path FROM zt_dept d.*LEFT JOIN zt_wb_dept_manager_override.*REGEXP \\?").
			WithArgs(append(techHQArgs(), "(^|[[:space:],;])lead3([[:space:],;]|$)")...).
			WillReturnRows(sqlmock.NewRows([]string{"id", "path"}).AddRow(uint(14), ",53,52,14,"))
		mock.ExpectQuery("(?s)SELECT DISTINCT id FROM zt_dept WHERE id IN \\(\\?\\) OR path LIKE \\?").
			WithArgs(uint(14), ",53,52,14,%").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(14)))

		got, err := repo.ListDeptTreeIDs(context.Background(), "lead3")
		if err != nil {
			t.Fatalf("case 3 lead3 error: %v", err)
		}
		if len(got) != 1 || got[0] != 14 {
			t.Fatalf("case 3 lead3 got = %v, want [14]", got)
		}

		// 查询原 zt_dept.manager 的 "lead1": 因为被补缺表覆盖，无法匹配到部门 14，返回空
		mock.ExpectQuery("(?s)SELECT d\\.id, COALESCE\\(d\\.path, ''\\) AS path FROM zt_dept d.*LEFT JOIN zt_wb_dept_manager_override.*REGEXP \\?").
			WithArgs(append(techHQArgs(), "(^|[[:space:],;])lead1([[:space:],;]|$)")...).
			WillReturnRows(sqlmock.NewRows([]string{"id", "path"}))

		got1, err1 := repo.ListDeptTreeIDs(context.Background(), "lead1")
		if err1 != nil {
			t.Fatalf("case 3 lead1 error: %v", err1)
		}
		if len(got1) != 0 {
			t.Fatalf("case 3 lead1 got = %v, want empty (overridden by lead3)", got1)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("case 3 expectations: %v", err)
		}
	})
}

// TestIsDeptManager_DeptManagerOverride 验证 IsDeptManager 识别补缺表：
// 1. 只在补缺表里的账号返回 true；
// 2. 只在 zt_dept.manager 里的账号返回 true；
// 3. 补缺表覆盖了原 manager 的部门，原 manager 返回 false。
func TestIsDeptManager_DeptManagerOverride(t *testing.T) {
	t.Run("override_only_returns_true", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		// 1. 只在补缺表里的账号返回 true (demo_leader 在补缺表有记录，zt_dept.manager 为空)
		mock.ExpectQuery("(?s)SELECT d\\.id, COALESCE\\(d\\.path, ''\\) AS path FROM zt_dept d.*LEFT JOIN zt_wb_dept_manager_override.*REGEXP \\?").
			WithArgs(append(techHQArgs(), "(^|[[:space:],;])demo_leader([[:space:],;]|$)")...).
			WillReturnRows(sqlmock.NewRows([]string{"id", "path"}).AddRow(uint(14), ",53,52,14,"))
		mock.ExpectQuery("(?s)SELECT DISTINCT id FROM zt_dept WHERE id IN \\(\\?\\) OR path LIKE \\?").
			WithArgs(uint(14), ",53,52,14,%").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(14)))

		isMgr, err := repo.IsDeptManager(context.Background(), "demo_leader")
		if err != nil {
			t.Fatalf("case 1 error: %v", err)
		}
		if !isMgr {
			t.Fatalf("case 1 got false, want true for override_only")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("case 1 expectations: %v", err)
		}
	})

	t.Run("zentao_only_returns_true", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		// 2. 只在 zt_dept.manager 里的账号返回 true
		mock.ExpectQuery("(?s)SELECT d\\.id, COALESCE\\(d\\.path, ''\\) AS path FROM zt_dept d.*LEFT JOIN zt_wb_dept_manager_override.*REGEXP \\?").
			WithArgs(append(techHQArgs(), "(^|[[:space:],;])lead1([[:space:],;]|$)")...).
			WillReturnRows(sqlmock.NewRows([]string{"id", "path"}).AddRow(uint(14), ",53,52,14,"))
		mock.ExpectQuery("(?s)SELECT DISTINCT id FROM zt_dept WHERE id IN \\(\\?\\) OR path LIKE \\?").
			WithArgs(uint(14), ",53,52,14,%").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(14)))

		isMgr, err := repo.IsDeptManager(context.Background(), "lead1")
		if err != nil {
			t.Fatalf("case 2 error: %v", err)
		}
		if !isMgr {
			t.Fatalf("case 2 got false, want true for zentao_only")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("case 2 expectations: %v", err)
		}
	})

	t.Run("overridden_original_manager_returns_false", func(t *testing.T) {
		repo, mock := newTestRepo(t)
		// 3. 补缺表覆盖了原 manager 的部门，原 manager 返回 false
		mock.ExpectQuery("(?s)SELECT d\\.id, COALESCE\\(d\\.path, ''\\) AS path FROM zt_dept d.*LEFT JOIN zt_wb_dept_manager_override.*REGEXP \\?").
			WithArgs(append(techHQArgs(), "(^|[[:space:],;])lead_old([[:space:],;]|$)")...).
			WillReturnRows(sqlmock.NewRows([]string{"id", "path"}))

		isMgr, err := repo.IsDeptManager(context.Background(), "lead_old")
		if err != nil {
			t.Fatalf("case 3 error: %v", err)
		}
		if isMgr {
			t.Fatalf("case 3 got true, want false for overridden manager")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("case 3 expectations: %v", err)
		}
	})
}

func TestListMappedTeamgroupIDsByDeptsUsesOnlyActiveMappings(t *testing.T) {
	repo, mock := newTestRepo(t)
	mock.ExpectQuery("(?s)FROM zt_wb_agileteam_orgmap m.*mapped.type = 'parent'.*m.status = 'active'.*m.deptId IN \\(\\?,\\?\\)").
		WithArgs(uint(20), uint(21)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(3)).AddRow(uint(11)).AddRow(uint(12)))

	got, err := repo.ListMappedTeamgroupIDsByDepts(context.Background(), []uint{20, 21})
	if err != nil {
		t.Fatalf("ListMappedTeamgroupIDsByDepts() error = %v", err)
	}
	want := []uint{3, 11, 12}
	if len(got) != len(want) {
		t.Fatalf("mapped teamgroup IDs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mapped teamgroup IDs = %v, want %v", got, want)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}

func TestListManagedTeamgroupIDsScopesParentCoachToFamilyAndChildCoachToSelf(t *testing.T) {
	repo, mock := newTestRepo(t)
	mock.ExpectQuery("(?s)FROM zt_teamgroup managed.*scoped.parent = managed.id.*REGEXP \\?").
		WithArgs("(^|[[:space:],;])lead1([[:space:],;]|$)").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint(3)).AddRow(uint(11)).AddRow(uint(12)))

	got, err := repo.ListManagedTeamgroupIDs(context.Background(), " lead1 ")
	if err != nil {
		t.Fatalf("ListManagedTeamgroupIDs() error = %v", err)
	}
	want := []uint{3, 11, 12}
	if len(got) != len(want) {
		t.Fatalf("managed teamgroup IDs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("managed teamgroup IDs = %v, want %v", got, want)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}

func TestAccountTokenRegexpDoesNotMatchAccountSubstring(t *testing.T) {
	rx, err := regexp.Compile(accountTokenRegexp("lead1"))
	if err != nil {
		t.Fatalf("account token regexp: %v", err)
	}
	if rx.MatchString("teamlead1") || rx.MatchString("lead10") {
		t.Fatal("account token must not match a longer account")
	}
	if !rx.MatchString("lead1, coach2") || !rx.MatchString("coach2;lead1") {
		t.Fatal("account token should match exact delimited accounts")
	}
}

func newTestRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	svc, mock := newTestService(t)
	return svc.repo, mock
}
