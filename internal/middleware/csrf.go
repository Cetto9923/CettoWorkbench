// =============================================================================
// 文件: internal/middleware/csrf.go
// 模块: 中间件
// 类型: middleware
// 职责: 接入 CSRF 防护中间件。
// 依赖: 无
// =============================================================================

package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/justinas/nosurf"
)

const csrfErrorHTML = "<!DOCTYPE html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><title>CSRF 校验失败</title></head><body><h1>CSRF 校验失败</h1></body></html>"

// CSRF 返回可挂载到标准 net/http 的 nosurf 中间件。
func CSRF(secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		csrf := nosurf.New(next)
		csrf.SetIsTLSFunc(func(r *http.Request) bool { return secure || r.TLS != nil })
		csrf.SetBaseCookie(http.Cookie{
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   secure,
			Path:     "/",
		})
		csrf.SetFailureHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.Header.Get("Content-Type"), "application/json") || strings.Contains(r.Header.Get("Accept"), "application/json") {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"success":false,"message":"CSRF 校验失败，请刷新页面后重试"}`))
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(csrfErrorHTML))
		}))
		return csrf
	}
}

// GetToken 获取当前请求的 CSRF Token，供模板渲染使用。
func GetToken(c *gin.Context) string {
	return nosurf.Token(c.Request)
}
