// =============================================================================
// 文件: internal/module/po/repodone_enrich_sql_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 用 SQL 基准锁定 fetchObjectContexts 拆分前后的行为等价：10 种对象类型的
//       查询文本、绑定参数、发出顺序、空 ID 跳过规则全部逐字固定。
//       拆分只允许把代码搬到同包私有函数，不允许改 SQL、改顺序、改错误语义。
// 依赖: github.com/DATA-DOG/go-sqlmock, gorm.io/driver/mysql, gorm.io/gorm
// =============================================================================

package po

import (
	"fmt"
	"testing"
)

// objectContextSQLCase 一组对象行及其期望发出的 SQL 序列。
type objectContextSQLCase struct {
	name string
	rows []doneActionDBRow
	want []string // 期望 SQL，按实际发出顺序
}

// objectContextAllRows 覆盖全部 10 种会触发查询的对象类型，ID 各不相同，
// 用于同时锁定"哪类对象发哪条 SQL"和"发出顺序"。
func objectContextAllRows() []doneActionDBRow {
	return []doneActionDBRow{
		{ID: 1, ObjectType: "demand", ObjectID: 11},
		{ID: 2, ObjectType: "story", ObjectID: 22},
		{ID: 3, ObjectType: "task", ObjectID: 33},
		{ID: 4, ObjectType: "bug", ObjectID: 44},
		{ID: 5, ObjectType: "todo", ObjectID: 55},
		{ID: 6, ObjectType: "charter", ObjectID: 66},
		{ID: 7, ObjectType: "planchange", ObjectID: 77},
		{ID: 8, ObjectType: "buildguideline", ObjectID: 88},
		{ID: 9, ObjectType: "review", ObjectID: 99},
		{ID: 10, ObjectType: "case", ObjectID: 111},
	}
}

// objectContextSQLCases 覆盖全类型、单类型、未知类型与空输入。
// 审批类（charter/planchange/buildguideline/review/case）先于研发类发出，
// 拆分不得改变这个顺序。
func objectContextSQLCases() []objectContextSQLCase {
	all := objectContextAllRows()
	pick := func(types ...string) []doneActionDBRow {
		var out []doneActionDBRow
		for _, row := range all {
			for _, tp := range types {
				if row.ObjectType == tp {
					out = append(out, row)
				}
			}
		}
		return out
	}
	demandSQL := "SELECT d.id, d.name, d.status, d.assignedTo, COALESCE(p.name, '') AS product_name, COALESCE(dp.name, '') AS pool_name FROM zt_demand AS d LEFT JOIN zt_product AS p ON p.id = d.product LEFT JOIN zt_demandpool AS dp ON dp.id = d.pool AND dp.deleted = '0' WHERE d.id IN (?)"
	storySQL := "SELECT s.id, s.title, s.status, s.assignedTo, COALESCE(p.name, '') AS product_name, COALESCE(dp.name, '') AS pool_name FROM zt_story AS s LEFT JOIN zt_product AS p ON p.id = s.product LEFT JOIN zt_demand AS d ON d.id = s.fromDemand AND d.deleted = '0' LEFT JOIN zt_demandpool AS dp ON dp.id = d.pool AND dp.deleted = '0' WHERE s.id IN (?)"
	taskSQL := "SELECT t.id, t.name, t.status, t.project, t.execution, t.assignedTo, COALESCE(p.name, '') AS project_name, COALESCE(e.name, '') AS execution_name, COALESCE(dp.name, '') AS pool_name FROM zt_task AS t LEFT JOIN zt_project AS p ON p.id = t.project LEFT JOIN zt_project AS e ON e.id = t.execution LEFT JOIN zt_story AS s ON s.id = t.story AND s.deleted = '0' LEFT JOIN zt_demand AS d ON d.id = s.fromDemand AND d.deleted = '0' LEFT JOIN zt_demandpool AS dp ON dp.id = d.pool AND dp.deleted = '0' WHERE t.id IN (?)"
	bugSQL := "SELECT b.id, b.title, b.status, b.project, b.execution, b.assignedTo, COALESCE(p.name, '') AS project_name, COALESCE(e.name, '') AS execution_name FROM zt_bug AS b LEFT JOIN zt_project AS p ON p.id = b.project LEFT JOIN zt_project AS e ON e.id = b.execution WHERE b.id IN (?)"
	todoSQL := "SELECT id, name, status FROM `zt_todo` WHERE id IN (?)"
	charterSQL := "SELECT c.id, c.project, COALESCE(p.name, '') AS project_name FROM zt_charter AS c LEFT JOIN zt_project AS p ON p.id = c.project AND p.deleted = '0' WHERE c.id IN (?) AND c.deleted = '0'"
	planchangeSQL := "SELECT pc.id, pc.title, pc.project, COALESCE(p.name, '') AS project_name FROM zt_planchange AS pc LEFT JOIN zt_project AS p ON p.id = pc.project AND p.deleted = '0' WHERE pc.id IN (?)"
	buildguideSQL := "SELECT bg.id, bg.projectID, COALESCE(p.name, '') AS project_name FROM zt_projectbuildguide AS bg LEFT JOIN zt_project AS p ON p.id = bg.projectID AND p.deleted = '0' WHERE bg.id IN (?) AND bg.deleted = '0'"
	reviewSQL := "SELECT rv.id, rv.title, rv.project, COALESCE(p.name, '') AS project_name FROM zt_review AS rv LEFT JOIN zt_project AS p ON p.id = rv.project AND p.deleted = '0' WHERE rv.id IN (?) AND rv.deleted = '0'"
	caseSQL := "SELECT ca.id, ca.title, ca.project, COALESCE(p.name, '') AS project_name FROM zt_case AS ca LEFT JOIN zt_project AS p ON p.id = ca.project AND p.deleted = '0' WHERE ca.id IN (?) AND ca.deleted = '0'"

	return []objectContextSQLCase{
		{
			name: "all object types",
			rows: all,
			want: []string{
				charterSQL, planchangeSQL, buildguideSQL, reviewSQL, caseSQL,
				demandSQL, storySQL, taskSQL, bugSQL, todoSQL,
			},
		},
		{name: "demand only", rows: pick("demand"), want: []string{demandSQL}},
		{name: "story only", rows: pick("story"), want: []string{storySQL}},
		{name: "task only", rows: pick("task"), want: []string{taskSQL}},
		{name: "bug only", rows: pick("bug"), want: []string{bugSQL}},
		{name: "todo only", rows: pick("todo"), want: []string{todoSQL}},
		{name: "charter only", rows: pick("charter"), want: []string{charterSQL}},
		{name: "planchange only", rows: pick("planchange"), want: []string{planchangeSQL}},
		{name: "buildguideline only", rows: pick("buildguideline"), want: []string{buildguideSQL}},
		{name: "review only", rows: pick("review"), want: []string{reviewSQL}},
		{name: "case only", rows: pick("case"), want: []string{caseSQL}},
		{
			name: "approval before engineering classes",
			rows: append(pick("demand", "charter"), pick("todo")...),
			want: []string{charterSQL, demandSQL, todoSQL},
		},
		// 未知对象类型不发查询：risk / issue 等类型没有上下文表。
		{name: "unknown type is skipped", rows: []doneActionDBRow{{ID: 1, ObjectType: "risk", ObjectID: 9}}, want: nil},
		{name: "empty rows short circuits", rows: nil, want: nil},
	}
}

