// =============================================================================
// 文件: internal/module/profile/routespermissions_test.go
// 模块: 个人资料
// 类型: test
// 职责: 验证个人写接口和偏好接口的生产权限绑定三态。
// =============================================================================
package profile

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
	"workbench/internal/model"
	"workbench/internal/pkg/perm"
)

func TestProfileRoutesPermissionStates(t *testing.T) {
	paths := []struct {
		method, path string
		p            perm.Permission
	}{
		{"PUT", "/profile", perm.PoHomeList}, {"PUT", "/profile/password", perm.PoHomeList},
		{"PUT", "/api/profile", perm.PoHomeList}, {"PUT", "/api/profile/password", perm.PoHomeList},
		{"GET", "/api/profile/get", perm.KanbanStory}, {"POST", "/api/profile/set", perm.KanbanStory},
	}
	for _, tc := range []struct {
		actor   *model.User
		granted bool
		want    int
	}{
		{nil, false, 401}, {&model.User{ID: 3, Account: "tester"}, false, 403}, {&model.User{ID: 3, Account: "tester"}, true, 400},
	} {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("currentUser", tc.actor)
			c.Set("userPerms", map[string]bool{perm.PoHomeList.String(): tc.granted, perm.KanbanStory.String(): tc.granted})
			c.Next()
		})
		NewHandler(nil, nil).RegisterRoutes(r.Group(""))
		for _, path := range paths {
			out := httptest.NewRecorder()
			req := httptest.NewRequest(path.method, path.path, nil)
			req.Header.Set("Accept", "application/json")
			r.ServeHTTP(out, req)
			if out.Code != tc.want {
				t.Fatalf("%s %s status=%d want=%d", path.method, path.path, out.Code, tc.want)
			}
		}
	}
}
