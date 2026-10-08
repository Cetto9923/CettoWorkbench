// =============================================================================
// 文件: internal/bootstrap/bootstrap.go
// 模块: 基础设施
// 类型: infra
// 职责: 编排应用启动依赖并启动 HTTP 服务。
// 依赖: internal/config
//       internal/pkg/database
//       internal/pkg/flash
//       internal/pkg/logger
//       internal/pkg/sqllog
//       internal/pkg/zentao
//       internal/pkg/menu
//       internal/pkg/ratelimit
//       internal/pkg/render
//       internal/pkg/session
//       internal/server
// =============================================================================

package bootstrap

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"workbench/internal/config"
	"workbench/internal/middleware"

	"workbench/internal/module/debug"
	"workbench/internal/module/dept"

	"workbench/internal/module/agileteam"
	"workbench/internal/module/build"
	"workbench/internal/module/kanban"
	"workbench/internal/module/login"
	"workbench/internal/module/loginlog"
	"workbench/internal/module/menu"
	"workbench/internal/module/metrics"
	"workbench/internal/module/operationlog"
	"workbench/internal/module/po"
	"workbench/internal/module/profile"
	"workbench/internal/module/query"
	"workbench/internal/module/role"
	"workbench/internal/module/schedule"
	"workbench/internal/module/testtask"
	"workbench/internal/module/user"
	"workbench/internal/pkg/database"
	"workbench/internal/pkg/flash"
	"workbench/internal/pkg/logger"
	"workbench/internal/pkg/ratelimit"
	"workbench/internal/pkg/render"
	"workbench/internal/pkg/session"
	"workbench/internal/pkg/sqllog"
	zentaopkg "workbench/internal/pkg/zentao"
	"workbench/internal/server"
)

// Run 加载配置、初始化日志、启动 HTTP 服务。
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	zentaopkg.SetConfig(cfg.Zentao)

	zapLog, err := logger.Init(cfg)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer func() { _ = zapLog.Sync() }()

	if err := sqllog.Init(cfg); err != nil {
		return fmt.Errorf("init sql log: %w", err)
	}
	defer func() { _ = sqllog.Sync() }()

	if err := zentaopkg.InitAPILog(cfg); err != nil {
		return fmt.Errorf("init zentao api log: %w", err)
	}
	defer func() { _ = zentaopkg.SyncAPILog() }()

	db, err := database.New(cfg)
	if err != nil {
		return fmt.Errorf("init database: %w", err)
	}
	defer func() { _ = database.Close(db) }()
	if err := database.CheckSchema(db); err != nil {
		return err
	}

	// 价值流只读备库：失败不阻断启动，PO 价值流降级为空阶段
	dbReadonly := openReadonlyPool(cfg, zapLog)
	if dbReadonly != nil {
		defer func() { _ = database.Close(dbReadonly) }()
	}

	sessionMgr := session.New(cfg)
	flash.SetDefault(sessionMgr)
	limiter := ratelimit.New(10, 20)

	isDev := cfg.App.Env == "dev"
	rend, err := render.New(cfg, isDev)
	if err != nil {
		return fmt.Errorf("init render: %w", err)
	}
	render.SetDefault(rend)

	w := &wiring{
		cfg:        cfg,
		db:         db,
		dbReadonly: dbReadonly,
		rend:       rend,
		zapLog:     zapLog,
		sessionMgr: sessionMgr,
		deps: server.RouteDeps{
			SessionMgr:   sessionMgr,
			DB:           db,
			LoginLimiter: ratelimit.New(3.0, 10),
		},
	}
	newBaseModules(w)
	newBusinessModules(w)

	srv := server.New(cfg, zapLog, db, sessionMgr, limiter, nil, w.deps)
	return srv.Run()
}

// openReadonlyPool 打开价值流只读备库；未配置或打开失败时返回 nil，由调用方降级。
func openReadonlyPool(cfg *config.Config, zapLog *zap.Logger) *gorm.DB {
	if strings.TrimSpace(cfg.DatabaseReadonly.Host) == "" {
		zapLog.Warn("databaseReadonly.host empty, value stream will degrade")
		return nil
	}
	ro, err := database.Open(cfg.DatabaseReadonly)
	if err != nil {
		zapLog.Warn("init databaseReadonly failed, value stream will degrade", zap.Error(err))
		return nil
	}
	return ro
}

// wiring 汇集模块装配所需的依赖，装配出的 Handler 写入 deps。
type wiring struct {
	cfg        *config.Config
	db         *gorm.DB
	dbReadonly *gorm.DB
	rend       *render.Renderer
	zapLog     *zap.Logger
	sessionMgr *scs.SessionManager
	userSvc    *user.Service
	deptSvc    *dept.Service
	deps       server.RouteDeps
}

