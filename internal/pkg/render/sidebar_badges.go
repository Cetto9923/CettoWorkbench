// =============================================================================
// 文件: internal/pkg/render/sidebar_badges.go
// 模块: render
// 类型: contract
// 职责: 侧栏角标数据结构与 provider；由 bootstrap 注入实现。
// =============================================================================

package render

import (
	"github.com/gin-gonic/gin"

	"workbench/internal/pkg/menu"
)

// SidebarBadgesProvider 侧栏角标数据源；由 bootstrap 注入 po.Service.SidebarBadges 闭包。
// 返回值会在 enrichData 期间写入 data["SidebarBadges"]，渲染失败时输出零值。
type SidebarBadgesProvider func(c *gin.Context) (SidebarBadges, error)

// SidebarBadges 侧栏角标计数字段。
type SidebarBadges struct {
	Todos       int  `json:"todos"`
	Done        int  `json:"done"`
	Notice      int  `json:"notice"`
	Unavailable bool `json:"unavailable"`
}

// 以下函数注册为模板 helper，供 layout/sidebar.html 调用。
// 角标按菜单 path 绑定，菜单挪到别的一级分组下时红点跟着走。

// navBadge 描述一个二级菜单的角标渲染结果。
type navBadge struct {
	Text    string
	Shown   bool
	Pending bool
}

// navMenuBadge 返回该二级菜单的角标文案与是否渲染角标。
// Go 模板要求多返回值时最后一个是 error，故用结构体承载「是否有角标」。
func navMenuBadge(m menu.Menu, badges SidebarBadges) navBadge {
	text, shown := SidebarBadgeOf(m, badges)
	return navBadge{Text: text, Shown: shown, Pending: badges.Unavailable}
}

// navGroupHasBadge 判断一级分组内是否有角标，用于窄轨红点。
func navGroupHasBadge(group menu.Menu, badges SidebarBadges) bool {
	return GroupHasBadge(group, badges)
}

// pinnableGroupItems 返回该一级分组下可加入「常用」的二级菜单（排除规划中占位）。
// 返回 map 以便模板用 .key / .title / .icon 访问。
func pinnableGroupItems(group menu.Menu) []map[string]string {
	out := make([]map[string]string, 0, len(group.Children))
	for _, item := range group.Children {
		if item.Planned || item.Path == "" {
			continue
		}
		out = append(out, map[string]string{
			"key":   menu.KeyFromPath(item.Path),
			"title": item.Title,
			"icon":  item.Icon,
		})
	}
	return out
}
