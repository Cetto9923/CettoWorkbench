// =============================================================================
// 文件: internal/pkg/render/sidebar_badges.go
// 模块: render
// 类型: contract
// 职责: 侧栏角标数据结构与 provider 接口。独立成文件以保持 render.go 在职责分离下
//       不超过 500 行上限；bootstrap 注入 po.Service.SidebarBadges 闭包，render 包
//       无反向依赖 po 包。
// =============================================================================

package render

import "github.com/gin-gonic/gin"

// SidebarBadgesProvider 侧栏角标数据源；由 bootstrap 注入 po.Service.SidebarBadges 闭包。
// 返回值会在 enrichData 期间写入 data["SidebarBadges"]，渲染失败时输出零值。
type SidebarBadgesProvider func(c *gin.Context) (SidebarBadges, error)

// SidebarBadges 与 sidebar_badges.go 字段对齐；这里独立声明避免 render 包反向 import po。
type SidebarBadges struct {
	Todos  int `json:"todos"`
	Done   int `json:"done"`
	Notice int `json:"notice"`
}
