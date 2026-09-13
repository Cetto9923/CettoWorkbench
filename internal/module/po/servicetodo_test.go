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

func TestTodoListReqValidate_ApprovalType(t *testing.T) {
	for _, valid := range []string{"", "all", "charter", "buildguideline", "planchange", "review", "reviewchange", "reviewbymanager"} {
		req := TodoListReq{ApprovalType: valid}
		if errs := req.Validate(); len(errs) != 0 {
			t.Fatalf("expected valid approvalType for %q, got: %#v", valid, errs)
		}
	}

	reqInvalid := TodoListReq{ApprovalType: "invalid_type"}
	if errs := reqInvalid.Validate(); len(errs) != 1 || errs[0].Field != "approvalType" {
		t.Fatalf("expected invalid approvalType error, got: %#v", errs)
	}
}
