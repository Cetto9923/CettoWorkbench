// =============================================================================
// 文件: internal/pkg/render/sidebar_badges.go
// 模块: render
// 类型: contract
// 职责: 侧栏角标数据结构与 provider；由 bootstrap 注入实现。
// =============================================================================

package render

import "github.com/gin-gonic/gin"

// SidebarBadgesProvider 侧栏角标数据源；由 bootstrap 注入 po.Service.SidebarBadges 闭包。
// 返回值会在 enrichData 期间写入 data["SidebarBadges"]，渲染失败时输出零值。
type SidebarBadgesProvider func(c *gin.Context) (SidebarBadges, error)

// SidebarBadges 侧栏角标计数字段。
type SidebarBadges struct {
	Todos  int `json:"todos"`
	Done   int `json:"done"`
	Notice int `json:"notice"`
}
