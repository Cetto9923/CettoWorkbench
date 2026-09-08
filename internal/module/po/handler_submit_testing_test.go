package po

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"workbench/internal/config"
	"workbench/internal/constants"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/render"
)

func TestSubmitTestTemplateRendersZenTaoEntry(t *testing.T) {
	t.Chdir(submitTestRepoRoot(t))
	renderer, err := render.New(&config.Config{
		App:    config.App{Name: "Workbench", Env: "dev"},
		Layout: config.Layout{Nav: "sidebar"},
	}, false)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	render.SetDefault(renderer)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/demands/63314/submit-test", func(c *gin.Context) {
		render.Page(c, http.StatusOK, constants.TEMPLATE_PO_SUBMIT_TEST, gin.H{
			"Title":       "提测办理",
			"CurrentPath": "/demands/submit-test",
			"SubmitTest": &SubmitTestPageData{
				DemandID:  63314,
				Title:     "测试需求",
				Stage:     "提测",
				Eligible: true,
				ZentaoURL: "http://zentao.test/index.php?m=demand&f=view&demandID=63314",
				Units: []SubmitTestUnit{{
					ProductName:   "产品一",
					ExecutionName: "执行一",
					Stories:       []SubmitTestStory{{ID: 1, Title: "研发需求一"}},
				}},
			},
		})
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/demands/63314/submit-test", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, want := range []string{"po-workbench-shell", "#63314", "产品一", "提交提测", "在禅道中查看", "http://zentao.test/index.php?m=demand"} {
		if !strings.Contains(body, want) {
			t.Fatalf("rendered page missing %q", want)
		}
	}
	if strings.Contains(body, "submitTestForm") || strings.Contains(body, "确认提测") || strings.Contains(body, "fetch(\"/demands") {
		t.Fatal("workbench must not render a local submit form")
	}
}

func TestSubmitTestErrorStatusPreservesDetailAuthorization(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"not_found", errorx.New(errorx.ErrCodeNotFound, "需求不存在"), http.StatusNotFound},
		{"forbidden", errorx.New(errorx.ErrCodeForbidden, "无权查看"), http.StatusForbidden},
		{"invalid", errorx.New(errorx.ErrCodeInvalidParam, "需求 ID 无效"), http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := submitTestErrorStatus(tc.err)
			if got != tc.want {
				t.Fatalf("status = %d, want %d", got, tc.want)
			}
		})
	}
}

func submitTestRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "web", "templates")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("repository root with templates not found")
	return ""
}
