// =============================================================================
// 文件: internal/module/po/reponotice_test.go
// 模块: PO 工作台
// 类型: action
// 职责: 验证通知分类和筛选请求的核心口径。
// 依赖: 无
// =============================================================================

package po

import "testing"

func TestClassifyNotice(t *testing.T) {
	cases := []struct{ objectType, action, want string }{
		{"demand", "reviewed", "approval"},
		{"task", "reminded", "reminder"},
		{"story", "assigned", "collaboration"},
		{"bug", "bugconfirmed", "risk"},
		{"system", "", "system"},
		{"demand", "activated", "business"},
	}
	for _, testCase := range cases {
		if actual := classifyNotice(testCase.objectType, testCase.action); actual != testCase.want {
			t.Fatalf("classifyNotice(%q, %q) = %q, want %q", testCase.objectType, testCase.action, actual, testCase.want)
		}
	}
}

func TestNoticeListReqValidate(t *testing.T) {
	req := NoticeListReq{}
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("default request errors: %#v", errs)
	}
	if req.Category != "all" || req.Page != 1 || req.PageSize != 20 {
		t.Fatalf("defaults = %#v", req)
	}
	invalid := NoticeListReq{Category: "guessed"}
	if errs := invalid.Validate(); len(errs) != 1 || errs[0].Field != "category" {
		t.Fatalf("invalid category errors: %#v", errs)
	}
}

func TestFilterNoticeRowsTreatsApprovalAsAnObjectFilter(t *testing.T) {
	rows := []noticeRow{{ObjectType: "demand", ActionCode: "reviewed"}, {ObjectType: "demand", ActionCode: "activated"}}
	filtered := filterNoticeRows(rows, NoticeListReq{ObjectType: "approval"})
	if len(filtered) != 1 || filtered[0].ActionCode != "reviewed" {
		t.Fatalf("approval filter = %#v", filtered)
	}
}
