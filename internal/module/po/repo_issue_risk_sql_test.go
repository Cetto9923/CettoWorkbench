// =============================================================================
// 文件: internal/module/po/repo_issue_risk_sql_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 用 GORM DryRun 捕获 FindIssueRiskList 与 CountIssueRiskProjects 生成的完整
//       SQL 文本与参数，作为「合并两处平行过滤条件」的行为基准。合并只允许减少
//       重复代码，两侧 WHERE 片段必须逐字相同且与基准一致。
// =============================================================================

package po

import (
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// dryRunRepo 构造一个只记录 SQL、不校验结果的 Repo。
func dryRunRepo(t *testing.T) (*Repo, *[]string) {
	t.Helper()
	var seen []string
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(
		func(_, actual string) error {
			seen = append(seen, actual)
			return nil // 接受任意 SQL，只记录
		},
	)))
	if err != nil {
		t.Fatalf("构造 dry-run 连接失败: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	gormDB, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开 dry-run 连接失败: %v", err)
	}

	// Count / Scan / Distinct 都以 Query 形式发出，逐条返回空结果。
	for i := 0; i < 8; i++ {
		mock.MatchExpectationsInOrder(false)
		mock.ExpectQuery("").WillReturnRows(sqlmock.NewRows([]string{"n"}))
	}
	return NewRepo(gormDB, gormDB), &seen
}

// issueRiskFilterCases 覆盖两个函数共享的全部过滤维度及其组合。
func issueRiskFilterCases() []struct {
	name    string
	account string
	req     IssueRiskListReq
} {
	return []struct {
		name    string
		account string
		req     IssueRiskListReq
	}{
		{"default personal scope", "alice", IssueRiskListReq{Kind: "issue", Page: 1, PageSize: 20}},
		{"risk alias", "alice", IssueRiskListReq{Kind: "risk", Page: 1, PageSize: 20}},
		{"relation myAction", "bob", IssueRiskListReq{Relation: "myAction", Page: 1, PageSize: 20}},
		{"relation mySubmit", "bob", IssueRiskListReq{Relation: "mySubmit", Page: 1, PageSize: 20}},
		{"status exact", "alice", IssueRiskListReq{Status: "active", Page: 1, PageSize: 20}},
		{"loop open", "alice", IssueRiskListReq{Loop: "open", Page: 1, PageSize: 20}},
		{"loop closed", "alice", IssueRiskListReq{Loop: "closed", Page: 1, PageSize: 20}},
		{"project", "alice", IssueRiskListReq{Project: 42, Page: 1, PageSize: 20}},
		{"keyword", "alice", IssueRiskListReq{Keyword: "登录", Page: 1, PageSize: 20}},
		{"overdue", "alice", IssueRiskListReq{Overdue: true, Page: 1, PageSize: 20}},
		{
			"team scope", "alice",
			IssueRiskListReq{Scope: "team", teamAccounts: []string{"t1", "t2"}, Page: 1, PageSize: 20},
		},
		{
			"all dimensions on risk alias", "alice",
			IssueRiskListReq{
				Kind: "risk", Relation: "myAction", Status: "active", Loop: "open",
				Project: 7, Keyword: "登录", Overdue: true, Page: 1, PageSize: 20,
			},
		},
		{
			"all dimensions on issue alias", "alice",
			IssueRiskListReq{
				Kind: "issue", Relation: "mySubmit", Status: "wait", Loop: "closed",
				Project: 3, Keyword: "支付", Overdue: true, Page: 1, PageSize: 20,
			},
		},
	}
}

