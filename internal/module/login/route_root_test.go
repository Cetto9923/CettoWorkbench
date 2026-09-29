// =============================================================================
// 文件: internal/module/login/route_root_test.go
// 模块: 登录
// 类型: test
// 职责: 根路由保持 577fcb63 的 302 登录页跳转行为。
// =============================================================================
package login

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRootRedirectMatchesBaseline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, loggedIn := range []bool{false, true} {
		r := gin.New()
		called := false
		redirect := func(c *gin.Context) {
			called = true
			if loggedIn {
				c.Redirect(303, "/home")
				c.Abort()
			}
		}
		(&Handler{}).RegisterRoutes(r, func(c *gin.Context) { c.Next() }, redirect, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 302 || w.Header().Get("Location") != "/login" || called {
			t.Fatalf("loggedIn=%v: code=%d location=%q middlewareCalled=%v", loggedIn, w.Code, w.Header().Get("Location"), called)
		}
	}
}
