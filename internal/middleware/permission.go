// Package middleware 校验权限并拦截无权限请求。
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"workbench/internal/pkg/perm"
)

const permissionDeniedHTML = "<!DOCTYPE html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><title>无权限</title></head><body><h1>无权限访问</h1></body></html>"

// RequirePerm 检查当前用户是否具备权限。
// 策略：超级管理员短路通过；非超级管理员基于 userPerms 校验；
// 自助登出默认放行；其余权限只使用认证中间件快照。
func RequirePerm(p perm.Permission) gin.HandlerFunc {
	return RequireAnyPerm(p)
}

// RequireAnyPerm 检查当前用户是否具备任一给定权限（OR）。
// 用于跨页面共享读接口（如看板与首页均可打开需求详情）。
func RequireAnyPerm(perms ...perm.Permission) gin.HandlerFunc {
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
		for _, p := range perms {
			if p == perm.AuthLogout || hasPermission(c, p) {
				c.Next()
				return
			}
		}
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

// RequireSuperAdmin 防止普通账号访问包含 SQL 与接口日志的诊断入口。
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if actor := CurrentUser(c); actor == nil || !actor.IsSuperAdmin {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}