// TestIssueRiskFilters_SharedConditionsIdentical 锁定两个函数在每个维度组合下
// 的 WHERE 条件逐字相同——这正是可以把七段过滤合并为一个私有函数的前提。
// 本测试在合并前后都必须通过。
func TestIssueRiskFilters_SharedConditionsIdentical(t *testing.T) {
	for _, tt := range issueRiskFilterCases() {
		t.Run(tt.name, func(t *testing.T) {
			listSQL := captureListSQL(t, tt.account, tt.req)
			projSQL := captureProjectSQL(t, tt.account, tt.req)

			listWhere := whereOf(listSQL)
			projWhere := whereOf(projSQL)

			if listWhere != projWhere {
				t.Errorf("两侧 WHERE 条件不一致，合并会改变行为：\n列表侧: %s\n统计侧: %s", listWhere, projWhere)
			}
		})
	}
}

// TestIssueRiskFilters_SQLBaseline 锁定各维度组合下的完整 SQL 文本。
// 合并过滤条件后本测试必须逐字通过。
func TestIssueRiskFilters_SQLBaseline(t *testing.T) {
	want := map[string]string{
		"default personal scope": " WHERE i.deleted = '0' AND ((i.createdBy = ? OR i.assignedTo = ?))",
		"risk alias":             " WHERE k.deleted = '0' AND ((k.createdBy = ? OR k.assignedTo = ?))",
		"relation myAction":      " WHERE i.deleted = '0' AND ((i.createdBy = ? OR i.assignedTo = ?)) AND i.assignedTo = ?",
		"relation mySubmit":      " WHERE i.deleted = '0' AND ((i.createdBy = ? OR i.assignedTo = ?)) AND i.createdBy = ?",
		"status exact":           " WHERE i.deleted = '0' AND ((i.createdBy = ? OR i.assignedTo = ?)) AND i.status = ?",
		"loop open":              " WHERE i.deleted = '0' AND ((i.createdBy = ? OR i.assignedTo = ?)) AND i.status IN (?,?,?,?,?,?)",
		"loop closed":            " WHERE i.deleted = '0' AND ((i.createdBy = ? OR i.assignedTo = ?)) AND i.status IN (?,?,?,?)",
		"project":                " WHERE i.deleted = '0' AND ((i.createdBy = ? OR i.assignedTo = ?)) AND p.id = ?",
		"keyword":                " WHERE i.deleted = '0' AND ((i.createdBy = ? OR i.assignedTo = ?)) AND ((CAST(i.id AS CHAR) LIKE ? OR i.title LIKE ? OR COALESCE(p.name,'') LIKE ?))",
		"team scope":             " WHERE i.deleted = '0' AND ((i.createdBy IN (?,?) OR i.assignedTo IN (?,?)))",
	}

	for _, tt := range issueRiskFilterCases() {
		t.Run(tt.name, func(t *testing.T) {
			got := whereOf(captureListSQL(t, tt.account, tt.req))

			exp, ok := want[tt.name]
			if !ok {
				if !strings.Contains(got, "deleted = '0'") {
					t.Fatalf("未覆盖的用例应至少含 deleted 条件，实际: %s", got)
				}
				return
			}
			if got != exp {
				t.Errorf("WHERE 条件与基准不一致:\n实际: %s\n基准: %s", got, exp)
			}
		})
	}
}

// TestIssueRiskFilters_OverdueUsesPlanDate 锁定逾期判定按计划日期列，
// 且 issue/risk 指向各自的计划日期列。
func TestIssueRiskFilters_OverdueUsesPlanDate(t *testing.T) {
	for _, tt := range []struct {
		kind string
		col  string
	}{
		{"issue", "i.deadline"},
		{"risk", "k.plannedClosedDate"},
	} {
		t.Run(tt.kind, func(t *testing.T) {
			sql := captureListSQL(t, "alice", IssueRiskListReq{
				Kind: tt.kind, Overdue: true, Page: 1, PageSize: 20,
			})
			want := tt.col + " IS NOT NULL AND CAST(" + tt.col + " AS CHAR) NOT LIKE '0000-00-00%' AND " + tt.col + " < ?"
			if !strings.Contains(sql, want) {
				t.Errorf("逾期条件应基于 %s，实际 SQL: %s", tt.col, sql)
			}
		})
	}
}

