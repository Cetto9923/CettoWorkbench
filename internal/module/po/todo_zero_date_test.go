// =============================================================================
// 文件: internal/module/po/todo_zero_date_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证待办与 KPI 查询中的零日期写法改用 dateUnsetExpr / dateSetExpr，
//       禁止使用 = '0000-00-00' 字面量比对。
// =============================================================================

package po

import (
	"strings"
	"testing"
)

func TestTodoAndKPIQueries_NoZeroDateLiterals(t *testing.T) {
	// 1. todo_query_repo filter clauses
	filterSQL := getDemandFilterClauses(TodoListReq{Stage: "schedule"})
	if strings.Contains(filterSQL, "= '0000-00-00'") || strings.Contains(filterSQL, "!= '0000-00-00'") {
		t.Fatalf("getDemandFilterClauses schedule must not contain zero-date literals: %s", filterSQL)
	}
	if !strings.Contains(filterSQL, "LIKE '0000-00-00%'") {
		t.Fatalf("getDemandFilterClauses schedule must use dateUnsetExpr (LIKE '0000-00-00%%'): %s", filterSQL)
	}

	actionSQL := getDemandFilterClauses(TodoListReq{Action: TodoActionSchedule})
	if strings.Contains(actionSQL, "= '0000-00-00'") || strings.Contains(actionSQL, "!= '0000-00-00'") {
		t.Fatalf("getDemandFilterClauses action schedule must not contain zero-date literals: %s", actionSQL)
	}

	// 2. todo_query_extra SQL builders
	sqlStory, _ := buildTodoStorySQL("alice")
	if strings.Contains(sqlStory, "= '0000-00-00'") || strings.Contains(sqlStory, "!= '0000-00-00'") {
		t.Fatalf("buildTodoStorySQL must not contain zero-date literals: %s", sqlStory)
	}
	if !strings.Contains(sqlStory, "LIKE '0000-00-00%'") {
		t.Fatalf("buildTodoStorySQL must use dateUnsetExpr: %s", sqlStory)
	}

	sqlIssue, _ := buildTodoIssueSQL("alice")
	if strings.Contains(sqlIssue, "= '0000-00-00'") || strings.Contains(sqlIssue, "!= '0000-00-00'") {
		t.Fatalf("buildTodoIssueSQL must not contain zero-date literals: %s", sqlIssue)
	}

	sqlTodo, _ := buildTodoPersonalSQL("alice")
	if strings.Contains(sqlTodo, "= '0000-00-00'") || strings.Contains(sqlTodo, "!= '0000-00-00'") {
		t.Fatalf("buildTodoPersonalSQL must not contain zero-date literals: %s", sqlTodo)
	}

	sqlTask, _ := buildTodoTesttaskSQL("alice")
	if strings.Contains(sqlTask, "= '0000-00-00'") || strings.Contains(sqlTask, "!= '0000-00-00'") {
		t.Fatalf("buildTodoTesttaskSQL must not contain zero-date literals: %s", sqlTask)
	}
}
