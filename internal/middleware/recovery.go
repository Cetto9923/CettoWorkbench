// =============================================================================
// 文件: internal/middleware/recovery.go
// 模块: 中间件
// 类型: middleware
// 职责: 捕获 panic 并返回统一错误响应。
// 依赖: 无
// =============================================================================

package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const recoveryHTML = "<!DOCTYPE html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><title>服务器错误</title></head><body><h1>服务器内部错误</h1></body></html>"

// Recovery 捕获 panic，记录错误与堆栈并返回 500 页面。
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered",
					zap.Any("panic", rec),
					zap.ByteString("stack", debug.Stack()),
					zap.String("path", c.Request.URL.Path),
				)
				c.Header("Content-Type", "text/html; charset=utf-8")
				c.Status(http.StatusInternalServerError)
				c.Abort()
				_, _ = c.Writer.WriteString(recoveryHTML)
			}
		}()
		c.Next()
	}
}
