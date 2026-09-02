// =============================================================================
// 文件: internal/middleware/permission.go
// 模块: 中间件
// 类型: middleware
// 职责: 校验权限并拦截无权限请求。
// 依赖: internal/pkg/perm
// =============================================================================

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"workbench/internal/pkg/perm"
)

const permissionDeniedHTML = "<!DOCTYPE html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><title>无权限</title></head><body><h1>无权限访问</h1></body></html>"

// RequirePerm 检查当前用户是否具备权限。
// 策略：超级管理员短路通过；自助动作 perm（systemPerms 中的项）对所有已登录用户默认放行；
// 其余权限基于 userPerms 校验。
func RequirePerm(p perm.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		u := CurrentUser(c)
		if u == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if u.IsSuperAdmin {
			c.Next()
			return
		}
		if perm.IsSystemPerm(p) {
			c.Next()
			return
		}
		if hasPermission(c, p) {
			c.Next()
			return
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Status(http.StatusForbidden)
		c.Abort()
		_, _ = c.Writer.WriteString(permissionDeniedHTML)
	}
}

func hasPermission(c *gin.Context, p perm.Permission) bool {
	permsVal, ok := c.Get("userPerms")
	if !ok {
		return false
	}
	perms, ok := permsVal.(map[string]bool)
	if !ok {
		return false
	}
	return perms[p.String()]
}
