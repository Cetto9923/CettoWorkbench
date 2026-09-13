// Package middleware 接入 CSRF 防护中间件。
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/justinas/nosurf"
)

const (
	csrfErrorHTML = "<!DOCTYPE html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><title>CSRF 校验失败</title></head><body><h1>CSRF 校验失败</h1></body></html>"
	csrfErrorJSON = `{"success":false,"code":403,"error":"CSRF 校验失败","message":"CSRF 校验失败"}`
)

// CSRF 返回可挂载到标准 net/http 的 nosurf 中间件。
// 生产与本地默认启用，无环境变量关闭总开关。
// cookieSecure 来自生效配置，不通过任意请求头动态改变。
func CSRF(cookieSecure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		csrf := nosurf.New(next)
		csrf.SetIsTLSFunc(func(r *http.Request) bool {
			return cookieSecure
		})
		csrf.SetBaseCookie(http.Cookie{
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   cookieSecure,
			Path:     "/",
		})
		csrf.SetFailureHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			accept := r.Header.Get("Accept")
			xrw := r.Header.Get("X-Requested-With")
			contentType := r.Header.Get("Content-Type")
			if strings.Contains(accept, "application/json") ||
				xrw == "XMLHttpRequest" ||
				strings.Contains(contentType, "application/json") {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(csrfErrorJSON))
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
	if c == nil || c.Request == nil {
		return ""
	}
	return nosurf.Token(c.Request)
}