// newBaseModules 装配用户、日志、菜单、部门、角色等基础模块。
func newBaseModules(w *wiring) {
	authRepo := login.NewRepo(w.db)
	w.userSvc = user.NewService(user.NewRepo(w.db), zentaopkg.API())
	authSvc := login.NewService(authRepo, w.sessionMgr, w.zapLog, w.userSvc)
	w.deps.AuthHandler = login.NewHandler(authSvc, w.zapLog)
	w.deps.RequireLogin = middleware.RequireLogin(w.sessionMgr, w.db)
	w.deps.RedirectIfLoggedIn = middleware.RedirectIfLoggedIn(w.sessionMgr)
	w.deps.UserHandler = user.NewHandler(w.rend, w.zapLog, w.userSvc)
	w.deps.LoginLogHandler = loginlog.NewHandler(loginlog.NewService(loginlog.NewRepo(w.db)))
	w.deps.OperationLogHandler = operationlog.NewHandler(operationlog.NewService(operationlog.NewRepo(w.db)))
	w.deps.MenuHandler = menu.NewHandler(w.rend, w.zapLog, menu.NewService(menu.NewRepo(w.db)))
	w.deptSvc = dept.NewService(dept.NewRepo(w.db))
	w.deps.DeptHandler = dept.NewHandler(w.rend, w.zapLog, w.deptSvc)
	w.deps.RoleHandler = role.NewHandler(w.rend, w.zapLog, role.NewService(role.NewRepo(w.db)))
}

// newBusinessModules 装配需求、排期、测试、提测、看板等业务模块。
func newBusinessModules(w *wiring) {
	scheduleSvc := schedule.NewService(schedule.NewRepo(w.db), w.userSvc, w.deptSvc, zentaopkg.API(), w.zapLog)
	w.deps.ScheduleHandler = schedule.NewHandler(w.rend, w.zapLog, scheduleSvc, strings.TrimRight(w.cfg.Zentao.URL, "/"))

	poSvc := po.NewService(po.NewRepo(w.dbReadonly, w.db), scheduleSvc, w.userSvc, zentaopkg.API(), w.zapLog)
	poHandler := po.NewHandler(poSvc, w.zapLog)
	w.deps.PoHandler = poHandler
	w.rend.SetSidebarBadgesProvider(sidebarBadgesProvider(poSvc))

	testtaskSvc := testtask.NewService(testtask.NewRepo(w.db), w.userSvc, zentaopkg.API(), w.zapLog)
	w.deps.TesttaskHandler = testtask.NewHandler(testtaskSvc, w.zapLog)
	buildSvc := build.NewService(build.NewRepo(w.db), w.userSvc, zentaopkg.API(), w.zapLog)
	w.deps.BuildHandler = build.NewHandler(buildSvc, w.zapLog)
	kanbanSvc := kanban.NewService(kanban.NewRepo(dbReadonlyOrPrimary(w.dbReadonly, w.db)), w.userSvc, poSvc, zentaopkg.API())
	w.deps.KanbanHandler = kanban.NewHandler(kanbanSvc, w.zapLog)
	w.deps.SqlPerfHandler = debug.NewHandler(debug.NewService(debug.NewRepo(w.cfg.Log.Dir)))

	readDB := dbReadonlyOrPrimary(w.dbReadonly, w.db)
	w.deps.QueryHandler = query.NewHandler(w.rend, query.NewService(query.NewRepo(readDB)), w.zapLog)
	w.deps.MetricsHandler = metrics.NewHandler(w.rend, metrics.NewService(metrics.NewRepo(readDB)), w.zapLog)
	w.deps.ProfileHandler = profile.NewHandler(profile.NewService(profile.NewRepo(w.db)), w.zapLog)

	agileTeamSvc := agileteam.NewService(agileteam.NewRepo(w.db, readDB), w.zapLog)
	w.deps.AgileTeamHandler = agileteam.NewHandler(agileTeamSvc, w.zapLog)
	poHandler.SetTeamViewAccess(agileTeamSvc.CanEnterDashboard)
	poHandler.SetTeamScopeAccounts(agileTeamSvc.DashboardAccounts)
	poHandler.SetTeamScopeGroupIDs(agileTeamSvc.DashboardGroupIDs)
}

// sidebarBadgesProvider 侧栏角标：注入 poSvc.SidebarBadges 为 render provider。
func sidebarBadgesProvider(poSvc *po.Service) func(*gin.Context) (render.SidebarBadges, error) {
	return func(c *gin.Context) (render.SidebarBadges, error) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 250*time.Millisecond)
		defer cancel()
		return poSvc.SidebarBadges(ctx, middleware.CurrentUser(c))
	}
}

// dbReadonlyOrPrimary 只读池可用则用之，否则降级主库。
func dbReadonlyOrPrimary(ro, primary *gorm.DB) *gorm.DB {
	if ro != nil {
		return ro
	}
	return primary
}
