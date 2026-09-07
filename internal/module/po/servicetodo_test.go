// =============================================================================
// 文件: internal/module/po/servicetodo_test.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的待办筛选、汇总与排序回归测试。
// 依赖: 无
// =============================================================================

package po

import (
	"strings"
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
		{DisplayID: "2", Title: "第二条", Owner: "薛方舟"},
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

func TestTodoListReqValidate_RejectsUnsupportedObjectTypes(t *testing.T) {
	// 验证 story 待办数据源暂未接入 (WAIT DECISION)
	reqStory := TodoListReq{ObjectType: "story"}
	errs := reqStory.Validate()
	if len(errs) != 1 || errs[0].Field != "objectType" || !strings.Contains(errs[0].Message, "WAIT DECISION") {
		t.Fatalf("expected WAIT DECISION for story, got: %#v", errs)
	}

	// 验证其它占位类型返回明确错误，拒绝伪成功
	for _, unsupported := range []string{"approval", "testtask", "issue", "risk", "todo"} {
		req := TodoListReq{ObjectType: unsupported}
		errs := req.Validate()
		if len(errs) != 1 || errs[0].Field != "objectType" || !strings.Contains(errs[0].Message, "待办数据源暂未接入") {
			t.Fatalf("expected unsupported message for %s, got: %#v", unsupported, errs)
		}
	}

	// 验证未知类型返回不支持
	reqInvalid := TodoListReq{ObjectType: "unknown"}
	errs = reqInvalid.Validate()
	if len(errs) != 1 || errs[0].Field != "objectType" || !strings.Contains(errs[0].Message, "不支持的对象类型") {
		t.Fatalf("expected unsupported type error, got: %#v", errs)
	}

	// 验证合法类型通过校验
	for _, valid := range []string{"", "all", "demand", "task", "bug"} {
		req := TodoListReq{ObjectType: valid}
		if errs := req.Validate(); len(errs) != 0 {
			t.Fatalf("expected valid for %q, got: %#v", valid, errs)
		}
	}
}

func TestTodoListReqValidate_RejectsUnsupportedTabs(t *testing.T) {
	for _, unsupported := range []string{"approval", "risk", "personal"} {
		req := TodoListReq{Tab: TodoTab(unsupported)}
		errs := req.Validate()
		if len(errs) != 1 || errs[0].Field != "tab" || !strings.Contains(errs[0].Message, "待办数据源暂未接入") {
			t.Fatalf("expected unsupported message for tab %s, got: %#v", unsupported, errs)
		}
	}

	for _, valid := range []string{"", "all", "demand", "execution", "testing"} {
		req := TodoListReq{Tab: TodoTab(valid)}
		if errs := req.Validate(); len(errs) != 0 {
			t.Fatalf("expected valid for tab %q, got: %#v", valid, errs)
		}
	}
}

func TestTodoListReqValidate_AcceptsPublishStage(t *testing.T) {
	req := TodoListReq{Stage: "publish"}
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("expected publish stage to be valid, got: %#v", errs)
	}

	reqInvalid := TodoListReq{Stage: "invalid_stage"}
	if errs := reqInvalid.Validate(); len(errs) != 1 || errs[0].Field != "stage" {
		t.Fatalf("expected invalid stage error, got: %#v", errs)
	}
}
