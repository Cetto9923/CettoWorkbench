// =============================================================================
// 文件: internal/pkg/render/render_test.go
// 模块: 基础设施
// 类型: infra
// 职责: 验证 CSRF Token 在模板渲染中的签发与传递 (Phase X0 / X1)。
// =============================================================================

package render

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/justinas/nosurf"

	"workbench/internal/config"
)

func findRepoRoot(t *testing.T) string {
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
	t.Fatalf("cannot find repo root containing web/templates")
	return ""
}

func TestRender_TokenRenderedAndSent(t *testing.T) {
	root := findRepoRoot(t)
	t.Chdir(root)

	cfg := &config.Config{
		App:    config.App{Name: "Workbench", Env: "dev"},
		Layout: config.Layout{Nav: "sidebar"},
	}
	r, err := New(cfg, false)
	if err != nil {
		t.Fatalf("New renderer failed: %v", err)
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("renderer", r)
	})

	engine.GET("/login", func(c *gin.Context) {
		Page(c, http.StatusOK, "auth/login", gin.H{"Title": "登录"})
	})

	handler := nosurf.New(engine)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()

	// 1. Verify meta tag contains non-empty csrf-token
	metaPattern := regexp.MustCompile(`<meta\s+name="csrf-token"\s+content="([^"]+)"`)
	metaMatches := metaPattern.FindStringSubmatch(body)
	if len(metaMatches) < 2 || strings.TrimSpace(metaMatches[1]) == "" {
		t.Fatalf("expected non-empty csrf-token in meta tag, got: %s", body)
	}
	token := metaMatches[1]

	// 2. Verify form hidden input contains the matching csrf_token
	inputPattern := regexp.MustCompile(`<input\s+type="hidden"\s+name="csrf_token"\s+value="([^"]+)"`)
	inputMatches := inputPattern.FindStringSubmatch(body)
	if len(inputMatches) < 2 || inputMatches[1] != token {
		t.Fatalf("expected form csrf_token value %q to match meta token, got %v", token, inputMatches)
	}
}

func TestRender_MissingMetaNoSilentSuccess(t *testing.T) {
	root := findRepoRoot(t)
	t.Chdir(root)

	cfg := &config.Config{
		App:    config.App{Name: "Workbench", Env: "dev"},
		Layout: config.Layout{Nav: "sidebar"},
	}
	r, err := New(cfg, false)
	if err != nil {
		t.Fatalf("New renderer failed: %v", err)
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("renderer", r)
	})

	engine.GET("/login-no-csrf", func(c *gin.Context) {
		Page(c, http.StatusOK, "auth/login", gin.H{"Title": "登录"})
	})

	// Direct engine request without nosurf middleware: server does NOT issue token
	req := httptest.NewRequest(http.MethodGet, "/login-no-csrf", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	// Meta content must be empty when no token was issued
	if !strings.Contains(body, `<meta name="csrf-token" content="">`) {
		t.Fatalf("expected empty csrf-token meta when unissued, got body snippet: %s", body[:min(len(body), 500)])
	}
	if !strings.Contains(body, `<input type="hidden" name="csrf_token" value=""`) {
		t.Fatalf("expected empty form csrf_token input when unissued")
	}

	// Verify nosurf rejects mutation without valid token
	postHandler := nosurf.New(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("MUTATION_SUCCESS"))
	}))

	postReq := httptest.NewRequest(http.MethodPost, "/mutate", strings.NewReader("data=test"))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postRec := httptest.NewRecorder()
	postHandler.ServeHTTP(postRec, postReq)

	if postRec.Code == http.StatusOK {
		t.Fatalf("expected failure on mutation without token, but got 200 OK (silent bypass)")
	}
}

func TestRender_NilRequestSafety(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = nil
	data := gin.H{}
	r := &Renderer{}
	r.enrichData(c, "test", data)
	if data["CSRFToken"] != "" {
		t.Fatalf("expected empty CSRFToken on nil request, got %v", data["CSRFToken"])
	}
}
