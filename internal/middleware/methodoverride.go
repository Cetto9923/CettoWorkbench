// =============================================================================
// 文件: internal/middleware/methodoverride.go
// 模块: 中间件
// 类型: middleware
// 职责: 将表单 _method 转换为真实 HTTP 方法。
// 依赖: 无
// =============================================================================

package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var allowedOverrideMethods = map[string]struct{}{
	"PUT":    {},
	"PATCH":  {},
	"DELETE": {},
}

// MethodOverride 支持通过 _method 表单字段将 POST 覆盖为 PUT/PATCH/DELETE。
func MethodOverride() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}

		_ = c.Request.ParseForm()
		override := strings.ToUpper(strings.TrimSpace(c.Request.Form.Get("_method")))
		if _, ok := allowedOverrideMethods[override]; ok {
			c.Request.Method = override
		}
		c.Next()
	}
}
