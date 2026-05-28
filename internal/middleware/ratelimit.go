// =============================================================================
// 文件: internal/middleware/ratelimit.go
// 模块: 中间件
// 类型: middleware
// 职责: 按 IP 执行请求限流。
// 依赖: internal/pkg/ratelimit
// =============================================================================

package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	ratelimitpkg "workbench/internal/pkg/ratelimit"
)

const rateLimitHTML = "<!DOCTYPE html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><title>请求过于频繁</title></head><body><h1>请求过于频繁，请稍后再试</h1></body></html>"

// RateLimit 按 key 执行限流，命中后返回 429。
func RateLimit(limiter *ratelimitpkg.Limiter, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limiter == nil || keyFunc == nil {
			c.Next()
			return
		}
		key := keyFunc(c)
		if limiter.Allow(key) {
			c.Next()
			return
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Status(http.StatusTooManyRequests)
		c.Abort()
		_, _ = c.Writer.WriteString(rateLimitHTML)
	}
}

// KeyByIP 按客户端 IP 进行限流。
func KeyByIP(c *gin.Context) string {
	return strings.TrimSpace(c.ClientIP())
}

// KeyByIPAndPath 按 IP + 路径进行限流。
func KeyByIPAndPath(c *gin.Context) string {
	return strings.TrimSpace(c.ClientIP()) + "|" + c.FullPath()
}

// KeyByAccountAndIP 按账号 + IP 进行限流。
func KeyByAccountAndIP(c *gin.Context) string {
	account := c.PostForm("account")
	if account == "" {
		account = c.Query("account")
	}
	return ratelimitpkg.KeyByIPAndAccount(c.ClientIP(), account)
}
