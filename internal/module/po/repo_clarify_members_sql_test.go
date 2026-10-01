// =============================================================================
// 文件: internal/module/po/repo_clarify_members_sql_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 用 SQL 基准锁定 FindProductMembers 的取数构成，作为拆分该超长函数的
//       行为基准。基准固定三件事：查询条数与顺序（产品 → 历史 PM → 账号姓名）、
//       zt_demandclarify.product 的绑定类型是字符串而非整型（该列是 varchar）、
//       账号集非空时才发第三条查询。
// 依赖: github.com/DATA-DOG/go-sqlmock, gorm.io/driver/mysql, gorm.io/gorm
// =============================================================================

package po

import (
	"strings"
	"testing"
)

// TestFindProductMembers_SQLBaseline 锁定三条查询的文本、顺序与参数。
// zt_demandclarify.product 是 varchar，绑定参数必须是字符串 ID；
// 传整型会让 MySQL/OceanBase 隐式转换失败。
func TestFindProductMembers_SQLBaseline(t *testing.T) {
	repo, rec := newSQLBaselineRepo(t, 8)
	if _, err := repo.FindProductMembers(t.Context(), []int64{1, 2}); err != nil {
		t.Fatalf("FindProductMembers 返回错误: %v", err)
	}
	if rec.count() != 2 {
		t.Fatalf("查询条数 = %d，期望 2（无账号时跳过姓名查询）", rec.count())
	}
	productSQL := "SELECT id, PO, QD, RD, feedback, ticket, createdBy, whitelist FROM `zt_product` WHERE id IN (?,?) AND deleted = '0'"
	if got := rec.sql(0); got != productSQL {
		t.Errorf("第 1 条 SQL 与基准不一致\n实际: %s\n基准: %s", got, productSQL)
	}
	pmSQL := "SELECT DISTINCT product, PM FROM `zt_demandclarify` WHERE product IN (?,?) AND PM != ''"
	if got := rec.sql(1); got != pmSQL {
		t.Errorf("第 2 条 SQL 与基准不一致\n实际: %s\n基准: %s", got, pmSQL)
	}
	// 姓名查询在账号集为空时不得发出。
	if q := rec.findLast(func(s string) bool { return strings.Contains(s, "zt_user") }); q != nil {
		t.Errorf("账号集为空时不应查询 zt_user，实际: %s", q.sql)
	}
}

// TestFindProductMembers_ClarifyProductBoundAsString 锁定 zt_demandclarify.product
// 的绑定参数是字符串而非 int64：该列在禅道里是 varchar，
// 用整型比较在 OceanBase 上会触发隐式转换并可能漏行。
func TestFindProductMembers_ClarifyProductBoundAsString(t *testing.T) {
	repo, rec := newSQLBaselineRepo(t, 8)
	if _, err := repo.FindProductMembers(t.Context(), []int64{7, 42}); err != nil {
		t.Fatalf("FindProductMembers 返回错误: %v", err)
	}
	args := rec.queries[1].args
	if len(args) != 2 {
		t.Fatalf("PM 查询参数个数 = %d，期望 2", len(args))
	}
	for i, want := range []string{"7", "42"} {
		s, ok := args[i].(string)
		if !ok {
			t.Errorf("第 %d 个参数类型 = %T，期望 string", i, args[i])
			continue
		}
		if s != want {
			t.Errorf("第 %d 个参数 = %q，期望 %q", i, s, want)
		}
	}
}

// TestFindProductMembers_QueryOrder 锁定取数顺序：产品 → 历史 PM → 账号姓名。
// 姓名查询依赖前两条收集到的账号，必须排在最后。
func TestFindProductMembers_QueryOrder(t *testing.T) {
	repo, rec := newSQLBaselineRepo(t, 8)
	if _, err := repo.FindProductMembers(t.Context(), []int64{1}); err != nil {
		t.Fatalf("FindProductMembers 返回错误: %v", err)
	}
	if !strings.Contains(rec.sql(0), "FROM `zt_product`") {
		t.Errorf("第 1 条应查产品表，实际: %s", rec.sql(0))
	}
	if !strings.Contains(rec.sql(1), "FROM `zt_demandclarify`") {
		t.Errorf("第 2 条应查需求分析表，实际: %s", rec.sql(1))
	}
}

// TestFindProductMembers_EmptyProductsShortCircuits 空产品列表不查库，
// 返回空结果而非报错。
func TestFindProductMembers_EmptyProductsShortCircuits(t *testing.T) {
	repo, rec := newSQLBaselineRepo(t, 8)
	got, err := repo.FindProductMembers(t.Context(), nil)
	if err != nil {
		t.Fatalf("空产品列表不应报错: %v", err)
	}
	if got == nil {
		t.Fatal("空产品列表应返回非 nil 空 map")
	}
	if len(got) != 0 {
		t.Errorf("空产品列表应返回空 map，实际 %v", got)
	}
	if rec.count() != 0 {
		t.Errorf("空产品列表不应查库，实际发出 %d 条查询", rec.count())
	}
}

// TestFindProductMembers_NameLookupSkippedWhenNoAccounts 锁定账号集为空时
// 跳过 zt_user 查询：产品行全部为空值时不产生多余往返。
func TestFindProductMembers_NameLookupSkippedWhenNoAccounts(t *testing.T) {
	repo, rec := newSQLBaselineRepo(t, 8)
	if _, err := repo.FindProductMembers(t.Context(), []int64{1}); err != nil {
		t.Fatalf("FindProductMembers 返回错误: %v", err)
	}
	for i, q := range rec.queries {
		if strings.Contains(q.sql, "SELECT account, realname") {
			t.Errorf("第 %d 条不应为姓名查询: %s", i, q.sql)
		}
	}
	if !containsAll(rec.sql(0), "id, PO, QD, RD, feedback, ticket, createdBy, whitelist") {
		t.Errorf("产品投影列不全，缺一列会漏掉某类参与人: %s", rec.sql(0))
	}
}
