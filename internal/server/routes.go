// =============================================================================
// 文件: internal/server/routes.go
// 模块: 基础设施
// 类型: infra
// 职责: 注册系统路由与路由分组中间件。
// 依赖: internal/middleware
// =============================================================================

package server

import (
	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"goframework/internal/middleware"
	"goframework/internal/module/backup"
	"goframework/internal/module/cronjob"
	"goframework/internal/module/dept"
	"goframework/internal/module/dictitem"
	"goframework/internal/module/dicttype"
	"goframework/internal/module/ledger"
	loginmodule "goframework/internal/module/login"
	"goframework/internal/module/loginlog"
	menumodule "goframework/internal/module/menu"
	"goframework/internal/module/operationlog"
	"goframework/internal/module/resource"
	"goframework/internal/module/role"
	"goframework/internal/module/user"
	"goframework/internal/module/zentao"
	ratelimitpkg "goframework/internal/pkg/ratelimit"
)

// RouteDeps 路由注册依赖。
type RouteDeps struct {
	SessionMgr          *scs.SessionManager
	DB                  *gorm.DB
	RequireLogin        gin.HandlerFunc
	RedirectIfLoggedIn  gin.HandlerFunc
	LoginLimiter        *ratelimitpkg.Limiter
	AuthHandler         *loginmodule.Handler
	UserHandler         *user.Handler
	LoginLogHandler     *loginlog.Handler
	OperationLogHandler *operationlog.Handler
	MenuHandler         *menumodule.Handler
	DeptHandler         *dept.Handler
	DictTypeHandler     *dicttype.Handler
	DictItemHandler     *dictitem.Handler
	RoleHandler         *role.Handler
	CronJobHandler      *cronjob.Handler
	BackupHandler       *backup.Handler
	ResourceHandler     *resource.Handler
	ZentaoHandler       *zentao.Handler
	LedgerHandler       *ledger.Handler
}

func registerRoutes(r *gin.Engine, deps RouteDeps) {
	// auth 模块自注册（含公开路由和需登录路由）
	if deps.AuthHandler != nil {
		deps.AuthHandler.RegisterRoutes(r, deps.RequireLogin, deps.RedirectIfLoggedIn, deps.LoginLimiter)
	}

	// 管理后台路由组（需登录）
	admin := r.Group("/admin")
	admin.Use(middleware.RequireLogin(deps.SessionMgr, deps.DB))
	admin.Use(middleware.RecordOperationLog(deps.DB, deps.SessionMgr))
	{
		admin.GET("/dashboard", middleware.ActiveNav("/admin/dashboard"), DashboardHandler)
		if deps.UserHandler != nil {
			deps.UserHandler.RegisterRoutes(admin)
		}
		if deps.LoginLogHandler != nil {
			deps.LoginLogHandler.RegisterRoutes(admin)
		}
		if deps.OperationLogHandler != nil {
			deps.OperationLogHandler.RegisterRoutes(admin)
		}
		if deps.MenuHandler != nil {
			deps.MenuHandler.RegisterRoutes(admin)
		}
		if deps.DeptHandler != nil {
			deps.DeptHandler.RegisterRoutes(admin)
		}
		if deps.DictTypeHandler != nil {
			deps.DictTypeHandler.RegisterRoutes(admin)
		}
		if deps.DictItemHandler != nil {
			deps.DictItemHandler.RegisterRoutes(admin)
		}
		if deps.RoleHandler != nil {
			deps.RoleHandler.RegisterRoutes(admin)
		}
		if deps.CronJobHandler != nil {
			deps.CronJobHandler.RegisterRoutes(admin)
		}
		if deps.BackupHandler != nil {
			deps.BackupHandler.RegisterRoutes(admin)
		}

		// 后续各模块：
		// deps.XxxHandler.RegisterRoutes(admin)
	}

	ledgerGrp := r.Group("/ledger")
	ledgerGrp.Use(middleware.RequireLogin(deps.SessionMgr, deps.DB))
	ledgerGrp.Use(middleware.RecordOperationLog(deps.DB, deps.SessionMgr))
	{
		if deps.LedgerHandler != nil {
			deps.LedgerHandler.RegisterRoutes(ledgerGrp)
		}
	}

	zentao := r.Group("/zentao")
	zentao.Use(middleware.RequireLogin(deps.SessionMgr, deps.DB))
	{
		if deps.ZentaoHandler != nil {
			deps.ZentaoHandler.RegisterRoutes(zentao)
		}
	}

	resources := r.Group("/resources")
	resources.Use(middleware.RequireLogin(deps.SessionMgr, deps.DB))
	{
		if deps.ResourceHandler != nil {
			deps.ResourceHandler.RegisterRoutes(resources)
		}
	}

}
