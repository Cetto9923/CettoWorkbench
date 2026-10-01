// =============================================================================
// 文件: internal/module/po/sqlbaseline_test.go
// 模块: PO 工作台
// 类型: test
// 职责: GORM DryRun SQL 基准工具：记录 Repo 查询期间发出的 SQL 文本与绑定参数，
//       供拆分超长函数前后逐字比对。连接走 sqlmock，不触真实库。
// 依赖: github.com/DATA-DOG/go-sqlmock, gorm.io/driver/mysql, gorm.io/gorm
// =============================================================================

package po

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// sqlBaselineQuery 一次查询的 SQL 文本与绑定参数。
type sqlBaselineQuery struct {
	sql  string
	args []any
}

// sqlBaselineRecorder 记录查询期间的全部 SQL 文本与参数。
type sqlBaselineRecorder struct {
	queries []sqlBaselineQuery
}

// sql 返回第 i 条查询的 SQL 文本。
func (r *sqlBaselineRecorder) sql(i int) string {
	if i < 0 || i >= len(r.queries) {
		return ""
	}
	return r.queries[i].sql
}

// count 已记录的查询条数。
func (r *sqlBaselineRecorder) count() int { return len(r.queries) }

// newSQLBaselineRepo 构造只拼 SQL、不校验结果的 Repo，并返回记录器。
// 回调挂在本次 gorm.Open 独有的 Callback 注册表上，不会跨用例泄漏。
func newSQLBaselineRepo(t *testing.T, queryBudget int) (*Repo, *sqlBaselineRecorder) {
	t.Helper()
	rec := &sqlBaselineRecorder{}

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("构造 sqlmock 连接失败: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	gormDB, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开 gorm 连接失败: %v", err)
	}
	// Scan/Find 走 gorm:row，Count 走 gorm:query，Raw 走 gorm:raw，三者都要挂钩子。
	record := func(tx *gorm.DB) {
		rec.queries = append(rec.queries, sqlBaselineQuery{
			sql:  tx.Statement.SQL.String(),
			args: append([]any(nil), tx.Statement.Vars...),
		})
	}
	const recorderName = "po:sql_baseline"
	// processor 是 gorm 内部未导出类型，只能逐个注册而不放进切片。
	if err := gormDB.Callback().Row().After("gorm:row").Register(recorderName, record); err != nil {
		t.Fatalf("注册 SQL 记录回调失败(row): %v", err)
	}
	if err := gormDB.Callback().Query().After("gorm:query").Register(recorderName, record); err != nil {
		t.Fatalf("注册 SQL 记录回调失败(query): %v", err)
	}
	if err := gormDB.Callback().Raw().After("gorm:raw").Register(recorderName, record); err != nil {
		t.Fatalf("注册 SQL 记录回调失败(raw): %v", err)
	}

	// Count / Scan / Find 都以 Query 形式发出，逐条返回空结果集。
	for i := 0; i < queryBudget; i++ {
		mock.MatchExpectationsInOrder(false)
		mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"n"}))
	}
	return NewRepo(gormDB, gormDB), rec
}
