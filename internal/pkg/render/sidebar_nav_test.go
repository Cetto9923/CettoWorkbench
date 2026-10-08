// =============================================================================
// 文件: internal/pkg/render/sidebar_nav_test.go
// 模块: 基础设施
// 类型: test
// 职责: 锁定侧栏角标的取值口径，避免页面上出现 undefined 或不刷新。
// =============================================================================
package render

import (
	"testing"

	"workbench/internal/pkg/menu"
)

func TestSidebarBadgeMatchesBadgesAPIFields(t *testing.T) {
	badges := SidebarBadges{Todos: 73, Notice: 5}
	cases := []struct {
		path      string
		wantText  string
		wantShown bool
	}{
		{"/todos", "73", true},
		{"/notice", "5", true},
		{"/home", "", false},
	}
	for _, tc := range cases {
		got := navMenuBadge(menu.Menu{Path: tc.path}, badges)
		if got.Text != tc.wantText || got.Shown != tc.wantShown {
			t.Errorf("%s 角标 = {%q %v}，期望 {%q %v}", tc.path, got.Text, got.Shown, tc.wantText, tc.wantShown)
		}
	}
}

// badges.js 以 data-sidebar-badge 的值去响应里取数，值必须存在，否则显示 undefined。
func TestSidebarBadgeKindIsResolvableInBadgesAPI(t *testing.T) {
	apiFields := map[string]bool{"todos": true, "done": true, "notice": true}
	for _, path := range []string{"/todos", "/notice"} {
		kind := menu.Menu{Path: path}.BadgeKind()
		if !apiFields[kind] {
			t.Errorf("%s 的角标类型 %q 在 /navigation/badges 响应中不存在", path, kind)
		}
	}
}

// 角标服务不可用时须渲染「…」并置 data-badge-pending，badges.js 才会自动补取。
func TestSidebarBadgePendingMarksAutoFetch(t *testing.T) {
	got := navMenuBadge(menu.Menu{Path: "/todos"}, SidebarBadges{Unavailable: true})
	if !got.Pending || got.Text != "…" {
		t.Errorf("不可用时应为 待刷新+「…」，实际 pending=%v text=%q", got.Pending, got.Text)
	}
	ok := navMenuBadge(menu.Menu{Path: "/todos"}, SidebarBadges{Todos: 3})
	if ok.Pending {
		t.Errorf("正常返回时不应标记待刷新")
	}
}

func TestSidebarBadgeNoticeCapAt99(t *testing.T) {
	if got := navMenuBadge(menu.Menu{Path: "/notice"}, SidebarBadges{Notice: 120}).Text; got != "99+" {
		t.Errorf("通知超过 99 应显示 99+，实际 %q", got)
	}
}

func TestGroupHasBadgeFollowsChildren(t *testing.T) {
	group := menu.Menu{Key: "menu_100", Children: []menu.Menu{
		{Path: "/home"},
		{Path: "/todos"},
	}}
	if !GroupHasBadge(group, SidebarBadges{Todos: 1}) {
		t.Error("含 /todos 的分组应有红点")
	}
	empty := menu.Menu{Key: "menu_110", Children: []menu.Menu{{Path: "/schedule"}}}
	if GroupHasBadge(empty, SidebarBadges{Todos: 1, Notice: 2}) {
		t.Error("无角标菜单的分组不应有红点")
	}
}
