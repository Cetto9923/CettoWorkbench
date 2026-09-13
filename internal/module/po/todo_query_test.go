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

	"workbench/internal/config"
	"workbench/internal/pkg/zentao"
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

func TestBuildTodoUnionSQL_PublishStage(t *testing.T) {
	sqlPublish, _ := buildTodoUnionSQL("user_a", TodoListReq{Stage: "publish"})
	if !strings.Contains(sqlPublish, "'demand' AS kind") {
		t.Errorf("expected demand in union for publish stage, got: %s", sqlPublish)
	}
	if !strings.Contains(sqlPublish, "waitdeliver") || !strings.Contains(sqlPublish, "released") {
		t.Errorf("expected publish stage SQL to check waitdeliver or released, got: %s", sqlPublish)
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

func TestBuildTodoFacetSQLIgnoresObjectTypeDimension(t *testing.T) {
	// 已选中 demand 时分面仍必须统计 task / bug，否则芯片计数归零后无法切回。
	facetSQL, args := buildTodoFacetSQL("user_a", TodoListReq{ObjectType: "demand"}, nil, "2026-09-05")
	for _, kind := range []string{"'demand' AS kind", "'task' AS kind", "'bug' AS kind"} {
		if !strings.Contains(facetSQL, kind) {
			t.Fatalf("expected facet union to keep %s, got: %s", kind, facetSQL)
		}
	}
	if !strings.Contains(facetSQL, "GROUP BY t.kind") {
		t.Fatalf("expected facet count to aggregate in SQL, got: %s", facetSQL)
	}
	if strings.Contains(facetSQL, "t.kind = 'demand'") {
		t.Fatalf("expected facet SQL to drop the object-domain tab condition, got: %s", facetSQL)
	}
	for _, arg := range args {
		if s, ok := arg.(string); ok && s != "user_a" && s != "2026-09-05" {
			t.Fatalf("unexpected literal argument %q in facet args %#v", s, args)
		}
	}

	// 其它维度仍与列表同口径：焦点、关系与关键词都必须进入分面 WHERE。
	facetFiltered, _ := buildTodoFacetSQL("user_a", TodoListReq{Focus: "overdue", Relation: RelationInCharge}, nil, "2026-09-05")
	if !strings.Contains(facetFiltered, "t.relation = '我负责'") || !strings.Contains(facetFiltered, "t.deadline_str <") {
		t.Fatalf("expected facet SQL to keep relation and focus conditions, got: %s", facetFiltered)
	}
}

func TestBuildTodoFacetsCoversOnlyWiredObjectTypes(t *testing.T) {
	facets := buildTodoFacets(map[string]int64{"demand": 7, "task": 2})
	if len(facets) != 9 {
		t.Fatalf("facets = %#v, want 9 facets", facets)
	}
	want := []TodoFacet{
		{Key: "approval", Label: "审批", Count: 0},
		{Key: "demand", Label: "业务需求", Count: 7},
		{Key: "story", Label: "研发需求", Count: 0},
		{Key: "task", Label: "任务", Count: 2},
		{Key: "bug", Label: "Bug", Count: 0},
		{Key: "risk", Label: "风险", Count: 0},
		{Key: "issue", Label: "问题", Count: 0},
		{Key: "todo", Label: "待办", Count: 0},
		{Key: "testtask", Label: "测试单", Count: 0},
	}
	for i, expected := range want {
		if facets[i] != expected {
			t.Fatalf("facets[%d] = %#v, want %#v", i, facets[i], expected)
		}
	}
	// 芯片 key 必须能通过 TodoListReq 校验，否则点击后是必然 422。
	for _, facet := range facets {
		req := TodoListReq{ObjectType: facet.Key}
		if errs := req.Validate(); len(errs) != 0 {
			t.Fatalf("facet key %q rejected by Validate: %#v", facet.Key, errs)
		}
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

	// Focus
	whereFocus, _ := buildTodoOuterWhere(TodoListReq{Focus: "today"}, nil, true, "2026-09-05")
	if !strings.Contains(whereFocus, "t.deadline_str = ?") {
		t.Errorf("expected focus today condition, got: %s", whereFocus)
	}
}

func TestBuildTodoUnionSQL_ApprovalType(t *testing.T) {
	// 审批场景过滤仅保留 approval，并且携带对应 objectType 条件
	sqlCharter, _ := buildTodoUnionSQL("user_a", TodoListReq{ApprovalType: "charter"})
	if !strings.Contains(sqlCharter, "'approval' AS kind") {
		t.Errorf("expected approval in union for approvalType charter, got: %s", sqlCharter)
	}
	if strings.Contains(sqlCharter, "'demand' AS kind") || strings.Contains(sqlCharter, "'task' AS kind") {
		t.Errorf("expected approvalType to exclude demand and task, got: %s", sqlCharter)
	}
	if !strings.Contains(sqlCharter, "ao.objectType = 'charter'") {
		t.Errorf("expected charter condition in approval query, got: %s", sqlCharter)
	}
}

func TestFormatTodoUnifiedItem_ApprovalURL(t *testing.T) {
	zentao.SetConfig(config.ZentaoConfig{URL: "http://zentao.test", RequestType: "PATH_INFO"})
	rowCharter := todoUnifiedRow{
		Kind:       "approval",
		ID:         2781,
		ObjectType: "charter",
		ObjectID:   152,
		ProjectID:  99,
	}
	itemCharter := formatTodoUnifiedItem(rowCharter, map[string]string{})
	if !strings.Contains(itemCharter.URL, "charter-view-99.html") && !strings.Contains(itemCharter.URL, "projectID=99") {
		t.Errorf("expected charter URL with projectID=99, got: %s", itemCharter.URL)
	}

	rowReview := todoUnifiedRow{
		Kind:       "approval",
		ID:         2782,
		ObjectType: "review",
		ObjectID:   304,
	}
	itemReview := formatTodoUnifiedItem(rowReview, map[string]string{})
	if !strings.Contains(itemReview.URL, "review-view-304.html") && !strings.Contains(itemReview.URL, "reviewID=304") {
		t.Errorf("expected review URL with reviewID=304, got: %s", itemReview.URL)
	}

	rowFallback := todoUnifiedRow{
		Kind: "approval",
		ID:   2783,
	}
	itemFallback := formatTodoUnifiedItem(rowFallback, map[string]string{})
	if !strings.Contains(itemFallback.URL, "approval-view-2783.html") && !strings.Contains(itemFallback.URL, "approvalID=2783") {
		t.Errorf("expected fallback approval URL with approval-view-2783.html or approvalID=2783, got: %s", itemFallback.URL)
	}
}
