package po

import "testing"

func TestDemandFollowLifecycleBucket(t *testing.T) {
	cases := map[string]string{
		"draft":       string(FollowLifecycleClarifying),
		"wait":        string(FollowLifecycleClarifying),
		"active":      string(FollowLifecycleClarifying),
		"developing":  string(FollowLifecycleImplementing),
		"testing":     string(FollowLifecycleImplementing),
		"waitdeliver": string(FollowLifecycleImplementing),
		"released":    string(FollowLifecycleReleased),
		"closed":      string(FollowLifecycleClosed),
	}
	for status, want := range cases {
		if got := demandFollowLifecycleBucket(status); got != want {
			t.Fatalf("status=%s got=%s want=%s", status, got, want)
		}
	}
}

func TestFollowProgress(t *testing.T) {
	st, lb := followProgress("closed", "2020-01-01", "2026-09-10")
	if st != "done" || lb != "已完成" {
		t.Fatalf("closed => %s/%s", st, lb)
	}
	st, lb = followProgress("developing", "2020-01-01", "2026-09-10")
	if st != "delayed" || lb != "延期" {
		t.Fatalf("overdue => %s/%s", st, lb)
	}
	st, lb = followProgress("developing", "2099-01-01", "2026-09-10")
	if st != "normal" || lb != "正常" {
		t.Fatalf("future => %s/%s", st, lb)
	}
	st, lb = followProgress("developing", "", "2026-09-10")
	if st != "unknown" || lb != "—" {
		t.Fatalf("empty deadline => %s/%s", st, lb)
	}
}

func TestFollowScheduleSummary(t *testing.T) {
	if got := followScheduleSummary("", "", ""); got != "" {
		t.Fatalf("empty want blank, got %q", got)
	}
	got := followScheduleSummary("2026-01-01", "", "2026-02-01")
	if got != "开发完成 2026-01-01 · 截止 2026-02-01" {
		t.Fatalf("got %q", got)
	}
}

func TestFollowListReqValidateScopes(t *testing.T) {
	// 默认未传 scope 应规范化为 open
	reqDefault := FollowListReq{}
	if errs := reqDefault.Validate(); len(errs) != 0 {
		t.Fatalf("unexpected errs: %+v", errs)
	}
	if reqDefault.Scope != FollowScopeOpen {
		t.Fatalf("default scope want=%s got=%s", FollowScopeOpen, reqDefault.Scope)
	}

	req := FollowListReq{Scope: "open", Lifecycle: "implementing"}
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("unexpected errs: %+v", errs)
	}

	reqCompat := FollowListReq{Scope: "key_open", Lifecycle: "implementing"}
	if errs := reqCompat.Validate(); len(errs) != 0 {
		t.Fatalf("unexpected errs: %+v", errs)
	}

	reqInvalid := FollowListReq{Scope: "need_focus"}
	if errs := reqInvalid.Validate(); len(errs) == 0 {
		t.Fatal("expected invalid scope error")
	}
}

func TestProjectWeeklyListReqValidateScopes(t *testing.T) {
	// 默认未传 scope 应规范化为 mine
	reqDefault := ProjectWeeklyListReq{}
	if errs := reqDefault.Validate(); len(errs) != 0 {
		t.Fatalf("unexpected errs: %+v", errs)
	}
	if reqDefault.Scope != "mine" {
		t.Fatalf("default weekly scope want=mine got=%s", reqDefault.Scope)
	}

	validScopes := []string{"mine", "participated", "watched", "all"}
	for _, sc := range validScopes {
		req := ProjectWeeklyListReq{Scope: sc}
		if errs := req.Validate(); len(errs) != 0 {
			t.Fatalf("valid scope %s returned unexpected errs: %+v", sc, errs)
		}
	}

	reqInvalid := ProjectWeeklyListReq{Scope: "invalid_scope"}
	if errs := reqInvalid.Validate(); len(errs) == 0 {
		t.Fatal("expected invalid scope error for invalid_scope")
	}
}