// TestFetchObjectContexts_SQLBaseline 锁定各对象类型组合下发出的 SQL 文本与顺序。
func TestFetchObjectContexts_SQLBaseline(t *testing.T) {
	for _, c := range objectContextSQLCases() {
		t.Run(c.name, func(t *testing.T) {
			repo, rec := newSQLBaselineRepo(t, 12)
			if _, err := repo.fetchObjectContexts(t.Context(), c.rows); err != nil {
				t.Fatalf("fetchObjectContexts 返回错误: %v", err)
			}
			if rec.count() != len(c.want) {
				t.Fatalf("查询条数 = %d，期望 %d\n实际: %#v", rec.count(), len(c.want), rec.queries)
			}
			for i, want := range c.want {
				if got := rec.sql(i); got != want {
					t.Errorf("第 %d 条 SQL 与基准不一致\n实际: %s\n基准: %s", i, got, want)
				}
			}
		})
	}
}

// TestFetchObjectContexts_ArgsMatchObjectIDs 锁定绑定参数按对象 ID 传入 IN 条件，
// 防止拆分时把某类对象的 ID 接到另一条查询上。
func TestFetchObjectContexts_ArgsMatchObjectIDs(t *testing.T) {
	repo, rec := newSQLBaselineRepo(t, 12)
	if _, err := repo.fetchObjectContexts(t.Context(), objectContextAllRows()); err != nil {
		t.Fatalf("fetchObjectContexts 返回错误: %v", err)
	}
	// 顺序：审批 5 类 → demand / story / task / bug / todo。
	wantIDs := []int64{66, 77, 88, 99, 111, 11, 22, 33, 44, 55}
	if rec.count() != len(wantIDs) {
		t.Fatalf("查询条数 = %d，期望 %d", rec.count(), len(wantIDs))
	}
	for i, want := range wantIDs {
		if got := rec.queries[i].args; len(got) != 1 || fmt.Sprint(got[0]) != fmt.Sprint(want) {
			t.Errorf("第 %d 条 SQL 参数 = %#v，期望 [%d]", i, got, want)
		}
	}
}

// TestFetchObjectContexts_MultipleIDsShareOneQuery 同一对象类型出现多个 ID 时
// 必须合并成一条 IN 查询，不允许拆成每行一条。
func TestFetchObjectContexts_MultipleIDsShareOneQuery(t *testing.T) {
	repo, rec := newSQLBaselineRepo(t, 12)
	rows := []doneActionDBRow{
		{ID: 1, ObjectType: "demand", ObjectID: 11},
		{ID: 2, ObjectType: "demand", ObjectID: 12},
		{ID: 3, ObjectType: "demand", ObjectID: 13},
	}
	if _, err := repo.fetchObjectContexts(t.Context(), rows); err != nil {
		t.Fatalf("fetchObjectContexts 返回错误: %v", err)
	}
	if rec.count() != 1 {
		t.Fatalf("同类对象应合并为 1 条查询，实际 %d 条", rec.count())
	}
	if got := rec.queries[0].args; len(got) != 3 {
		t.Fatalf("合并查询参数 = %#v，期望 3 个 ID", got)
	}
}
