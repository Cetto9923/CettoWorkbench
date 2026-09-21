// =============================================================================
// 文件: internal/module/po/repodone_test.go
// 模块: PO 工作台
// 类型: action
// 职责: 验证我的已办正式动作注册口径。
// 依赖: 无
// =============================================================================

package po

import (
	"strings"
	"testing"

	"workbench/internal/config"
	"workbench/internal/pkg/zentao"
)

func TestFormalDoneActionsExcludeNoise(t *testing.T) {
	for _, key := range []string{"user:login", "demand:edit", "task:comment"} {
		if _, ok := formalDoneActions[key]; ok {
			t.Fatalf("noise action %q must not enter formal done", key)
		}
	}
	for _, key := range []string{"demand:clarify", "task:finished", "bug:resolved"} {
		if _, ok := formalDoneActions[key]; !ok {
			t.Fatalf("formal action %q missing", key)
		}
	}
}

func TestDoneListReqAllowsApprovalTab(t *testing.T) {
	req := DoneListReq{Tab: DoneTabApproval}
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("approval tab should be valid: %#v", errs)
	}
	if req.Tab != DoneTabApproval {
		t.Fatalf("unexpected tab after validation: %q", req.Tab)
	}
}

func TestApprovalDoneScopeContainsOnlyReviewActions(t *testing.T) {
	scope := buildApprovalDoneScopeSQL()
	for _, action := range []string{"reviewed", "reviewchange", "reviewbymanager", "approvalreview"} {
		if !strings.Contains(scope, action) {
			t.Fatalf("approval scope missing %q", action)
		}
	}
	for _, shadow := range []string{"reviewpassed", "reviewrejected", "demand:edit"} {
		if strings.Contains(scope, shadow) {
			t.Fatalf("approval scope must not include shadow/edit action %q", shadow)
		}
	}
}

func TestFormalDoneActionsExcludeDuplicateReviewPassed(t *testing.T) {
	for _, key := range []string{"demand:reviewpassed", "demand:reviewrejected", "story:reviewpassed", "story:reviewrejected"} {
		if _, ok := formalDoneActions[key]; ok {
			t.Fatalf("duplicate shadow review action %q must not enter formal done", key)
		}
	}
	for _, key := range []string{"demand:reviewed", "demand:reviewchange", "demand:reviewbymanager"} {
		if _, ok := formalDoneActions[key]; !ok {
			t.Fatalf("human review action %q missing", key)
		}
	}
}

func TestResolveDoneActionResult(t *testing.T) {
	cases := []struct {
		action, objectType, extra, def string
		wantCode, wantText             string
	}{
		{"reviewed", "demand", "pass", "done", "approved", "已通过"},
		{"reviewed", "demand", "refuse", "done", "rejected", "已驳回"},
		{"reviewed", "story", "Pass", "done", "approved", "已通过"},
		{"reviewed", "story", "Reject,cancel", "done", "rejected", "已驳回"},
		{"reviewchange", "demand", "pass", "done", "approved", "已通过"},
		{"reviewchange", "demand", "refuse", "done", "rejected", "已驳回"},
		{"reviewed", "demand", "", "done", "done", "已完成"},
		{"dispatch", "demand", "", "done", "done", "已完成"},
	}
	for _, tc := range cases {
		code, text := resolveDoneActionResult(tc.action, tc.objectType, tc.extra, tc.def)
		if code != tc.wantCode || text != tc.wantText {
			t.Errorf("resolveDoneActionResult(%s, %s, %s) = (%s, %s), want (%s, %s)",
				tc.action, tc.objectType, tc.extra, code, text, tc.wantCode, tc.wantText)
		}
	}
}

func TestApprovalDoneScopeCoversProjectAndCaseReviews(t *testing.T) {
	scope := buildApprovalDoneScopeSQL()
	for _, objectType := range []string{"charter", "planchange", "buildguideline", "review", "case"} {
		if !strings.Contains(scope, "a.objectType = '"+objectType+"'") {
			t.Fatalf("approval scope missing %q: %s", objectType, scope)
		}
	}
}

func TestBuildDoneObjectScopeApprovalDoesNotFallBackToObjectType(t *testing.T) {
	scope, args := buildDoneObjectScopeSQL(DoneTabApproval, "")
	if scope != buildApprovalDoneScopeSQL() || len(args) != 0 {
		t.Fatalf("approval scope = (%q, %#v)", scope, args)
	}
}

