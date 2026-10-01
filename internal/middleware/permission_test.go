// =============================================================================
// 文件: internal/middleware/permission_test.go
// 模块: 中间件
// 类型: test
// 职责: 验证诊断入口的登录与超级管理员限制，不连接真实数据库。
// 依赖: 无
// =============================================================================

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
)

func TestDebugLoginAndSuperAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, account, accept string
		userID                int64
		want                  int
	}{
		{"匿名页面", "", "text/html", 0, http.StatusSeeOther},
		{"匿名接口", "", "application/json", 0, http.StatusUnauthorized},
		{"普通用户", "test", "text/html", 1, http.StatusForbidden},
		{"超级管理员", "admin", "text/html", 2, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := debugLoginMockDB(t, tc.userID, tc.account)
			mgr := scs.New()
			req := httptest.NewRequest(http.MethodGet, "/debug", nil)
			ctx, err := mgr.Load(req.Context(), "")
			if err != nil {
				t.Fatal(err)
			}
			mgr.Put(ctx, "userID", tc.userID)
			req = req.WithContext(ctx)
			req.Header.Set("Accept", tc.accept)
			router := gin.New()
			router.GET("/debug", RequireLogin(mgr, db), RequireSuperAdmin(), func(c *gin.Context) { c.Status(http.StatusOK) })
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status=%d, want %d", rec.Code, tc.want)
			}
		})
	}
}
