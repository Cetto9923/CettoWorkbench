// =============================================================================
// 文件: internal/module/po/servicetodo_test.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的待办请求校验回归测试。
// 依赖: 无
// =============================================================================

package po

import (
	"strings"
	"testing"
)

func TestTodoListReqValidate_RejectsUnsupportedObjectTypes(t *testing.T) {
	// 验证未知类型返回不支持
	reqInvalid := TodoListReq{ObjectType: "unknown"}
	errs := reqInvalid.Validate()
	if len(errs) != 1 || errs[0].Field != "objectType" || !strings.Contains(errs[0].Message, "不支持的对象类型") {
		t.Fatalf("expected unsupported type error, got: %#v", errs)
	}

	// 验证全部已接入合法类型通过校验
	for _, valid := range []string{"", "all", "approval", "demand", "story", "task", "bug", "risk", "issue", "todo", "testtask"} {
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
