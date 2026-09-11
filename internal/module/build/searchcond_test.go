package build

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeOperator(t *testing.T) {
	t.Parallel()
	if got := NormalizeOperator("include"); got != "include" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeOperator("bogus"); got != "=" {
		t.Fatalf("fallback got %q", got)
	}
}

func TestNormalizeAndOr(t *testing.T) {
	t.Parallel()
	if got := NormalizeAndOr("OR"); got != "or" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeAndOr("x"); got != "and" {
		t.Fatalf("got %q", got)
	}
}

func TestIsAllowedSearchField(t *testing.T) {
	t.Parallel()
	if !IsAllowedSearchField("title") {
		t.Fatal("title should be allowed")
	}
	if IsAllowedSearchField("product") {
		t.Fatal("product should be blocked for linkStory")
	}
	if IsAllowedSearchField("drop table") {
		t.Fatal("injection should be blocked")
	}
}

func TestBuildCondClauseIncludeTitle(t *testing.T) {
	t.Parallel()
	sql, args, ok := BuildCondClause(SearchCond{Field: "title", Operator: "include", Value: "登录"}, "")
	if !ok {
		t.Fatal("expected ok")
	}
	if !strings.Contains(sql, "LIKE") || len(args) != 1 || args[0] != "%登录%" {
		t.Fatalf("sql=%q args=%v", sql, args)
	}
}

func TestBuildCondClauseSkipEmpty(t *testing.T) {
	t.Parallel()
	_, _, ok := BuildCondClause(SearchCond{Field: "title", Operator: "include", Value: ""}, "")
	if ok {
		t.Fatal("empty value should skip")
	}
}

func TestBuildCondClauseEqualStatus(t *testing.T) {
	t.Parallel()
	sql, args, ok := BuildCondClause(SearchCond{Field: "status", Operator: "=", Value: "active"}, "select")
	if !ok || len(args) != 1 || args[0] != "active" {
		t.Fatalf("sql=%q args=%v ok=%v", sql, args, ok)
	}
	if !strings.Contains(sql, "status") {
		t.Fatalf("sql=%q", sql)
	}
}

func TestBuildCondClausePlanInclude(t *testing.T) {
	t.Parallel()
	sql, args, ok := BuildCondClause(SearchCond{Field: "plan", Operator: "include", Value: "12"}, "select")
	if !ok || len(args) != 1 {
		t.Fatalf("ok=%v args=%v", ok, args)
	}
	if !strings.Contains(sql, "CONCAT") {
		t.Fatalf("sql=%q", sql)
	}
}

func TestBuildCondClauseDateEqual(t *testing.T) {
	t.Parallel()
	sql, args, ok := BuildCondClause(SearchCond{Field: "openedDate", Operator: "=", Value: "2026-09-11"}, "date")
	if !ok || len(args) != 2 {
		t.Fatalf("ok=%v args=%v sql=%q", ok, args, sql)
	}
	if !strings.Contains(sql, ">=") || !strings.Contains(sql, "<=") {
		t.Fatalf("sql=%q", sql)
	}
}

func TestResolveDynamicValueToday(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 11, 15, 0, 0, 0, time.Local)
	begin, end, ok := ResolveDynamicDate("$today", now)
	if !ok {
		t.Fatal("expected ok")
	}
	if begin.Format("2006-01-02") != "2026-09-11" || end.Format("2006-01-02 15:04:05") != "2026-09-11 23:59:59" {
		t.Fatalf("begin=%v end=%v", begin, end)
	}
}

func TestLinkStoryListReqIsBySearch(t *testing.T) {
	t.Parallel()
	req := LinkStoryListReq{BrowseType: "bySearch"}
	if !req.IsBySearch() {
		t.Fatal("expected bySearch")
	}
	req2 := LinkStoryListReq{}
	if req2.IsBySearch() {
		t.Fatal("default should not bySearch")
	}
}

func TestBuildCondClauseModuleBelong(t *testing.T) {
	t.Parallel()
	sql, args, ok := BuildCondClause(SearchCond{Field: "module", Operator: "belong", Value: "5"}, "select", 5, 6, 7)
	if !ok {
		t.Fatal("expected ok")
	}
	if !strings.Contains(sql, "IN") || len(args) != 3 {
		t.Fatalf("sql=%q args=%v", sql, args)
	}
}
