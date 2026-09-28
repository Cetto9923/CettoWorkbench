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
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"workbench/internal/config"
	"workbench/internal/middleware"
	"workbench/internal/model"

	"workbench/internal/module/debug"
	"workbench/internal/module/dept"

	"workbench/internal/module/agileteam"
	"workbench/internal/module/build"
	"workbench/internal/module/follow"
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
	if err := db.AutoMigrate(&model.OperationLog{}); err != nil {
		return fmt.Errorf("ensure zt_operation_logs: %w", err)
	}

	// 价值流只读备库：失败不阻断启动，PO 价值流降级为空阶段
	var dbReadonly *gorm.DB
	if strings.TrimSpace(cfg.DatabaseReadonly.Host) != "" {
		ro, roErr := database.Open(cfg.DatabaseReadonly)
		if roErr != nil {
			zapLog.Warn("init databaseReadonly failed, value stream will degrade", zap.Error(roErr))
		} else {
			dbReadonly = ro
			defer func() { _ = database.Close(dbReadonly) }()
		}
	} else {
		zapLog.Warn("databaseReadonly.host empty, value stream will degrade")
	}
	sessionMgr := session.New(cfg)
	flash.SetDefault(sessionMgr)
	limiter := ratelimit.New(10, 20)
	loginLimiter := ratelimit.New(3.0, 10)

	isDev := cfg.App.Env == "dev"
	rend, err := render.New(cfg, isDev)
	if err != nil {
		return fmt.Errorf("init render: %w", err)
	}
	render.SetDefault(rend)

	authRepo := login.NewRepo(db)
	userRepo := user.NewRepo(db)
	userSvc := user.NewService(userRepo, zentaopkg.API())
	authSvc := login.NewService(authRepo, sessionMgr, zapLog, userSvc)
	authHandler := login.NewHandler(authSvc, zapLog)
	requireLogin := middleware.RequireLogin(sessionMgr, db)
	redirectIfLoggedIn := middleware.RedirectIfLoggedIn(sessionMgr)
	userHandler := user.NewHandler(rend, zapLog, userSvc)
	loginLogRepo := loginlog.NewRepo(db)
	loginLogSvc := loginlog.NewService(loginLogRepo)
	loginLogHandler := loginlog.NewHandler(loginLogSvc)
	operationLogRepo := operationlog.NewRepo(db)
	operationLogSvc := operationlog.NewService(operationLogRepo)
	operationLogHandler := operationlog.NewHandler(operationLogSvc)
	menuRepo := menu.NewRepo(db)
	menuSvc := menu.NewService(menuRepo)
	menuHandler := menu.NewHandler(rend, zapLog, menuSvc)
	deptRepo := dept.NewRepo(db)
	deptSvc := dept.NewService(deptRepo)
	deptHandler := dept.NewHandler(rend, zapLog, deptSvc)
	roleRepo := role.NewRepo(db)
	roleSvc := role.NewService(roleRepo)
	roleHandler := role.NewHandler(rend, zapLog, roleSvc)

	scheduleRepo := schedule.NewRepo(db)
	scheduleSvc := schedule.NewService(scheduleRepo, userSvc, deptSvc, zentaopkg.API(), zapLog)
	scheduleHandler := schedule.NewHandler(rend, zapLog, scheduleSvc, strings.TrimRight(cfg.Zentao.URL, "/"))
	poRepo := po.NewRepo(dbReadonly, db)
	poSvc := po.NewService(poRepo, scheduleSvc, userSvc, zentaopkg.API(), zapLog)
	poHandler := po.NewHandler(poSvc, zapLog)
	// 侧栏角标：注入 poSvc.SidebarBadges 为 render provider。
	rend.SetSidebarBadgesProvider(func(c *gin.Context) (render.SidebarBadges, error) {
		v, ok := c.Get("currentUser")
		if !ok {
			return render.SidebarBadges{}, nil
		}
		u, ok := v.(*model.User)
		if !ok || u == nil {
			return render.SidebarBadges{}, nil
		}
		b, err := poSvc.SidebarBadges(c.Request.Context(), &po.SidebarActor{Account: u.Account, ID: u.ID})
		if err != nil {
			return render.SidebarBadges{Todos: b.Todos, Done: b.Done, Notice: b.Notice}, err
		}
		return render.SidebarBadges{Todos: b.Todos, Done: b.Done, Notice: b.Notice}, nil
	})
	testtaskRepo := testtask.NewRepo(db)
	testtaskSvc := testtask.NewService(testtaskRepo, userSvc, zentaopkg.API(), zapLog)
	testtaskHandler := testtask.NewHandler(testtaskSvc, zapLog)
	buildRepo := build.NewRepo(db)
	buildSvc := build.NewService(buildRepo, userSvc, zentaopkg.API(), zapLog)
	buildHandler := build.NewHandler(buildSvc, zapLog)
	followHandler := follow.NewHandler(zapLog)
	kanbanReadDB := dbReadonly
	if kanbanReadDB == nil {
		kanbanReadDB = db
	}
	kanbanRepo := kanban.NewRepo(kanbanReadDB)
	kanbanSvc := kanban.NewService(kanbanRepo, userSvc, poSvc, zentaopkg.API())
	kanbanHandler := kanban.NewHandler(kanbanSvc, zapLog)
	sqlPerfRepo := debug.NewRepo(cfg.Log.Dir)
	sqlPerfSvc := debug.NewService(sqlPerfRepo)
	sqlPerfHandler := debug.NewHandler(sqlPerfSvc)

	queryRepo := query.NewRepo(dbReadonlyOrPrimary(dbReadonly, db))
	querySvc := query.NewService(queryRepo)
	queryHandler := query.NewHandler(rend, querySvc, zapLog)
	metricsHandler := metrics.NewHandler(rend, metrics.NewService(metrics.NewRepo(dbReadonlyOrPrimary(dbReadonly, db))), zapLog)
	profileHandler := profile.NewHandler(profile.NewService(profile.NewRepo(db)), zapLog)
	agileTeamRepo := agileteam.NewRepo(db, dbReadonlyOrPrimary(dbReadonly, db))
	agileTeamSvc := agileteam.NewService(agileTeamRepo, zapLog)
	agileTeamHandler := agileteam.NewHandler(agileTeamSvc, zapLog)
	poHandler.SetTeamViewAccess(agileTeamSvc.CanEnterDashboard)
	poHandler.SetTeamScopeAccounts(agileTeamSvc.DashboardAccounts)
	poHandler.SetTeamScopeGroupIDs(agileTeamSvc.DashboardGroupIDs)

	routeDeps := server.RouteDeps{
		SessionMgr:          sessionMgr,
		DB:                  db,
		RequireLogin:        requireLogin,
		RedirectIfLoggedIn:  redirectIfLoggedIn,
		LoginLimiter:        loginLimiter,
		AuthHandler:         authHandler,
		UserHandler:         userHandler,
		LoginLogHandler:     loginLogHandler,
		OperationLogHandler: operationLogHandler,
		MenuHandler:         menuHandler,
		DeptHandler:         deptHandler,
		RoleHandler:         roleHandler,
		PoHandler:           poHandler,
		QueryHandler:        queryHandler,
		MetricsHandler:      metricsHandler,
		ProfileHandler:      profileHandler,
		AgileTeamHandler:    agileTeamHandler,
		FollowHandler:       followHandler,
		KanbanHandler:       kanbanHandler,
		ScheduleHandler:     scheduleHandler,
		TesttaskHandler:     testtaskHandler,
		BuildHandler:        buildHandler,
		SqlPerfHandler:      sqlPerfHandler,
	}

	srv := server.New(cfg, zapLog, db, sessionMgr, limiter, nil, routeDeps)
	return srv.Run()
}

// dbReadonlyOrPrimary 只读池可用则用之，否则降级主库。
func dbReadonlyOrPrimary(ro, primary *gorm.DB) *gorm.DB {
	if ro != nil {
		return ro
	}
	return primary
}
