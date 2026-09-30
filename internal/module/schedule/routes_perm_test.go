// =============================================================================
// 文件: internal/module/schedule/routes_perm_test.go
// 模块: 排期工作台
// 类型: security regression
// 职责: 锁定排期写路径必须绑定 RequirePerm(perm.PoSchedule)：
//       持有该权限的账号可放行到 handler，未持有者必须 403，未登录必须 401。
//       handler 全部替换为探针，只验证中间件口径，不触碰 DB。
// =============================================================================

package schedule

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"workbench/internal/middleware"
	"workbench/internal/model"
	"workbench/internal/pkg/perm"
)

func probeOK(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"reached": "handler"}) }

// newPermProbeRouter 造一个只挂排期写路径、handler 全部替换为探针的引擎。
// 返回的引擎会按给定 userPerms 注入 currentUser，模拟 RequireLogin 之后的中间件链。
func newPermProbeRouter(actor *model.User, userPerms map[string]bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if actor != nil {
			c.Set("currentUser", actor)
			c.Set("userPerms", userPerms)
		}
		c.Next()
	})
	g := r.Group("/schedule")
	g.POST("/demands/:id/save-scheduling", middleware.RequirePerm(perm.PoSchedule), probeOK)
	g.POST("/stories/:id/save-scheduling", middleware.RequirePerm(perm.PoSchedule), probeOK)
	g.POST("/stories/:id/save-tasks", middleware.RequirePerm(perm.PoSchedule), probeOK)
	g.POST("/windows", middleware.RequirePerm(perm.PoSchedule), probeOK)
	g.PUT("/windows/:id", middleware.RequirePerm(perm.PoSchedule), probeOK)
	g.DELETE("/windows/:id", middleware.RequirePerm(perm.PoSchedule), probeOK)
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
	r := newPermProbeRouter(
		&model.User{ID: 1, Account: "demo_po"},
		map[string]bool{perm.PoSchedule.String(): true},
	)
	for _, p := range scheduleWritePaths {
		t.Run(p.method+" "+p.path, func(t *testing.T) {
			if rec := doProbe(t, r, p.method, p.path); rec.Code != http.StatusOK {
				t.Fatalf("%s %s = %d, want 200 (持 po:schedule 应放行)", p.method, p.path, rec.Code)
			}
		})
	}
}

// 未持有 po:schedule 的账号必须 403，且不得触达 handler。
func TestScheduleWritePaths_MissingPermIsForbidden(t *testing.T) {
	r := newPermProbeRouter(
		&model.User{ID: 2, Account: "no_perm"},
		map[string]bool{perm.ScheduleList.String(): true},
	)
	for _, p := range scheduleWritePaths {
		t.Run(p.method+" "+p.path, func(t *testing.T) {
			if rec := doProbe(t, r, p.method, p.path); rec.Code != http.StatusForbidden {
				t.Fatalf("%s %s = %d, want 403 (无 po:schedule 应被拦)", p.method, p.path, rec.Code)
			}
		})
	}
}

// 未登录时 RequirePerm 必须 401，而不是放行或 403。
func TestScheduleWritePaths_AnonymousIsUnauthorized(t *testing.T) {
	r := newPermProbeRouter(nil, nil)
	for _, p := range scheduleWritePaths {
		t.Run(p.method+" "+p.path, func(t *testing.T) {
			if rec := doProbe(t, r, p.method, p.path); rec.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s = %d, want 401", p.method, p.path, rec.Code)
			}
		})
	}
}
