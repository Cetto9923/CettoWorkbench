// =============================================================================
// 文件: internal/server/routes_test.go
// 模块: 基础设施
// 类型: infra
// 职责: 锁定 debug routes 的 authentication wiring —— Wave 0A (P0-AUTH-01)。
// 依赖: internal/middleware
// =============================================================================

package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"

	"workbench/internal/middleware"
	"workbench/internal/module/debug"
)

// noopHandler 不连任何 module，纯粹用于 wiring 验证。
// 若 middleware 放行，会到达这里并返回 200 + 标记字符串。
func noopHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(http.StatusOK, "REACHED_HANDLER:"+c.FullPath())
	}
}

// newDebugTestRouter 构造一个最小 gin router，模拟 debug group 的真实 wiring。
//
// anon 测试使用真 middleware.RequireLogin(sessionMgr, nil)：
// RequireLogin 在 userID<=0 时直接 abortUnauthenticated（auth.go:32-39），
// 完全不碰 DB。因此传 nil DB 也能完美模拟 anon 路径，符合 unit test 边界。
//
// authenticated 测试无法简单模拟 scs session（Put 需要 LoadAndSave 创建过的 ctx），
// 且 userID>0 后真 RequireLogin 会 load menus/perms（不在 Wave 0A scope）。
// 因此 authed case 用 stub middleware 验证 "wiring 不挡"：
// 通过 X-Test-Authed header 显式表达"测试用登录态"，与 anon 测试分离。
func newDebugTestRouter(t *testing.T, mode string) (*gin.Engine, *scs.SessionManager) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mgr := scs.New()
	mgr.Lifetime = 0

	debugGroup := r.Group("/debug")
	switch mode {
	case "real":
		// 真 RequireLogin: userID<=0 时 abort，nil DB 不影响 anon 路径
		debugGroup.Use(middleware.RequireLogin(mgr, nil))
	case "stub":
		debugGroup.Use(func(c *gin.Context) {
			// 测试 control：显式标记 "已登录"，仅用于 wiring 通过性验证
			if c.GetHeader("X-Test-Authed") == "1" {
				c.Set("currentUser", &struct{}{})
				c.Next()
				return
			}
			// 未标记则按真 RequireLogin 的 anon 行为拒绝
			accept := c.GetHeader("Accept")
			if strings.Contains(accept, "application/json") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error":   "未登录或会话已过期",
				})
				return
			}
			c.Redirect(http.StatusSeeOther, "/login?redirect="+c.Request.URL.RequestURI())
			c.Abort()
		})
	default:
		t.Fatalf("unknown mode: %s", mode)
	}
	no := noopHandler()
	debugGroup.GET("/sqlperf", no)
	debugGroup.GET("/sqlperf/requests", no)
	return r, mgr
}

func doWithSession(mgr *scs.SessionManager, r *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	handler := mgr.LoadAndSave(r)
	handler.ServeHTTP(rr, req)
	return rr
}

// TestDebugRoutes_AnonymousBlocked 验证 anon 访问 /debug/* 被真 RequireLogin 拦下。
// 覆盖 HTML redirect 和 JSON 401 两种 contract。
func TestDebugRoutes_AnonymousBlocked(t *testing.T) {
	r, mgr := newDebugTestRouter(t, "real")

	cases := []struct {
		name string
		path string
	}{
		{"sqlperf", "/debug/sqlperf"},
		{"sqlperf_requests", "/debug/sqlperf/requests"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Case A: HTML Accept → 303 redirect to /login
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Accept", "text/html")
			rr := doWithSession(mgr, r, req)
			if rr.Code != http.StatusSeeOther {
				t.Fatalf("anon HTML accept: status = %d, want %d", rr.Code, http.StatusSeeOther)
			}
			loc := rr.Header().Get("Location")
			if !strings.HasPrefix(loc, "/login?redirect=") {
				t.Fatalf("anon HTML accept: Location = %q, want /login?redirect=... prefix", loc)
			}
			if strings.Contains(rr.Body.String(), "REACHED_HANDLER") {
				t.Fatalf("anon HTML accept: handler was reached (body=%q)", rr.Body.String())
			}

			// Case B: JSON Accept → 401 JSON envelope
			req2 := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req2.Header.Set("Accept", "application/json")
			rr2 := doWithSession(mgr, r, req2)
			if rr2.Code != http.StatusUnauthorized {
				t.Fatalf("anon JSON accept: status = %d, want %d", rr2.Code, http.StatusUnauthorized)
			}
			body := rr2.Body.String()
			if !strings.Contains(body, `"success":false`) {
				t.Fatalf("anon JSON accept: body missing success:false envelope: %q", body)
			}
			if strings.Contains(body, "REACHED_HANDLER") {
				t.Fatalf("anon JSON accept: handler was reached (body=%q)", body)
			}
		})
	}
}

