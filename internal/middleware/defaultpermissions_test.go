// =============================================================================
// 文件: internal/middleware/defaultpermissions_test.go
// 模块: 认证权限
// 类型: test
// 职责: 证明真实认证仍默认授予关联及看板权限，未用权限不再默认放行。
// =============================================================================
package middleware

import (
	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"testing"
	"workbench/internal/pkg/perm"
)

func TestLoginDefaultPermissions(t *testing.T) {
	for _, tc := range []struct {
		p    perm.Permission
		want int
	}{
		{perm.BuildLinkStory, 200}, {perm.KanbanStory, 200}, {perm.AuthLogout, 200},
		{perm.PoDemandReview, 403}, {perm.PoDemandAcceptance, 403},
	} {
		t.Run(tc.p.String(), func(t *testing.T) {
			mgr := scs.New()
			req := httptest.NewRequest("GET", "/probe", nil)
			ctx, err := mgr.Load(req.Context(), "")
			if err != nil {
				t.Fatal(err)
			}
			mgr.Put(ctx, "userID", int64(3))
			req = req.WithContext(ctx)
			req.Header.Set("Accept", "application/json")
			r := gin.New()
			r.GET("/probe", RequireLogin(mgr, debugLoginMockDB(t, 3, "tester")), RequirePerm(tc.p), func(c *gin.Context) { c.Status(http.StatusOK) })
			out := httptest.NewRecorder()
			r.ServeHTTP(out, req)
			if out.Code != tc.want {
				t.Fatalf("status=%d want=%d", out.Code, tc.want)
			}
		})
	}
}
