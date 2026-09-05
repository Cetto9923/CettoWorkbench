// =============================================================================
// 文件: internal/module/po/todo_query_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证待办 SQL 查询构建器、过滤条件及全局排序一致性。
// 依赖: 无
// =============================================================================

package po

import (
	"strings"
	"testing"
)

func TestBuildTodoUnionSQLObjectFiltering(t *testing.T) {
	// 1. 全部类型包含 demand, task, bug
	sqlAll, _ := buildTodoUnionSQL("user_a", TodoListReq{})
	if !strings.Contains(sqlAll, "'demand' AS kind") ||
		!strings.Contains(sqlAll, "'task' AS kind") ||
		!strings.Contains(sqlAll, "'bug' AS kind") {
		t.Errorf("expected all 3 kinds in union, got: %s", sqlAll)
	}

	// 2. 仅 demand
	sqlDemand, _ := buildTodoUnionSQL("user_a", TodoListReq{ObjectType: "demand"})
	if !strings.Contains(sqlDemand, "'demand' AS kind") ||
		strings.Contains(sqlDemand, "'task' AS kind") ||
		strings.Contains(sqlDemand, "'bug' AS kind") {
		t.Errorf("expected only demand in union, got: %s", sqlDemand)
	}

	// 3. 仅 task
	sqlTask, _ := buildTodoUnionSQL("user_a", TodoListReq{ObjectType: "task"})
	if strings.Contains(sqlTask, "'demand' AS kind") ||
		!strings.Contains(sqlTask, "'task' AS kind") ||
		strings.Contains(sqlTask, "'bug' AS kind") {
		t.Errorf("expected only task in union, got: %s", sqlTask)
	}

	// 4. 阶段过滤仅保留 demand
	sqlStage, _ := buildTodoUnionSQL("user_a", TodoListReq{Stage: "accept"})
	if !strings.Contains(sqlStage, "'demand' AS kind") ||
		strings.Contains(sqlStage, "'task' AS kind") ||
		strings.Contains(sqlStage, "'bug' AS kind") {
		t.Errorf("expected stage filter to exclude task and bug, got: %s", sqlStage)
	}
}

func TestBuildTodoOuterWhereKeywords(t *testing.T) {
	displayMap := map[string]string{
		"user_a": "张三(user_a)",
		"user_b": "李四",
	}

	// 搜索人名 "张三"
	where, args := buildTodoOuterWhere(TodoListReq{Keyword: "张三"}, displayMap, false, "2026-09-05")
	if !strings.Contains(where, "owner_account IN") {
		t.Errorf("expected keyword matching owner name to filter by owner_account, got: %s", where)
	}
	hasUserA := false
	for _, arg := range args {
		if accounts, ok := arg.([]string); ok {
			for _, acc := range accounts {
				if acc == "user_a" {
					hasUserA = true
				}
			}
		}
	}
	if !hasUserA {
		t.Errorf("expected user_a in matching accounts args, got: %#v", args)
	}

	// 搜索编号 "US12"
	whereNum, _ := buildTodoOuterWhere(TodoListReq{Keyword: "US12"}, displayMap, false, "2026-09-05")
	if strings.Contains(whereNum, "owner_account IN") {
		t.Errorf("expected keyword with no matching owners to not filter by owner_account, got: %s", whereNum)
	}
	if !strings.Contains(whereNum, "LOWER(t.display_id) LIKE") {
		t.Errorf("expected display_id LIKE in where, got: %s", whereNum)
	}
}

func TestBuildTodoOuterWhereDimensions(t *testing.T) {
	// Relation
	whereRel, _ := buildTodoOuterWhere(TodoListReq{Relation: RelationInCharge}, nil, false, "2026-09-05")
	if !strings.Contains(whereRel, "t.relation = '我负责'") {
		t.Errorf("expected relation condition, got: %s", whereRel)
	}

	// Responsibility
	whereResp, _ := buildTodoOuterWhere(TodoListReq{Responsibility: ResponsibilityMyAction}, nil, false, "2026-09-05")
	if !strings.Contains(whereResp, "t.responsibility = '待我处理'") {
		t.Errorf("expected responsibility condition, got: %s", whereResp)
	}

	// Tab
	whereTab, _ := buildTodoOuterWhere(TodoListReq{Tab: TodoTabDemand}, nil, true, "2026-09-05")
	if !strings.Contains(whereTab, "t.kind = 'demand'") {
		t.Errorf("expected tab condition, got: %s", whereTab)
	}

	// Focus
	whereFocus, _ := buildTodoOuterWhere(TodoListReq{Focus: "today"}, nil, true, "2026-09-05")
	if !strings.Contains(whereFocus, "t.deadline_str = ?") {
		t.Errorf("expected focus today condition, got: %s", whereFocus)
	}
}
