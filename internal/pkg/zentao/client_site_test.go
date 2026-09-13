package zentao

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// 站点控制层动作必须打到站点页而不是 api.php/v1，并自带同源 Referer、Token 与表单编码。
func TestDoSiteFormSendsTokenRefererAndFormBody(t *testing.T) {
	clearUserToken("u-site")
	var (
		gotPath    string
		gotReferer string
		gotToken   string
		gotCT      string
		gotForm    url.Values
		sawAPI     bool
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/tokens") {
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "tok-site"})
			return
		}
		if strings.Contains(r.URL.Path, "api.php") {
			sawAPI = true
		}
		gotPath = r.URL.Path
		gotReferer = r.Header.Get("Referer")
		gotToken = r.Header.Get("Token")
		gotCT = r.Header.Get("Content-Type")
		_ = r.ParseForm()
		gotForm = r.PostForm
		_, _ = io.WriteString(w, `{"result":"success","load":"/build-view-1-story.html"}`)
	}))
	t.Cleanup(srv.Close)

	form := url.Values{}
	form.Add("stories[]", "1")
	form.Add("stories[]", "2")
	c := NewClient(srv.URL)
	if err := c.DoSiteForm(context.Background(), "u-site", http.MethodPost, "/build-linkStory-9.html", form); err != nil {
		t.Fatalf("DoSiteForm: %v", err)
	}

	if sawAPI {
		t.Fatalf("站点动作不应打到 api.php，实际 path = %q", gotPath)
	}
	if gotPath != "/build-linkStory-9.html" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotReferer != srv.URL+"/" {
		t.Fatalf("Referer = %q, want 同源 %q", gotReferer, srv.URL+"/")
	}
	if gotToken != "tok-site" {
		t.Fatalf("Token = %q", gotToken)
	}
	if !strings.HasPrefix(gotCT, "application/x-www-form-urlencoded") {
		t.Fatalf("Content-Type = %q", gotCT)
	}
	if vals := gotForm["stories[]"]; len(vals) != 2 || vals[0] != "1" || vals[1] != "2" {
		t.Fatalf("stories[] = %#v", vals)
	}
}

// CSRF 被触发时站点返回 HTTP 200 + 页面 HTML，属于静默失败，必须报错而不是当成成功。
func TestValidateSiteActionResponseRejectsHTMLSilentFailure(t *testing.T) {
	body := []byte(`<style class="zin-page-css">.items-center{align-items:center!important;}</style><div zui-create>页面</div>`)
	err := validateSiteActionResponse(body, http.StatusOK)
	if err == nil {
		t.Fatal("expected error for HTML body on HTTP 200")
	}
	if !errors.Is(err, ErrZentaoAPIError) || !strings.Contains(err.Error(), "拒绝") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSiteActionResponseAcceptsControllerSuccess(t *testing.T) {
	if err := validateSiteActionResponse([]byte(`{"result":"success","load":"/build-view-1-story.html"}`), http.StatusOK); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	err := validateSiteActionResponse([]byte(`{"result":"fail","message":"保存失败"}`), http.StatusOK)
	if err == nil || !strings.Contains(err.Error(), "保存失败") {
		t.Fatalf("expected fail surfaced, got %v", err)
	}
}

func TestLooksLikeHTML(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"empty", "", false},
		{"action json", `{"result":"success"}`, false},
		{"style tag", `<style class="x">`, true},
		{"doctype", `<!DOCTYPE html><html>`, true},
		{"leading whitespace div", "  \n <div>x</div>", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := looksLikeHTML([]byte(tc.in)); got != tc.want {
				t.Fatalf("looksLikeHTML(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
