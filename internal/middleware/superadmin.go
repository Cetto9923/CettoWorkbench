// =============================================================================
// 文件: internal/middleware/superadmin.go
// 模块: 中间件
// 类型: middleware
// 职责: 校验当前已登录用户是否为超级管理员。
// 依赖: internal/model
// =============================================================================

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireSuperAdmin 要求请求已登录且当前用户为超级管理员。
// 必须挂在 RequireLogin 之后使用。
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := CurrentUser(c)
		if u == nil {
			abortUnauthenticated(c)
			return
		}
		if !u.IsSuperAdmin {
			if expectsJSON(c) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"success": false,
					"error":   "无权限访问",
				})
				return
			}
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.Status(http.StatusForbidden)
			c.Abort()
			_, _ = c.Writer.WriteString(permissionDeniedHTML)
			return
		}
		c.Next()
	}
}
