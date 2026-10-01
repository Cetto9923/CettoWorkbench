// =============================================================================
// 文件: internal/module/build/routespermissions_test.go
// 模块: 版本管理
// 类型: test
// 职责: 使用生产路由验证匿名、缺权限、放行及对象级拒绝。
// =============================================================================
package build

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
	"workbench/internal/model"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/zentao"
)

func buildRouter(actor *model.User, granted map[string]bool, svc *Service) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("currentUser", actor)
		c.Set("userPerms", granted)
		c.Request = c.Request.WithContext(zentao.WithAccount(c.Request.Context(), "tester"))
		c.Next()
	})
	NewHandler(svc, nil).RegisterRoutes(r.Group(""))
	return r
}

func TestBuildRoutesPermissionStates(t *testing.T) {
	for _, tc := range []struct {
		name    string
		actor   *model.User
		granted map[string]bool
		want    int
	}{
		{"anonymous", nil, nil, 401}, {"missing", &model.User{ID: 3}, nil, 403},
		{"granted", &model.User{ID: 3}, map[string]bool{perm.BuildLinkStory.String(): true}, 400},
	} {
		for _, action := range []string{"linkstories", "unlinkstories"} {
			out := httptest.NewRecorder()
			buildRouter(tc.actor, tc.granted, nil).ServeHTTP(out, httptest.NewRequest("POST", "/builds/x/"+action, nil))
			if out.Code != tc.want {
				t.Fatalf("%s/%s status=%d want=%d", tc.name, action, out.Code, tc.want)
			}
		}
	}
	// 读列表不要求写权限，已认证的通用权限快照即可进入参数校验。
	r := buildRouter(&model.User{ID: 3}, map[string]bool{perm.KanbanStory.String(): true}, nil)
	for _, action := range []string{"linkstory", "linkedstories"} {
		out := httptest.NewRecorder()
		r.ServeHTTP(out, httptest.NewRequest("GET", "/builds/x/"+action, nil))
		if out.Code != 400 {
			t.Fatalf("read requires write permission: %d", out.Code)
		}
	}
}

func TestBuildRoutesObjectDeniedBeforeOutboundCall(t *testing.T) {
	for _, action := range []string{"linkstories", "unlinkstories"} {
		svc, hits, _ := setupBuildWrite(t, buildWriteCase{origins: []uint{41, 42}, denied: true})
		r := buildRouter(&model.User{ID: 3, Account: "tester"}, map[string]bool{perm.BuildLinkStory.String(): true}, svc)
		req := httptest.NewRequest("POST", "/builds/9/"+action, strings.NewReader(`{"stories":"1,2"}`))
		req.Header.Set("Content-Type", "application/json")
		out := httptest.NewRecorder()
		r.ServeHTTP(out, req)
		if out.Code != 403 || !strings.Contains(out.Body.String(), "US42") || hits.Load() != 0 {
			t.Fatalf("status=%d body=%s hits=%d", out.Code, out.Body.String(), hits.Load())
		}
	}
}