// TestDebugRoutes_AuthenticatedPasses 验证已登录态不会被 wiring 拦下。
// 用 stub middleware + X-Test-Authed header 模拟登录态，验证 handler 可达。
// Wave 0A 不锁 userID>0 后的 menu/perm 加载（auth.go:60-73），那是后续 Wave 的范围。
func TestDebugRoutes_AuthenticatedPasses(t *testing.T) {
	r, mgr := newDebugTestRouter(t, "stub")

	cases := []struct {
		name string
		path string
	}{
		{"sqlperf", "/debug/sqlperf"},
		{"sqlperf_requests", "/debug/sqlperf/requests"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("X-Test-Authed", "1")
			rr := doWithSession(mgr, r, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("authed: status = %d, want 200 (body=%q)", rr.Code, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), "REACHED_HANDLER:"+tc.path) {
				t.Fatalf("authed: handler not reached (body=%q)", rr.Body.String())
			}
		})
	}
}

// newProductionRouter 用真实 registerRoutes 构造 router,锁定 routes.go 里的
// debugGroup.Use(middleware.RequireLogin(...)) 这一行 wiring。
//
// 用真实 debug.Handler + 空 logDir 的 Repo:anon 请求被 RequireLogin 拦下,
// 永远不进入 handler,所以 handler 链路不需要真实 sqllog/DB。
// 如有人删除 routes.go 中的 debugGroup.Use(RequireLogin(...)),本测试将失败
// (handler 会被错误地到达)。
func newProductionRouter(t *testing.T) (*gin.Engine, *scs.SessionManager) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mgr := scs.New()
	mgr.Lifetime = 0

	repo := debug.NewRepo(t.TempDir())
	svc := debug.NewService(repo)
	sqlPerfHandler := debug.NewHandler(svc)

	deps := RouteDeps{
		SessionMgr:     mgr,
		DB:             nil, // anon 路径不进 DB 分支(userID<=0 时 abortUnauthenticated 不读 db)
		SqlPerfHandler: sqlPerfHandler,
	}
	registerRoutes(r, deps)
	return r, mgr
}

// TestDebugRoutes_ProductionWiring_AnonymousBlocked 验证真实 registerRoutes
// 真的把 RequireLogin 挂到 /debug group 上(而不是只验证 RequireLogin 本身能拦)。
// 覆盖 HTML redirect 和 JSON 401 两种 contract。
func TestDebugRoutes_ProductionWiring_AnonymousBlocked(t *testing.T) {
	r, mgr := newProductionRouter(t)

	cases := []struct {
		name string
		path string
	}{
		{"sqlperf", "/debug/sqlperf"},
		{"sqlperf_requests", "/debug/sqlperf/requests"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Case A: JSON Accept → 401 JSON envelope
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Accept", "application/json")
			rr := doWithSession(mgr, r, req)
			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("anon JSON: status = %d, want %d", rr.Code, http.StatusUnauthorized)
			}
			body := rr.Body.String()
			if !strings.Contains(body, `"success":false`) {
				t.Fatalf("anon JSON: missing success:false envelope: %q", body)
			}
			if strings.Contains(body, "REACHED_HANDLER") {
				t.Fatalf("anon JSON: handler was reached (body=%q) — RequireLogin 未挂到 debug group", body)
			}

			// Case B: HTML Accept → 303 redirect to /login
			req2 := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req2.Header.Set("Accept", "text/html")
			rr2 := doWithSession(mgr, r, req2)
			if rr2.Code != http.StatusSeeOther {
				t.Fatalf("anon HTML: status = %d, want %d", rr2.Code, http.StatusSeeOther)
			}
			loc := rr2.Header().Get("Location")
			if !strings.HasPrefix(loc, "/login?redirect=") {
				t.Fatalf("anon HTML: Location = %q, want /login?redirect=... prefix", loc)
			}
		})
	}
}
