// =============================================================================
// 文件: internal/module/po/servicetodo_test.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的待办筛选、汇总与排序回归测试。
// 依赖: 无
// =============================================================================

package po

import (
	"testing"
	"time"
)

func TestFilterTodoDimensionsKeepsSevenDimensionsAsAND(t *testing.T) {
	items := []TodoItem{
		{ID: 1, Kind: "demand", Relation: "我负责", Responsibility: "待我处理"},
		{ID: 2, Kind: "demand", Relation: "我配合", Responsibility: "待我跟进"},
		{ID: 3, Kind: "task", Relation: "我负责", Responsibility: "待我处理"},
	}
	req := TodoListReq{
		Action:         TodoActionReview,
		Stage:          "accept",
		Relation:       RelationInCharge,
		Responsibility: ResponsibilityMyAction,
	}

	got := filterTodoDimensions(items, req)
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("filtered items = %#v, want only demand 1", got)
	}
}

func TestSortTodoItemsUsesPriorityDeadlineAndID(t *testing.T) {
	items := []TodoItem{
		{ID: 1, Priority: "P2", Deadline: "2026-09-01"},
		{ID: 2, Priority: "P1", Deadline: ""},
		{ID: 3, Priority: "P1", Deadline: "2026-09-03"},
		{ID: 4, Priority: "P1", Deadline: "2026-09-03"},
	}

	sortTodoItems(items)
	want := []int64{4, 3, 2, 1}
	for i, id := range want {
		if items[i].ID != id {
			t.Fatalf("items[%d].ID = %d, want %d", i, items[i].ID, id)
		}
	}
}

func TestFilterTodoKeywordIncludesOwner(t *testing.T) {
	items := []TodoItem{
		{DisplayID: "US1", Title: "第一条", Owner: "王慧贤"},
		{DisplayID: "TASK-2", Title: "第二条", Owner: "薛方舟"},
	}
	got := filterTodoKeyword(items, "王慧贤")
	if len(got) != 1 || got[0].DisplayID != "US1" {
		t.Fatalf("owner keyword result=%+v, want US1", got)
	}
}

func TestTodoSummaryAndFocusUseSameDeadlineRule(t *testing.T) {
	now := time.Now()
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	items := []TodoItem{
		{ID: 1, Deadline: yesterday, Priority: "P1"},
		{ID: 2, Deadline: today, Blocked: true},
		{ID: 3, Deadline: ""},
	}

	summary := summarizeTodoItems(items)
	if summary.Pending != 3 || summary.Today != 1 || summary.Overdue != 1 || summary.Blocked != 1 || summary.P1 != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	if got := filterTodoFocus(items, "overdue", now); len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("overdue items = %#v, want item 1", got)
	}
}