func TestObjectViewURLZeroIDReturnsEmpty(t *testing.T) {
	if url := objectViewURL("demand", 0); url != "" {
		t.Fatalf("objectViewURL with id=0 should return empty, got %q", url)
	}
	if url := objectViewURL("story", 0); url != "" {
		t.Fatalf("objectViewURL with id=0 should return empty, got %q", url)
	}
	if url := objectViewURL("task", 0); url != "" {
		t.Fatalf("objectViewURL with id=0 should return empty, got %q", url)
	}
}

func TestObjectViewURLValidID(t *testing.T) {
	zentao.SetConfig(config.ZentaoConfig{URL: "http://zentao.test"})
	url := objectViewURL("demand", 42)
	if !strings.Contains(url, "demand") || !strings.Contains(url, "42") {
		t.Fatalf("unexpected demand url: %q", url)
	}
}

// 章程详情必须有所属项目上下文，不能用 charter objectID 伪造详情链接。
func TestObjectViewURLCharterRequiresProject(t *testing.T) {
	zentao.SetConfig(config.ZentaoConfig{URL: "http://zentao.test"})
	url := objectViewURL("charter", 42)
	if url != "" {
		t.Fatalf("charter without project must not emit a URL, got %q", url)
	}
}

func TestObjectViewURLBuildguidelineRequiresProject(t *testing.T) {
	zentao.SetConfig(config.ZentaoConfig{URL: "http://zentao.test"})
	url := objectViewURL("buildguideline", 42)
	if url != "" {
		t.Fatalf("buildguideline without project must not emit a URL, got %q", url)
	}
}

// 验证 buildguideline 在对象自身 ID 缺失时回退到 projectID。
func TestObjectViewURLWithProjectBuildguidelineFallback(t *testing.T) {
	zentao.SetConfig(config.ZentaoConfig{URL: "http://zentao.test"})
	url := objectViewURLWithProject("buildguideline", 0, 7)
	if !strings.Contains(url, "m=buildguideline") {
		t.Fatalf("buildguideline fallback url must use m=buildguideline, got %q", url)
	}
	if !strings.Contains(url, "projectID=7") {
		t.Fatalf("buildguideline fallback url must use projectID=7, got %q", url)
	}
	if strings.Contains(url, "id=0") {
		t.Fatalf("buildguideline fallback url must not carry id=0, got %q", url)
	}
}

// 验证 charter 在对象自身 ID 缺失时回退到 projectID。
func TestObjectViewURLWithProjectCharterFallback(t *testing.T) {
	zentao.SetConfig(config.ZentaoConfig{URL: "http://zentao.test"})
	url := objectViewURLWithProject("charter", 0, 12)
	if !strings.Contains(url, "m=charter") {
		t.Fatalf("charter fallback url must use m=charter, got %q", url)
	}
	if !strings.Contains(url, "projectID=12") {
		t.Fatalf("charter fallback url must use projectID=12, got %q", url)
	}
	if strings.Contains(url, "id=0") {
		t.Fatalf("charter fallback url must not carry id=0, got %q", url)
	}
}

// 验证 charter / buildguideline 在 objectID 与 projectID 都为 0 时返回空（不跳首页）。
func TestObjectViewURLCharterBuildguidelineZeroIDsReturnEmpty(t *testing.T) {
	zentao.SetConfig(config.ZentaoConfig{URL: "http://zentao.test"})
	if url := objectViewURL("charter", 0); url != "" {
		t.Fatalf("charter with id=0 must return empty, got %q", url)
	}
	if url := objectViewURL("buildguideline", 0); url != "" {
		t.Fatalf("buildguideline with id=0 must return empty, got %q", url)
	}
	if url := objectViewURLWithProject("charter", 0, 0); url != "" {
		t.Fatalf("charter with id=0 and projectID=0 must return empty, got %q", url)
	}
	if url := objectViewURLWithProject("buildguideline", 0, 0); url != "" {
		t.Fatalf("buildguideline with id=0 and projectID=0 must return empty, got %q", url)
	}
}

// 验证 planchange 使用对象自身 ID 生成 URL（与禅道 max5 一致）。
func TestObjectViewURLPlanchangeUsesObjectID(t *testing.T) {
	zentao.SetConfig(config.ZentaoConfig{URL: "http://zentao.test"})
	url := objectViewURL("planchange", 21)
	if !strings.Contains(url, "m=planchange") || !strings.Contains(url, "f=view") {
		t.Fatalf("planchange url must use m=planchange&f=view, got %q", url)
	}
	if !strings.Contains(url, "ID=21") {
		t.Fatalf("planchange url must use ID={ID}, got %q", url)
	}
}

