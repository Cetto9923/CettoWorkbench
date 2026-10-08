// =============================================================================
// 文件: internal/pkg/render/sidebar_nav.go
// 模块: 基础设施
// 类型: infra
// 职责: 为侧栏模板准备一级分组高亮、二级高亮与角标数据。
// =============================================================================
package render

import (
	"strings"

	"workbench/internal/pkg/menu"

	"github.com/gin-gonic/gin"
)

// enrichSidebarNav 写入侧栏导航数据：
//   - SidebarGroups：过滤后的菜单树（复用 data["CurrentMenus"]）
//   - ActiveGroupKey：当前地址所属的一级分组，命中失败时回退用户记忆的分组
//   - ActiveItemKey：当前地址命中的二级菜单，用于二级高亮
func enrichSidebarNav(c *gin.Context, data gin.H, menus []menu.Menu, remembered string) {
	group := menu.ActiveGroupKey(menus, currentRequestPath(c))
	if group == "" {
		group = strings.TrimSpace(remembered)
	}
	data["SidebarGroups"] = menus
	data["ActiveGroupKey"] = group
	data["ActiveItemKey"] = menu.ActiveItemKey(menus, currentRequestPath(c))
}

// currentRequestPath 返回当前请求路径；无请求时返回空串。
func currentRequestPath(c *gin.Context) string {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return ""
	}
	return c.Request.URL.Path
}

// SidebarBadgeOf 返回菜单对应角标的展示值与是否有角标。
// 99 以上显示 99+；角标服务不可用时（Unavailable）返回 "…"。
func SidebarBadgeOf(m menu.Menu, badges SidebarBadges) (string, bool) {
	var (
		value   int
		pending bool
	)
	switch m.BadgeKind() {
	case "todos":
		value, pending = badges.Todos, badges.Unavailable
	case "notice":
		value, pending = badges.Notice, badges.Unavailable
	default:
		return "", false
	}
	if pending {
		return "…", true
	}
	// 未读通知超 99 显示 99+；待办数按实际值显示，与改前模板一致。
	if m.BadgeKind() == "notice" && value > 99 {
		return "99+", true
	}
	return itoa(value), true
}

// GroupHasBadge 判断一级分组内是否有需要提示的角标（待办或未读通知）。
func GroupHasBadge(group menu.Menu, badges SidebarBadges) bool {
	for _, item := range group.Children {
		if _, ok := SidebarBadgeOf(item, badges); ok {
			return true
		}
	}
	return false
}

// RememberedRailGroup 读取用户上次选择的一级分组：优先 ?nav= 查询参数，其次 po_active_rail cookie。
func RememberedRailGroup(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if c.Request.URL != nil {
		if v := strings.TrimSpace(c.Request.URL.Query().Get("nav")); v != "" {
			return v
		}
	}
	if cookie, err := c.Request.Cookie("po_active_rail"); err == nil {
		return strings.TrimSpace(cookie.Value)
	}
	return ""
}

// itoa 把计数转成字符串，避免为单一用途引入 strconv。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
