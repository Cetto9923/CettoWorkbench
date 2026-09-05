// =============================================================================
// 文件: internal/server/http_chain_test.go
// 模块: 基础设施
// 类型: infra
// 职责: 验证生产 HTTP 链 (SCS -> CSRF -> Gin) 的全链路集成行为 (Phase X1)。
// =============================================================================

package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/justinas/nosurf"
	"go.uber.org/zap"

	"workbench/internal/config"
	ratelimitpkg "workbench/internal/pkg/ratelimit"
)

func newTestServer(t *testing.T, cookieSecure bool) *Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	_, source, _, _ := runtime.Caller(0)
	t.Chdir(filepath.Join(filepath.Dir(source), "../.."))

	cfg := &config.Config{
		App: config.App{
			Name: "Workbench",
			Env:  "dev",
			Addr: ":0",
		},
		Session: config.Session{
			CookieName:    "workbench_session",
			LifetimeHours: 24,
			CookieSecure:  cookieSecure,
		},
		RateLimit: config.RateLimit{
			GlobalRPS: 1000,
		},
	}

	sessionMgr := scs.New()
	sessionMgr.Cookie.Name = cfg.Session.CookieName
	sessionMgr.Cookie.Secure = cookieSecure

	srv := New(cfg, zap.NewNop(), nil, sessionMgr, ratelimitpkg.New(100, 200), nil, RouteDeps{})
	srv.Setup() // Test routes must inherit the actual Gin middleware too.
	return srv
}

// TestHTTPChain_ProductionChainMissingOrWrongToken 验证从实际构造 handler 发送 POST，
// 缺失或错误 token 均返回 403 且无业务处理执行。
func TestHTTPChain_ProductionChainMissingOrWrongToken(t *testing.T) {
	srv := newTestServer(t, false)

	businessExecuted := false
	srv.engine.POST("/test/mutate", func(c *gin.Context) {
		businessExecuted = true
		c.String(http.StatusOK, "BUSINESS_OK")
	})

	handler := srv.BuildHandler()

	// 1. Missing token
	req1 := httptest.NewRequest(http.MethodPost, "/test/mutate", strings.NewReader("name=value"))
	req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 for missing token, got %d", rec1.Code)
	}
	if businessExecuted {
		t.Fatalf("business logic MUST NOT be executed when token is missing")
	}
	if strings.Contains(rec1.Body.String(), "BUSINESS_OK") {
		t.Fatalf("unexpected business response in body: %s", rec1.Body.String())
	}

	// 2. Wrong token
	req2 := httptest.NewRequest(http.MethodPost, "/test/mutate", strings.NewReader("name=value"))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.Header.Set("Sec-Fetch-Site", "same-origin")
	req2.Header.Set("X-CSRF-Token", "wrong-csrf-token-12345678901234567890123456789012")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 for wrong token, got %d", rec2.Code)
	}
	if businessExecuted {
		t.Fatalf("business logic MUST NOT be executed when token is wrong")
	}
}

// TestHTTPChain_ValidTokenRoundtrip 验证 GET 获取真实 cookie/token 后提交 POST 成功。
func TestHTTPChain_ValidTokenRoundtrip(t *testing.T) {
	srv := newTestServer(t, false)

	srv.engine.GET("/test/page", func(c *gin.Context) {
		token := nosurf.Token(c.Request)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, `<meta name="csrf-token" content="`+token+`">`)
	})

	srv.engine.POST("/test/action", func(c *gin.Context) {
		c.String(http.StatusOK, "ACTION_SUCCESS")
	})

	handler := srv.BuildHandler()

	// Step 1: GET page to obtain CSRF cookie and token
	getReq := httptest.NewRequest(http.MethodGet, "/test/page", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("GET /test/page failed: code=%d", getRec.Code)
	}

	cookies := getRec.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			csrfCookie = c
		}
	}
	if csrfCookie == nil {
		t.Fatalf("expected csrf_token cookie in GET response")
	}

	metaPattern := regexp.MustCompile(`<meta\s+name="csrf-token"\s+content="([^"]+)"`)
	matches := metaPattern.FindStringSubmatch(getRec.Body.String())
	if len(matches) < 2 || matches[1] == "" {
		t.Fatalf("failed to extract csrf-token from page: %s", getRec.Body.String())
	}
	token := matches[1]

	// Step 2: POST with valid X-CSRF-Token header
	postReq := httptest.NewRequest(http.MethodPost, "/test/action", strings.NewReader("k=v"))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("Sec-Fetch-Site", "same-origin")
	postReq.Header.Set("X-CSRF-Token", token)
	postReq.AddCookie(csrfCookie)

	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("POST with valid header token failed: code=%d, body=%s", postRec.Code, postRec.Body.String())
	}
	if postRec.Body.String() != "ACTION_SUCCESS" {
		t.Fatalf("expected ACTION_SUCCESS, got %s", postRec.Body.String())
	}

	// Step 3: POST with valid form field
	formValues := url.Values{}
	formValues.Set("csrf_token", token)
	formValues.Set("foo", "bar")
	formReq := httptest.NewRequest(http.MethodPost, "/test/action", strings.NewReader(formValues.Encode()))
	formReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	formReq.Header.Set("Sec-Fetch-Site", "same-origin")
	formReq.AddCookie(csrfCookie)

	formRec := httptest.NewRecorder()
	handler.ServeHTTP(formRec, formReq)

	if formRec.Code != http.StatusOK {
		t.Fatalf("POST with valid form token failed: code=%d, body=%s", formRec.Code, formRec.Body.String())
	}
	if formRec.Body.String() != "ACTION_SUCCESS" {
		t.Fatalf("expected ACTION_SUCCESS, got %s", formRec.Body.String())
	}
}

