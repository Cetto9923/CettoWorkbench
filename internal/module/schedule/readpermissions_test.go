// =============================================================================
// 文件: internal/module/schedule/readpermissions_test.go
// 模块: 排期工作台
// 类型: test
// 职责: 锁定生产读路由权限绑定及安装种子的排期授权。
// =============================================================================
package schedule

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"workbench/internal/model"
	"workbench/internal/pkg/perm"
)

func TestScheduleReadPermissionBinding(t *testing.T) {
	paths := []string{"/schedule/filter-counts", "/schedule/matching-plans", "/schedule/demands/x/scheduling", "/schedule/demands/x/review-to-story-notice", "/schedule/stories/x/scheduling", "/schedule/products/x/projects", "/schedule/projects/x/executions", "/schedule/stories/x/tasks", "/schedule/windows/x", "/schedule/window-options", "/schedule/windows", "/schedule"}
	for _, actor := range []*model.User{nil, {ID: 3, Account: "tester"}} {
		r := newPermProbeRouter(nil, actor, nil)
		want := 403
		if actor == nil {
			want = 401
		}
		for _, path := range paths {
			if out := doProbe(t, r, "GET", path); out.Code != want {
				t.Fatalf("%s status=%d want=%d", path, out.Code, want)
			}
		}
	}
	r := newPermProbeRouter(nil, &model.User{ID: 3, Account: "tester"}, map[string]bool{perm.ScheduleList.String(): true})
	if out := doProbe(t, r, "GET", "/schedule/stories/x/tasks"); out.Code != 400 {
		t.Fatalf("allowed request should reach ID validation: %d", out.Code)
	}
}

func TestInstalledRolesKeepScheduleReadAccess(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	sql, err := os.ReadFile(filepath.Join(filepath.Dir(file), "../../../db/install.sql"))
	if err != nil {
		t.Fatal(err)
	}
	roles := regexp.MustCompile(`\((\d+), '(super_admin|po)',`).FindAllStringSubmatch(string(sql), -1)
	if len(roles) != 2 {
		t.Fatalf("unexpected seeded roles: %v", roles)
	}
	for _, role := range roles {
		if !strings.Contains(string(sql), "("+role[1]+", 'schedule:list',") {
			t.Fatalf("seeded role %s loses schedule access", role[2])
		}
	}
}
