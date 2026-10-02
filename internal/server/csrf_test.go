// =============================================================================
// 文件: internal/server/csrf_test.go
// 模块: HTTP 服务
// 类型: test
// 职责: 通过真实排期路由验证默认挂载的 CSRF 防护。
// =============================================================================

package server

import (
	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/justinas/nosurf"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"workbench/internal/config"
	"workbench/internal/module/schedule"
)

func TestProductionSchedulingRouteRequiresCSRF(t *testing.T) {
	mgr := scs.New()
	srv := New(&config.Config{}, zap.NewNop(), nil, mgr, nil, nil, RouteDeps{SessionMgr: mgr, ScheduleHandler: schedule.NewHandler(nil, zap.NewNop(), nil, "")})
	var validToken string
	srv.engine.Use(func(c *gin.Context) { validToken = nosurf.Token(c.Request); c.Next() })
	registerRoutes(srv.engine, srv.routeDeps)
	handler := mgr.LoadAndSave(srv.httpServer.Handler)
	get := httptest.NewRequest(http.MethodGet, "/schedule", nil)
	get.Header.Set("Accept", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, get)
	var cookie *http.Cookie
	for _, c := range res.Result().Cookies() {
		if c.Name == "csrf_token" {
			cookie = c
		}
	}
	if cookie == nil {
		t.Fatal("production handler did not establish CSRF cookie")
	}
	submittedToken := validToken
	for _, token := range []string{"", "invalid", submittedToken} {
		req := httptest.NewRequest(http.MethodPost, "/schedule/stories/1/save-tasks", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set("X-CSRF-Token", token)
		req.AddCookie(cookie)
		result := httptest.NewRecorder()
		handler.ServeHTTP(result, req)
		if token == submittedToken {
			if result.Code != http.StatusUnauthorized {
				t.Fatalf("valid CSRF should reach real login guard: %d %s", result.Code, result.Body.String())
			}
		} else if result.Code != http.StatusForbidden || !strings.Contains(result.Body.String(), "CSRF") {
			t.Fatalf("invalid CSRF should be rejected before route: %d %s", result.Code, result.Body.String())
		}
	}
}
