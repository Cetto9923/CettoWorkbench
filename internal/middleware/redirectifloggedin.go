// =============================================================================
// 文件: internal/middleware/redirectifloggedin.go
// 模块: 中间件
// 类型: middleware
// 职责: 已登录用户访问登录页时自动重定向，避免重复登录。
// 依赖: internal/pkg/session
// =============================================================================

package middleware

import (
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"

	"workbench/internal/pkg/session"
)

// RedirectIfLoggedIn 若当前用户已登录（Session 中存在有效 userID），
// 仅用于登录页等“已登录不应访问”的公开路由。
func RedirectIfLoggedIn(sessionMgr *scs.SessionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if session.GetUserID(c.Request.Context(), sessionMgr) > 0 {
			c.Redirect(http.StatusSeeOther, "/home")
			c.Abort()
			return
		}
		c.Next()
	}
}
