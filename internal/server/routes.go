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

	"workbench/internal/middleware"

	"workbench/internal/module/debug"
	"workbench/internal/module/dept"

	loginmodule "workbench/internal/module/login"
	"workbench/internal/module/loginlog"
	menumodule "workbench/internal/module/menu"
	"workbench/internal/module/operationlog"
	pomodule "workbench/internal/module/po"
	"workbench/internal/module/role"
	"workbench/internal/module/schedule"
	"workbench/internal/module/user"
	"workbench/internal/pkg/perm"
	ratelimitpkg "workbench/internal/pkg/ratelimit"
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
	RoleHandler         *role.Handler
	PoHandler           *pomodule.Handler
	ScheduleHandler     *schedule.Handler
	SqlPerfHandler      *debug.Handler
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
		if deps.RoleHandler != nil {
			deps.RoleHandler.RegisterRoutes(admin)
		}
	}

	// 排期工作台（菜单 path：/po/schedule）
	if deps.ScheduleHandler != nil {
		r.GET("/po/schedule",
			middleware.RequireLogin(deps.SessionMgr, deps.DB),
			middleware.RecordOperationLog(deps.DB, deps.SessionMgr),
			middleware.RequirePerm(perm.ScheduleList),
			middleware.ActiveNav("/po/schedule"),
			deps.ScheduleHandler.Index,
		)
		r.GET("/po/schedule/matching-plans",
			middleware.RequireLogin(deps.SessionMgr, deps.DB),
			middleware.RecordOperationLog(deps.DB, deps.SessionMgr),
			middleware.RequirePerm(perm.ScheduleList),
			deps.ScheduleHandler.GetMatchingPlans,
		)
		r.POST("/po/schedule/windows",
			middleware.RequireLogin(deps.SessionMgr, deps.DB),
			middleware.RecordOperationLog(deps.DB, deps.SessionMgr),
			middleware.RequirePerm(perm.ScheduleCreate),
			deps.ScheduleHandler.CreateWindow,
		)
		r.GET("/po/schedule/windows",
			middleware.RequireLogin(deps.SessionMgr, deps.DB),
			middleware.RecordOperationLog(deps.DB, deps.SessionMgr),
			middleware.RequirePerm(perm.ScheduleList),
			deps.ScheduleHandler.ListWindows,
		)
		r.GET("/po/schedule/windows/:id",
			middleware.RequireLogin(deps.SessionMgr, deps.DB),
			middleware.RecordOperationLog(deps.DB, deps.SessionMgr),
			middleware.RequirePerm(perm.ScheduleList),
			deps.ScheduleHandler.GetWindow,
		)
		r.PUT("/po/schedule/windows/:id",
			middleware.RequireLogin(deps.SessionMgr, deps.DB),
			middleware.RecordOperationLog(deps.DB, deps.SessionMgr),
			middleware.RequirePerm(perm.ScheduleUpdate),
			deps.ScheduleHandler.UpdateWindow,
		)
		r.DELETE("/po/schedule/windows/:id",
			middleware.RequireLogin(deps.SessionMgr, deps.DB),
			middleware.RecordOperationLog(deps.DB, deps.SessionMgr),
			middleware.RequirePerm(perm.ScheduleDelete),
			deps.ScheduleHandler.DeleteWindow,
		)
	}

	// PO 工作台（菜单 path：/po/home）
	po := r.Group("/po")
	po.Use(middleware.RequireLogin(deps.SessionMgr, deps.DB))
	po.Use(middleware.RecordOperationLog(deps.DB, deps.SessionMgr))
	{
		if deps.PoHandler != nil {
			deps.PoHandler.RegisterRoutes(po)
		}
	}

	debugGroup := r.Group("/debug")
	{
		if deps.SqlPerfHandler != nil {
			deps.SqlPerfHandler.RegisterRoutes(debugGroup)
		}
	}
}
