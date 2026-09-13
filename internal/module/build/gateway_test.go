package build

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"workbench/internal/pkg/zentao"
)

func TestSplitStoryIDCSV(t *testing.T) {
	got := splitStoryIDCSV(" 1, 2 ,2,, 3 ")
	want := []string{"1", "2", "3"}
	if len(got) != len(want) {
		t.Fatalf("splitStoryIDCSV = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitStoryIDCSV = %#v, want %#v", got, want)
		}
	}
	if ids := splitStoryIDCSV(" , "); len(ids) != 0 {
		t.Fatalf("blank csv should yield no id, got %#v", ids)
	}
}

// 关联/解除必须打到禅道原生控制层路径，且两者读取的 $_POST 字段名不同。
func TestPostBuildStoriesUsesControllerPathAndFieldNames(t *testing.T) {
	var (
		gotPath string
		gotForm url.Values
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/tokens") {
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "tok-build"})
			return
		}
		gotPath = r.URL.Path
		_ = r.ParseForm()
		gotForm = r.PostForm
		_, _ = io.WriteString(w, `{"result":"success","load":"/build-view-9-story.html"}`)
	}))
	t.Cleanup(srv.Close)

	client := zentao.NewClient(srv.URL)

	if err := linkBuildStories(context.Background(), client, linkBuildStoriesReq{BuildID: 9, Account: "u-gw", Stories: "1,2"}); err != nil {
		t.Fatalf("linkBuildStories: %v", err)
	}
	if gotPath != "/build-linkStory-9.html" {
		t.Fatalf("link path = %q, want /build-linkStory-9.html", gotPath)
	}
	if vals := gotForm["stories[]"]; len(vals) != 2 || vals[0] != "1" || vals[1] != "2" {
		t.Fatalf("link stories[] = %#v", vals)
	}

	if err := unlinkBuildStories(context.Background(), client, linkBuildStoriesReq{BuildID: 9, Account: "u-gw", Stories: "1"}); err != nil {
		t.Fatalf("unlinkBuildStories: %v", err)
	}
	if gotPath != "/build-batchUnlinkStory-9.html" {
		t.Fatalf("unlink path = %q, want /build-batchUnlinkStory-9.html", gotPath)
	}
	if vals := gotForm["storyIdList[]"]; len(vals) != 1 || vals[0] != "1" {
		t.Fatalf("unlink storyIdList[] = %#v", vals)
	}
}

func TestPostBuildStoriesValidatesInput(t *testing.T) {
	client := zentao.NewClient("http://127.0.0.1:1")
	cases := []struct {
		name string
		req  linkBuildStoriesReq
	}{
		{"build id zero", linkBuildStoriesReq{BuildID: 0, Account: "u", Stories: "1"}},
		{"blank account", linkBuildStoriesReq{BuildID: 1, Account: "   ", Stories: "1"}},
		{"blank stories", linkBuildStoriesReq{BuildID: 1, Account: "u", Stories: " , "}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := linkBuildStories(context.Background(), client, tc.req); err == nil {
				t.Fatal("expected validation error before remote call")
			}
		})
	}
}
