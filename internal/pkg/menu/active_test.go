// =============================================================================
// 文件: internal/pkg/menu/active_test.go
// 模块: 基础设施
// 类型: test
// 职责: 锁定当前地址到一级分组/二级菜单的映射，覆盖侧栏全部入口与历史地址。
// =============================================================================
package menu

import "testing"

// testMenus 与 db/upgrade_menu_config.sql 的侧栏种子结构一致。
func testMenus() []Menu {
	return []Menu{
		{Key: "menu_100", Title: "我的工作台", Children: []Menu{
			{Key: "home", Path: "/home"},
			{Key: "todos", Path: "/todos"},
			{Key: "done", Path: "/done"},
			{Key: "notice", Path: "/notice"},
			{Key: "follow", Path: "/follow"},
		}},
		{Key: "menu_110", Title: "需求规划", Children: []Menu{
			{Key: "schedule", Path: "/schedule"},
			{Key: "query", Path: "/query", ActivePaths: []string{"/demands/"}},
			{Key: "version_follow", Path: "/version-follow"},
		}},
		{Key: "menu_120", Title: "团队协作", Children: []Menu{
			{Key: "board_demand", Path: "/board/demand", ActivePaths: []string{"/board/task"}},
			{Key: "agileteam", Path: "/agileteam", ActivePaths: []string{"/pmo"}},
			{Key: "iterations", Planned: true},
		}},
		{Key: "menu_140", Title: "治理分析", Children: []Menu{
			{Key: "issues_risk", Path: "/issues/risk", ActivePaths: []string{"/issue-risk"}},
			{Key: "metrics_radar", Path: "/metrics/radar"},
			{Key: "metrics_manage", Path: "/metrics/manage"},
			{Key: "quality_alert", Planned: true},
		}},
		{Key: "menu_150", Title: "组织管理", Children: []Menu{
			{Key: "roles", Path: "/admin/roles"},
			{Key: "menus", Path: "/admin/menus"},
		}},
	}
}

func TestActiveGroupKeyMatchesSidebarEntries(t *testing.T) {
	cases := []struct {
		path      string
		wantGroup string
		wantItem  string
	}{
		{"/home", "menu_100", "home"},
		{"/todos", "menu_100", "todos"},
		{"/done", "menu_100", "done"},
		{"/notice", "menu_100", "notice"},
		{"/follow", "menu_100", "follow"},
		{"/schedule", "menu_110", "schedule"},
		{"/query", "menu_110", "query"},
		{"/demands/58857", "menu_110", "query"},
		{"/demands/58857/primary-action", "menu_110", "query"},
		{"/version-follow", "menu_110", "version_follow"},
		{"/version-follow/items", "menu_110", "version_follow"},
		{"/board/demand", "menu_120", "board_demand"},
		{"/board/task", "menu_120", "board_demand"},
		{"/agileteam", "menu_120", "agileteam"},
		{"/pmo", "menu_120", "agileteam"},
		{"/issues/risk", "menu_140", "issues_risk"},
		{"/issue-risk", "menu_140", "issues_risk"},
		{"/metrics/radar", "menu_140", "metrics_radar"},
		{"/metrics/manage", "menu_140", "metrics_manage"},
		{"/admin/roles", "menu_150", "roles"},
		{"/admin/menus", "menu_150", "menus"},
		{"/admin/menus/1/edit", "menu_150", "menus"},
	}
	menus := testMenus()
	for _, tc := range cases {
		if got := ActiveGroupKey(menus, tc.path); got != tc.wantGroup {
			t.Errorf("ActiveGroupKey(%q) = %q，期望 %q", tc.path, got, tc.wantGroup)
		}
		if got := ActiveItemKey(menus, tc.path); got != tc.wantItem {
			t.Errorf("ActiveItemKey(%q) = %q，期望 %q", tc.path, got, tc.wantItem)
		}
	}
}

func TestActiveGroupKeyIgnoresUnrelatedAndNearMissPaths(t *testing.T) {
	menus := testMenus()
	// 不属于任何菜单的地址：管理面板靠记忆分组，不高亮任何一级。
	for _, path := range []string{"", "/login", "/profile", "/kanban/story", "/queryx", "/boardx"} {
		if got := ActiveGroupKey(menus, path); got != "" {
			t.Errorf("ActiveGroupKey(%q) = %q，期望空串", path, got)
		}
		if got := ActiveItemKey(menus, path); got != "" {
			t.Errorf("ActiveItemKey(%q) = %q，期望空串", path, got)
		}
	}
}

func TestPrefixLen(t *testing.T) {
	cases := []struct {
		path, prefix string
		want         int
	}{
		{"/board/demand", "/board/demand", len("/board/demand")},
		{"/board/demand/items", "/board/demand", len("/board/demand")},
		{"/demands/123", "/demands/", len("/demands/")},
		{"/demands", "/demands/", 0},
		{"/boardx", "/board/demand", 0},
		{"/board", "/board/demand", 0},
		{"", "/board", 0},
	}
	for _, tc := range cases {
		if got := prefixLen(tc.path, tc.prefix); got != tc.want {
			t.Errorf("prefixLen(%q, %q) = %d，期望 %d", tc.path, tc.prefix, got, tc.want)
		}
	}
}

func TestFilterKeepsPlannedRowsAndDropsUnpermitted(t *testing.T) {
	menus := []Menu{
		{Key: "g1", Title: "分组一", Children: []Menu{
			{Key: "a", Path: "/a"},
			{Key: "p", Planned: true},
		}},
		{Key: "g2", Title: "分组二", Perm: "x:list", Children: []Menu{
			{Key: "b", Path: "/b"},
		}},
		{Key: "g3", Title: "空分组", Children: nil},
	}
	got := Filter(menus, map[string]bool{"y:list": true}, false)
	if len(got) != 1 || got[0].Key != "g1" {
		t.Fatalf("应只保留分组一，实际 %v", got)
	}
	// 规划中占位没有 path 也不能被丢掉。
	if len(got[0].Children) != 2 {
		t.Fatalf("规划中占位应保留，实际 %d 个子项", len(got[0].Children))
	}
	if got := Filter(menus, map[string]bool{"x:list": true}, false); len(got) != 2 {
		t.Fatalf("有 x:list 时应保留两组，实际 %d", len(got))
	}
	if got := Filter(menus, nil, true); len(got) != 3 {
		t.Fatalf("超管应看到全部三组，实际 %d", len(got))
	}
}

func TestRailLabelAndBadgeKind(t *testing.T) {
	if got := (Menu{Title: "需求规划", ShortTitle: "规划"}).RailLabel(); got != "规划" {
		t.Errorf("RailLabel = %q", got)
	}
	if got := (Menu{Title: "需求规划"}).RailLabel(); got != "需求规划" {
		t.Errorf("短名为空应回退 title，实际 %q", got)
	}
	// 角标类型必须与 /navigation/badges 的 JSON 字段一致（todos / notice），
	// 否则 badges.js 取不到数字，页面上会显示 undefined。
	if got := (Menu{Path: "/todos"}).BadgeKind(); got != "todos" {
		t.Errorf("BadgeKind(/todos) = %q，期望 todos", got)
	}
	if got := (Menu{Path: "/notice"}).BadgeKind(); got != "notice" {
		t.Errorf("BadgeKind(/notice) = %q，期望 notice", got)
	}
	if got := (Menu{Path: "/home"}).BadgeKind(); got != "" {
		t.Errorf("BadgeKind(/home) = %q，期望空串", got)
	}
}
