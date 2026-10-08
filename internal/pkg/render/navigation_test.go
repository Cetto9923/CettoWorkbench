// =============================================================================
// 文件: internal/pkg/render/navigation_test.go
// 模块: 基础设施
// 职责: 验证公共外壳、导航权限快照与顶部布局的渲染契约。
// =============================================================================
package render

import (
	"html/template"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"workbench/internal/pkg/menu"
)

func TestNavigationUsesAuthenticatedPermissions(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	renderer := &Renderer{templateDir: filepath.Join(filepath.Dir(source), "../../../web/templates"), cache: make(map[string]*template.Template)}
	cases := []struct {
		name, path, nav string
		granted         map[string]bool
		hidden          bool
	}{
		{"missing", "/admin/users", "sidebar", nil, false},
		{"department", "/admin/dept-overrides", "sidebar", map[string]bool{"dept:list": true}, false},
		{"top", "/admin/roles", "top", map[string]bool{"role:list": true}, false},
		{"standalone", "/profile", "sidebar", map[string]bool{"user:list": true}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(out)
			ctx.Request = httptest.NewRequest("GET", tc.path, nil)
			ctx.Set("userPerms", tc.granted)
			menus := adminMenuFixture(tc.granted)
			data := gin.H{
				"LayoutNav": tc.nav, "HideChrome": tc.hidden,
				"UserPerms":    map[string]bool{"role:list": true},
				"Title":        "<script>bad()</script>",
				"CurrentMenus": menus, "SidebarGroups": menus,
				"CurrentPath":    tc.path,
				"ActiveGroupKey": menu.ActiveGroupKey(menus, tc.path),
			}
			if err := renderer.renderPage(ctx, 403, "error", data); err != nil {
				t.Fatal(err)
			}
			html := out.Body.String()
			assertAdminLinks(t, html, tc.granted, tc.hidden)
			if strings.Contains(html, "<script>bad()</script>") {
				t.Error("title not escaped")
			}
			if !tc.hidden && strings.Count(html, `id="themeQuickToggle"`) != 1 {
				t.Error("duplicate or missing theme control")
			}
			if tc.nav == "top" && !strings.Contains(html, `id="topnav-secondary"`) {
				t.Error("secondary navigation missing")
			}
		})
	}
}

// adminMenuFixture 构造「组织管理」分组配置：仅保留未被下线的管理入口，
// 与 zt_menus 种子一致；调用方传入已授权权限以便校验过滤结果。
func adminMenuFixture(granted map[string]bool) []menu.Menu {
	all := []menu.Menu{
		{Key: "roles", Title: "角色管理", Path: "/admin/roles", Perm: "role:list", Type: "menu"},
		{Key: "menus", Title: "菜单管理", Path: "/admin/menus", Perm: "menu:list", Type: "menu"},
		{Key: "operation_logs", Title: "操作日志", Path: "/admin/operation-logs", Perm: "operationlog:list", Type: "menu"},
		{Key: "login_logs", Title: "登录日志", Path: "/admin/login-logs", Perm: "loginlog:list", Type: "menu"},
	}
	kept := all[:0:0]
	for _, item := range all {
		if granted[item.Perm] {
			kept = append(kept, item)
		}
	}
	return []menu.Menu{{Key: "menu_150", Title: "组织管理", Type: "directory", Children: kept}}
}

func assertAdminLinks(t *testing.T, html string, granted map[string]bool, hidden bool) {
	t.Helper()
	for path, code := range map[string]string{"users": "user:list", "roles": "role:list", "depts": "dept:list", "dept-overrides": "dept:list", "menus": "menu:list", "operation-logs": "operationlog:list", "login-logs": "loginlog:list"} {
		got := strings.Contains(html, `href="/admin/`+path+`"`)
		retired := path == "users" || path == "depts" || path == "dept-overrides"
		if got != (granted[code] && !hidden && !retired) {
			t.Errorf("%s link visibility=%v", path, got)
		}
	}
}

func TestProfileHasOnePageTitle(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	renderer := &Renderer{templateDir: filepath.Join(filepath.Dir(source), "../../../web/templates"), cache: make(map[string]*template.Template)}
	for _, nav := range []string{"sidebar", "top"} {
		out := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(out)
		ctx.Request = httptest.NewRequest("GET", "/profile", nil)
		data := gin.H{"LayoutNav": nav, "Title": "个人资料"}
		if err := renderer.renderPage(ctx, 200, "profile/index", data); err != nil {
			t.Fatal(err)
		}
		html := out.Body.String()
		count := strings.Count(html, `>个人资料</h2>`) + strings.Count(html, `class="topbar-page-title">个人资料</span>`)
		if count != 1 {
			t.Errorf("%s has %d primary titles", nav, count)
		}
	}
}
