// =============================================================================
// 文件: internal/module/schedule/routes_perm_test.go
// 模块: 排期工作台
// 类型: security regression
// 职责: 锁定排期写路径必须绑定 RequirePerm(perm.PoSchedule)：
//       持有该权限的账号可放行到 handler，未持有者必须 403，未登录必须 401。
//       通过生产 RegisterRoutes 注册，业务依赖使用 nil 或 sqlmock，不触碰 DB。
// =============================================================================

package schedule

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"workbench/internal/model"
	"workbench/internal/pkg/perm"
)

// newPermProbeRouter 使用生产路由注册，防止测试与生产权限绑定发生漂移。
// 返回的引擎会按给定 userPerms 注入 currentUser，模拟 RequireLogin 之后的中间件链。
func newPermProbeRouter(svc *Service, actor *model.User, userPerms map[string]bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if actor != nil {
			c.Set("currentUser", actor)
			c.Set("userPerms", userPerms)
		}
		c.Next()
	})
	NewHandler(nil, nil, svc, "").RegisterRoutes(r.Group(""))
	return r
}

// scheduleWritePaths 覆盖全部 6 条排期写路径。
var scheduleWritePaths = []struct {
	method string
	path   string
}{
	{"POST", "/schedule/demands/1/save-scheduling"},
	{"POST", "/schedule/stories/1/save-scheduling"},
	{"POST", "/schedule/stories/1/save-tasks"},
	{"POST", "/schedule/windows"},
	{"PUT", "/schedule/windows/1"},
	{"DELETE", "/schedule/windows/1"},
}

func doProbe(t *testing.T, r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Accept", "application/json")
	r.ServeHTTP(rec, req)
	return rec
}

// 持有 po:schedule 的账号必须放行到 handler（不得被误拦）。
func TestScheduleWritePaths_PermittedUserReachesHandler(t *testing.T) {
	for _, p := range scheduleWritePaths {
		t.Run(p.method+" "+p.path, func(t *testing.T) {
			db, _ := newAuthzMockDB(t)
			r := newPermProbeRouter(NewService(NewRepo(db), nil, nil, nil, nil),
				&model.User{ID: 1, Account: "demo_po"},
				map[string]bool{perm.PoSchedule.String(): true})
			rec := doProbe(t, r, p.method, p.path)
			if rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
				t.Fatalf("%s %s = %d, want neither 401 nor 403", p.method, p.path, rec.Code)
			}
		})
	}
}

// 未持有 po:schedule 的账号必须 403，且不得触达 handler。
func TestScheduleWritePaths_MissingPermIsForbidden(t *testing.T) {
	r := newPermProbeRouter(
		nil, &model.User{ID: 2, Account: "no_perm"},
		map[string]bool{perm.ScheduleList.String(): true},
	)
	for _, p := range scheduleWritePaths {
		t.Run(p.method+" "+p.path, func(t *testing.T) {
			rec := doProbe(t, r, p.method, p.path)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("%s %s = %d, want 403 (无 po:schedule 应被拦)", p.method, p.path, rec.Code)
			}
			// nil 依赖配合中间件独有响应，避免业务层 403 掩盖路由漏绑。
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body["success"] != false || body["error"] != "无权限访问" || body["message"] != nil {
				t.Fatalf("未被权限中间件拦截：%s", rec.Body.String())
			}
		})
	}
}

// 未登录时 RequirePerm 必须 401，而不是放行或 403。
func TestScheduleWritePaths_AnonymousIsUnauthorized(t *testing.T) {
	r := newPermProbeRouter(nil, nil, nil)
	for _, p := range scheduleWritePaths {
		t.Run(p.method+" "+p.path, func(t *testing.T) {
			if rec := doProbe(t, r, p.method, p.path); rec.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s = %d, want 401", p.method, p.path, rec.Code)
			}
		})
	}
}
