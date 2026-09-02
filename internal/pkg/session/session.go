// =============================================================================
// 文件: internal/pkg/session/session.go
// 模块: 基础设施
// 类型: infra
// 职责: 初始化会话管理并封装会话读写辅助方法。
// 依赖: internal/config
// =============================================================================

package session

import (
	"context"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"

	"workbench/internal/config"
)

const (
	userIDKey = "userID"
)

// New 创建并配置 SCS SessionManager。
func New(cfg *config.Config) *scs.SessionManager {
	mgr := scs.New()
	mgr.Lifetime = time.Duration(cfg.Session.LifetimeHours) * time.Hour
	mgr.Cookie.Name = cfg.Session.CookieName
	mgr.Cookie.HttpOnly = true
	mgr.Cookie.SameSite = http.SameSiteLaxMode
	mgr.Cookie.Secure = cfg.Session.CookieSecure
	mgr.Cookie.Path = "/"
	return mgr
}

// PutUserID 将登录用户 ID 写入会话。
func PutUserID(ctx context.Context, mgr *scs.SessionManager, userID int64) {
	mgr.Put(ctx, userIDKey, userID)
}

// GetUserID 从会话读取登录用户 ID。
func GetUserID(ctx context.Context, mgr *scs.SessionManager) int64 {
	val := mgr.Get(ctx, userIDKey)

	switch v := val.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

// Renew 轮换会话 token，常用于登录后防会话固定攻击。
func Renew(ctx context.Context, mgr *scs.SessionManager) error {
	return mgr.RenewToken(ctx)
}

// Clear 清空当前会话，常用于登出。
func Clear(ctx context.Context, mgr *scs.SessionManager) error {
	return mgr.Destroy(ctx)
}
