// =============================================================================
// 文件: internal/module/po/form_test.go
// 模块: PO 工作台
// 类型: action
// 职责: PO 工作台表单（参数解析 / 状态枚举 / 去重 key）单测。
// 依赖: 无
// =============================================================================

package po

import "testing"

func TestDemandsReqValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		status      string
		wantErr     bool
		wantErrMsg  string
		wantNormed  string
	}{
		{name: "empty", status: "", wantErr: true, wantErrMsg: "状态不能为空"},
		{name: "whitespace only", status: "   ", wantErr: true, wantErrMsg: "状态不能为空"},
		{name: "invalid status", status: "garbage", wantErr: true, wantErrMsg: "无效的价值流状态"},
		{name: "valid all", status: "all", wantErr: false, wantNormed: "all"},
		{name: "valid schedule", status: "schedule", wantErr: false, wantNormed: "schedule"},
		{name: "valid released", status: "released", wantErr: false, wantNormed: "released"},
		{name: "valid with leading/trailing space", status: "  clarify  ", wantErr: false, wantNormed: "clarify"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := &DemandsReq{Status: tc.status}
			errs := req.Validate()

			if !tc.wantErr {
				if len(errs) != 0 {
					t.Fatalf("unexpected errors: %+v", errs)
				}
				if req.Status != tc.wantNormed {
					t.Fatalf("status not normalized: got %q want %q", req.Status, tc.wantNormed)
				}
				return
			}

			if len(errs) == 0 {
				t.Fatalf("expected validation error, got nil")
			}
			found := false
			for _, e := range errs {
				if e.Message == tc.wantErrMsg {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected error containing %q, got %+v", tc.wantErrMsg, errs)
			}
		})
	}
}

func TestWorkItemKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		kind, id string
		want     string
	}{
		{kind: "demand", id: "123", want: "demand:123"},
		{kind: "story", id: "456", want: "story:456"},
		{kind: "", id: "1", want: ":1"},
	}

	for _, tc := range cases {
		if got := workItemKey(tc.kind, tc.id); got != tc.want {
			t.Errorf("workItemKey(%q,%q)=%q want %q", tc.kind, tc.id, got, tc.want)
		}
	}
}

func TestValueStreamLabelForStatus(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"accept":         "受理",
		"clarify":        "澄清",
		"schedule":       "排期",
		"developing":     "提测",
		"testing":        "联调测试",
		"waitacceptance": "验收",
		"acceptanced":    "发起交付",
		"waitdeliver":    "发布",
		"released":       "评价",
		"all":            "全部",
	}
	for status, want := range cases {
		if got := valueStreamLabelForStatus(status); got != want {
			t.Errorf("status=%q label=%q want %q", status, got, want)
		}
	}
	if got := valueStreamLabelForStatus("__unknown__"); got != "" {
		t.Errorf("unknown = %q want empty", got)
	}
}

func TestIsValidValueStreamStatus(t *testing.T) {
	t.Parallel()

	valid := []string{
		"all", "accept", "clarify", "schedule", "developing", "testing",
		"waitacceptance", "acceptanced", "waitdeliver", "released",
	}
	for _, s := range valid {
		if !isValidValueStreamStatus(s) {
			t.Errorf("%q should be valid", s)
		}
	}
	invalid := []string{"", "draft", "GARBAGE", "accepted"}
	for _, s := range invalid {
		if isValidValueStreamStatus(s) {
			t.Errorf("%q should be invalid", s)
		}
	}
}