// TestHTTPChain_LoginCookie303 验证合法提交返回 303 且外层 SCS 成功持久化 session cookie。
func TestHTTPChain_LoginCookie303(t *testing.T) {
	srv := newTestServer(t, false)

	srv.engine.GET("/login", func(c *gin.Context) {
		token := nosurf.Token(c.Request)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, `<meta name="csrf-token" content="`+token+`">`)
	})

	srv.engine.POST("/login", func(c *gin.Context) {
		srv.sessionMgr.Put(c.Request.Context(), "userID", int64(42))
		c.Redirect(http.StatusSeeOther, "/dashboard")
	})

	handler := srv.BuildHandler()

	// GET to get CSRF token
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, httptest.NewRequest(http.MethodGet, "/login", nil))
	csrfCookies := getRec.Result().Cookies()

	metaPattern := regexp.MustCompile(`<meta\s+name="csrf-token"\s+content="([^"]+)"`)
	matches := metaPattern.FindStringSubmatch(getRec.Body.String())
	if len(matches) < 2 {
		t.Fatalf("failed to extract csrf-token")
	}
	token := matches[1]

	// POST login with token
	postReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("username=admin&password=pw"))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("Sec-Fetch-Site", "same-origin")
	postReq.Header.Set("X-CSRF-Token", token)
	for _, c := range csrfCookies {
		postReq.AddCookie(c)
	}

	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303 See Other, got %d", postRec.Code)
	}
	if loc := postRec.Header().Get("Location"); loc != "/dashboard" {
		t.Fatalf("expected Location /dashboard, got %q", loc)
	}

	// Verify session cookie was saved
	hasSessionCookie := false
	for _, c := range postRec.Result().Cookies() {
		if c.Name == "workbench_session" && c.Value != "" {
			hasSessionCookie = true
			break
		}
	}
	if !hasSessionCookie {
		t.Fatalf("expected workbench_session cookie in 303 response Set-Cookie, got: %v", postRec.Header().Values("Set-Cookie"))
	}
}

// TestHTTPChain_AllUnsafeMethods 验证 4 种不安全方法无 token 均被拦截为 403。
func TestHTTPChain_AllUnsafeMethods(t *testing.T) {
	srv := newTestServer(t, false)

	executedMethod := ""
	recordExecution := func(c *gin.Context) {
		executedMethod = c.Request.Method
		c.String(http.StatusOK, "OK")
	}

	srv.engine.POST("/test/resource", recordExecution)
	srv.engine.PUT("/test/resource", recordExecution)
	srv.engine.PATCH("/test/resource", recordExecution)
	srv.engine.DELETE("/test/resource", recordExecution)
	srv.engine.GET("/test/resource", recordExecution)
	srv.engine.HEAD("/test/resource", recordExecution)

	handler := srv.BuildHandler()

	unsafeMethods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for _, method := range unsafeMethods {
		t.Run(method+"_403", func(t *testing.T) {
			executedMethod = ""
			req := httptest.NewRequest(method, "/test/resource", strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("expected status 403 for %s, got %d", method, rec.Code)
			}
			if executedMethod != "" {
				t.Fatalf("business handler executed for %s without token", method)
			}
		})
	}

	// Safe methods pass
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		t.Run(method+"_200", func(t *testing.T) {
			req := httptest.NewRequest(method, "/test/resource", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200 for %s, got %d", method, rec.Code)
			}
		})
	}
}

// TestHTTPChain_CrossOriginOrJSONFailure 验证 JSON/AJAX 拒绝时返回 403 JSON，不误判为 401。
func TestHTTPChain_CrossOriginOrJSONFailure(t *testing.T) {
	srv := newTestServer(t, false)

	srv.engine.POST("/api/endpoint", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	handler := srv.BuildHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/endpoint", strings.NewReader(`{"data":"123"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden, got %d (must not be 401)", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Fatalf("expected application/json, got %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"success":false`) || !strings.Contains(body, `"code":403`) || !strings.Contains(body, `"error":"CSRF 校验失败"`) {
		t.Fatalf("expected JSON 403 response envelope, got %s", body)
	}
}