// TestIssueRiskFilters_ListKeepsUserJoins 锁定列表侧保留 4 个用户 JOIN，
// 统计侧只 JOIN 项目表——两者不可互相污染。
func TestIssueRiskFilters_ListKeepsUserJoins(t *testing.T) {
	req := IssueRiskListReq{Kind: "issue", Page: 1, PageSize: 20}

	list := captureListSQL(t, "alice", req)
	for _, j := range []string{"LEFT JOIN zt_user cu", "LEFT JOIN zt_user au", "LEFT JOIN zt_user ru", "LEFT JOIN zt_user clu"} {
		if !strings.Contains(list, j) {
			t.Errorf("列表侧缺少 %s", j)
		}
	}

	proj := captureProjectSQL(t, "alice", req)
	if strings.Contains(proj, "LEFT JOIN zt_user") {
		t.Errorf("统计侧不应 JOIN 用户表: %s", proj)
	}
	if !strings.Contains(proj, "LEFT JOIN zt_project p") {
		t.Errorf("统计侧缺少项目 JOIN: %s", proj)
	}
}

// TestCountIssueRiskProjects_EmptyAccountShortCircuits 空账号不查库。
func TestCountIssueRiskProjects_EmptyAccountShortCircuits(t *testing.T) {
	repo, _ := dryRunRepo(t)
	projects, err := repo.CountIssueRiskProjects(t.Context(), "  ", IssueRiskListReq{Page: 1, PageSize: 20})
	if err != nil || len(projects) != 0 {
		t.Fatalf("空账号应短路返回空结果，得到 projects=%v err=%v", projects, err)
	}
}

// TestCountIssueRiskProjects_TeamScopeWithoutAccountsShortCircuits 团队视角无账号时短路。
func TestCountIssueRiskProjects_TeamScopeWithoutAccountsShortCircuits(t *testing.T) {
	repo, _ := dryRunRepo(t)
	projects, err := repo.CountIssueRiskProjects(t.Context(), "alice",
		IssueRiskListReq{Scope: "team", teamAccounts: nil, Page: 1, PageSize: 20})
	if err != nil || len(projects) != 0 {
		t.Fatalf("团队视角无账号应短路，得到 projects=%v err=%v", projects, err)
	}
}

// whereOf 取出 SQL 的 WHERE 子句（到 ORDER BY / LIMIT 之前）。
func whereOf(sql string) string {
	if i := strings.Index(sql, " WHERE "); i >= 0 {
		rest := sql[i+len(" WHERE "):]
		for _, tail := range []string{" ORDER BY ", " LIMIT "} {
			if j := strings.Index(rest, tail); j >= 0 {
				rest = rest[:j]
			}
		}
		return " WHERE " + rest
	}
	return ""
}

// captureListSQL 捕获 FindIssueRiskList 主查询（Select 那条）拼出的 SQL 文本。
func captureListSQL(t *testing.T, account string, req IssueRiskListReq) string {
	t.Helper()
	repo, seen := dryRunRepo(t)
	if _, _, err := repo.FindIssueRiskList(t.Context(), account, req); err != nil {
		t.Fatalf("FindIssueRiskList 失败: %v", err)
	}
	return lastSelectSQL(*seen)
}

// captureProjectSQL 捕获 CountIssueRiskProjects 拼出的 SQL 文本。
func captureProjectSQL(t *testing.T, account string, req IssueRiskListReq) string {
	t.Helper()
	repo, seen := dryRunRepo(t)
	if _, err := repo.CountIssueRiskProjects(t.Context(), account, req); err != nil {
		t.Fatalf("CountIssueRiskProjects 失败: %v", err)
	}
	return lastSelectSQL(*seen)
}

// lastSelectSQL 取最后一条非 count 的查询文本。
func lastSelectSQL(seen []string) string {
	for i := len(seen) - 1; i >= 0; i-- {
		if !strings.HasPrefix(seen[i], "SELECT count(") {
			return seen[i]
		}
	}
	if len(seen) == 0 {
		return ""
	}
	return seen[len(seen)-1]
}
