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
	"sync"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/menu"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/session"
)

var authColumnCache sync.Map // key: table.column, value: bool

// RequireLogin 要求请求已登录，否则重定向到登录页。
func RequireLogin(mgr *scs.SessionManager, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := mgr.GetInt64(c.Request.Context(), "userID")
		if userID <= 0 {
			userID = session.GetUserID(c.Request.Context(), mgr)
		}
		if userID <= 0 {
			target := "/login?redirect=" + url.QueryEscape(c.Request.URL.RequestURI())
			c.Redirect(http.StatusSeeOther, target)
			c.Abort()
			return
		}
		if db == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		var user model.User
		if err := loadUserByID(c.Request.Context(), db, userID, &user); err != nil {
			_ = session.Clear(c.Request.Context(), mgr)
			target := "/login?redirect=" + url.QueryEscape(c.Request.URL.RequestURI())
			c.Redirect(http.StatusSeeOther, target)
			c.Abort()
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
		currentMenus := menu.Filter(menus, userPerms, user.IsSuperAdmin)

		c.Set("currentUser", &user)
		c.Set("userPerms", userPerms)
		c.Set("currentMenus", currentMenus)
		c.Next()
	}
}

func loadUserByID(ctx context.Context, db *gorm.DB, userID int64, user *model.User) error {
	q := db.WithContext(ctx).Unscoped().Table("zt_gf_user").Where("id = ?", userID)
	if hasColumn(ctx, db, "zt_gf_user", "deleted") {
		q = q.Where("deleted = 0")
	}
	return q.Take(user).Error
}

func hasColumn(ctx context.Context, db *gorm.DB, table, column string) bool {
	cacheKey := table + "." + column
	if v, ok := authColumnCache.Load(cacheKey); ok {
		return v.(bool)
	}
	ok := db.WithContext(ctx).Migrator().HasColumn(table, column)
	authColumnCache.Store(cacheKey, ok)
	return ok
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