// 验证 review / case 使用对象自身 ID 生成 URL。
func TestObjectViewURLReviewAndCaseUseObjectID(t *testing.T) {
	zentao.SetConfig(config.ZentaoConfig{URL: "http://zentao.test"})
	reviewURL := objectViewURL("review", 9)
	if !strings.Contains(reviewURL, "m=review") {
		t.Fatalf("review url must use m=review, got %q", reviewURL)
	}
	if !strings.Contains(reviewURL, "reviewID=9") {
		t.Fatalf("review url must use reviewID=9, got %q", reviewURL)
	}
	caseURL := objectViewURL("case", 5)
	if !strings.Contains(caseURL, "m=case") {
		t.Fatalf("case url must use m=case, got %q", caseURL)
	}
	if !strings.Contains(caseURL, "caseID=5") {
		t.Fatalf("case url must use caseID=5, got %q", caseURL)
	}
}

// 验证 zentao 包内新增 helper：CharterViewURL/BuildguidelineViewURL/
// PlanchangeViewURL/ReviewViewURL/CaseViewURL 的 URL 形态与 id 回退。
func TestZentaoApprovalHelpersURLShape(t *testing.T) {
	zentao.SetConfig(config.ZentaoConfig{URL: "http://zentao.test"})
	cases := []struct {
		name        string
		got         string
		wantSubstrs []string
	}{
		{
			name:        "CharterViewURL_uses_projectID",
			got:         zentao.CharterViewURL(11, 99),
			wantSubstrs: []string{"m=charter", "f=view", "projectID=99"},
		},
		{
			name:        "CharterViewURL_fallback_projectID",
			got:         zentao.CharterViewURL(0, 99),
			wantSubstrs: []string{"m=charter", "f=view", "projectID=99"},
		},
		{
			name:        "CharterViewURL_without_project_returns_empty",
			got:         zentao.CharterViewURL(0, 0),
			wantSubstrs: nil,
		},
		{
			name:        "BuildguidelineViewURL_uses_projectID",
			got:         zentao.BuildguidelineViewURL(22, 88),
			wantSubstrs: []string{"m=buildguideline", "f=view", "projectID=88"},
		},
		{
			name:        "BuildguidelineViewURL_fallback_projectID",
			got:         zentao.BuildguidelineViewURL(0, 88),
			wantSubstrs: []string{"m=buildguideline", "f=view", "projectID=88"},
		},
		{
			name:        "BuildguidelineViewURL_without_project_returns_empty",
			got:         zentao.BuildguidelineViewURL(0, 0),
			wantSubstrs: nil,
		},
		{
			name:        "PlanchangeViewURL_uses_ID",
			got:         zentao.PlanchangeViewURL(33),
			wantSubstrs: []string{"m=planchange", "f=view", "ID=33"},
		},
		{
			name:        "ReviewViewURL_uses_reviewID",
			got:         zentao.ReviewViewURL(44),
			wantSubstrs: []string{"m=review", "f=view", "reviewID=44"},
		},
		{
			name:        "CaseViewURL_uses_caseID",
			got:         zentao.CaseViewURL(55),
			wantSubstrs: []string{"m=case", "f=view", "caseID=55"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantSubstrs == nil {
				if tc.got != "" {
					t.Fatalf("expected empty, got %q", tc.got)
				}
				return
			}
			if tc.got == "" {
				t.Fatalf("expected non-empty url")
			}
			for _, s := range tc.wantSubstrs {
				if !strings.Contains(tc.got, s) {
					t.Fatalf("url %q missing %q", tc.got, s)
				}
			}
		})
	}
}

// 验证 charter 走 objectViewURLWithProject 时使用所属项目 ID，避免白屏。
func TestObjectViewURLWithProjectCharterUsesProjectID(t *testing.T) {
	zentao.SetConfig(config.ZentaoConfig{URL: "http://zentao.test"})
	url := objectViewURLWithProject("charter", 7, 99)
	if !strings.Contains(url, "projectID=99") {
		t.Fatalf("charter must use project id=99, got %q", url)
	}
	if strings.Contains(url, "id=7") {
		t.Fatalf("charter must not use charter object id=7, got %q", url)
	}
}
