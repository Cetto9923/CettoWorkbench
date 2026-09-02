// =============================================================================
// 文件: internal/middleware/auth.go
// 模块: 中间件
// 类型: middleware
// 职责: 校验登录态并注入当前用户上下文。
// 依赖: internal/model
//       internal/pkg/menu
//       internal/pkg/session
// =============================================================================

package middleware

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/menu"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/session"
)

// RequireLogin 要求请求已登录，否则重定向到登录页。
func RequireLogin(mgr *scs.SessionManager, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := mgr.GetInt64(c.Request.Context(), "userID")
		if userID <= 0 {
			userID = session.GetUserID(c.Request.Context(), mgr)
		}
		if userID <= 0 {
			abortUnauthenticated(c)
			return
		}
		if db == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		var user model.User
		if err := loadUserByID(c.Request.Context(), db, userID, &user); err != nil {
			_ = session.Clear(c.Request.Context(), mgr)
			abortUnauthenticated(c)
			return
		}

		userPerms := map[string]bool{
			perm.AuthLogout.String(): true,
		}
		if user.IsSuperAdmin {
			for _, p := range perm.All() {
				userPerms[p.String()] = true
			}
		} else {
			loadedPerms, err := perm.LoadUserPermissionSet(c.Request.Context(), db, user.ID)
			if err != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			for code := range loadedPerms {
				userPerms[code] = true
			}
		}
		menus, err := menu.LoadFromDB(c.Request.Context(), db)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		// 当前系统不做权限过滤
		currentMenus := menus
		// currentMenus := menu.Filter(menus, userPerms, user.IsSuperAdmin)

		c.Set("currentUser", &user)
		c.Set("userPerms", userPerms)
		c.Set("currentMenus", currentMenus)
		c.Next()
	}
}

func expectsJSON(c *gin.Context) bool {
	accept := strings.ToLower(c.GetHeader("Accept"))
	requestedWith := strings.ToLower(c.GetHeader("X-Requested-With"))
	return strings.Contains(accept, "application/json") || requestedWith == "xmlhttprequest"
}

func abortUnauthenticated(c *gin.Context) {
	if expectsJSON(c) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "未登录或会话已过期",
		})
		return
	}
	target := "/login?redirect=" + url.QueryEscape(c.Request.URL.RequestURI())
	c.Redirect(http.StatusSeeOther, target)
	c.Abort()
}

func loadUserByID(ctx context.Context, db *gorm.DB, userID int64, user *model.User) error {
	return db.WithContext(ctx).
		Where("id = ? AND deleted = ?", userID, "0").
		First(user).Error
}

// TODO 返回用户信息、权限
// CurrentUser 获取当前已登录用户，未登录返回 nil。
func CurrentUser(c *gin.Context) *model.User {
	v, ok := c.Get("currentUser")
	if !ok {
		return nil
	}
	u, _ := v.(*model.User)
	return u
}

// MustLogin 返回当前用户，不存在时 panic。
func MustLogin(c *gin.Context) *model.User {
	u := CurrentUser(c)
	if u == nil {
		panic("current user not found")
	}
	return u
}
