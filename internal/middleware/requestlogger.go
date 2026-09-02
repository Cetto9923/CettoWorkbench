// =============================================================================
// 文件: internal/middleware/requestlogger.go
// 模块: 中间件
// 类型: middleware
// 职责: 记录请求日志。
// 依赖: internal/pkg/logger
// =============================================================================

package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	loggerpkg "workbench/internal/pkg/logger"
)

// RequestLogger 记录请求基础信息，不记录请求体。
func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}
	return func(c *gin.Context) {
		start := time.Now()
		c.Request = c.Request.WithContext(loggerpkg.ToContext(c.Request.Context(), logger))
		c.Next()

		if shouldIgnoreRequestLog(c) {
			return
		}

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("clientIP", c.ClientIP()),
			zap.String("userAgent", c.Request.UserAgent()),
		}

		status := c.Writer.Status()
		switch {
		case status >= http.StatusInternalServerError:
			logger.Error("http request", fields...)
		case status >= http.StatusBadRequest:
			// 常见可预期客户端错误（如登录校验失败）降级为 Info，减少误报噪音。
			if isExpectedClientError(c, status) {
				logger.Info("http request", fields...)
				return
			}
			logger.Warn("http request", fields...)
		default:
			logger.Info("http request", fields...)
		}
	}
}

func shouldIgnoreRequestLog(c *gin.Context) bool {
	return c.Request.Method == http.MethodGet &&
		c.Request.URL.Path == "/.well-known/appspecific/com.chrome.devtools.json" &&
		c.Writer.Status() == http.StatusNotFound
}

func isExpectedClientError(c *gin.Context, status int) bool {
	return c.Request.Method == http.MethodPost &&
		c.Request.URL.Path == "/login" &&
		status == http.StatusUnprocessableEntity
}
