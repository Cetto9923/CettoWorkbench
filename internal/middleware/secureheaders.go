// =============================================================================
// 文件: internal/middleware/secureheaders.go
// 模块: 中间件
// 类型: middleware
// 职责: 注入安全响应头。
// 依赖: 无
// =============================================================================

package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const cspValue = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:;"

// SecureHeaders 设置基础安全响应头。
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "same-origin")
		c.Header("Content-Security-Policy", cspValue)
		if c.Request.URL.Path != "/login" && !strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Header("Cache-Control", "no-store")
		}
		c.Next()
	}
}
