// =============================================================================
// 文件: internal/middleware/activenav.go
// 模块: 中间件
// 类型: middleware
// 职责: 将当前请求所属侧栏菜单（与 zt_menus.path 一致）写入 Gin 上下文，供模板 ActiveNavKey 与 Menu.Key 对齐高亮。
// 依赖: internal/pkg/menu
// =============================================================================

package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"workbench/internal/pkg/menu"
)

// ActiveNav 声明本路由组对应的菜单入口 path（须与数据库 zt_menus.path 一致，例如 "/resources/categories"）。
// 渲染页通过 menu.KeyFromPath 转为与 Menu.Key 相同的键，再与侧栏节点匹配。
func ActiveNav(menuPath string) gin.HandlerFunc {
	key := menu.KeyFromPath(strings.TrimSpace(menuPath))
	return func(c *gin.Context) {
		c.Set(menu.ContextActiveNavKey, key)
		c.Next()
	}
}
