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

	"workbench/internal/module/agileteam"
	"workbench/internal/module/debug"
	"workbench/internal/module/dept"

	"workbench/internal/module/build"
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
	if err := ensureOperationLogSchema(db); err != nil {
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
	authSvc := login.NewService(authRepo, sessionMgr, zapLog)
	authHandler := login.NewHandler(authSvc, zapLog)
	requireLogin := middleware.RequireLogin(sessionMgr, db)
	redirectIfLoggedIn := middleware.RedirectIfLoggedIn(sessionMgr)
	userRepo := user.NewRepo(db)
	userSvc := user.NewService(userRepo)
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
	scheduleSvc := schedule.NewService(scheduleRepo, zapLog)
	scheduleHandler := schedule.NewHandler(rend, zapLog, scheduleSvc, strings.TrimRight(cfg.Zentao.URL, "/"))
	queryRepo := query.NewRepo(dbReadonlyOrPrimary(dbReadonly, db))
	querySvc := query.NewService(queryRepo)
	queryHandler := query.NewHandler(rend, querySvc, zapLog)
	metricsHandler := metrics.NewHandler(rend, metrics.NewService(metrics.NewRepo(dbReadonlyOrPrimary(dbReadonly, db))), zapLog)
	// PO 查询走只读池（可 nil 降级）；关注/已读写入必须走主库。
	poRepo := po.NewRepo(dbReadonly, db)
	poSvc := po.NewService(poRepo, scheduleSvc, userSvc, zapLog)
	poHandler := po.NewHandler(poSvc, zapLog)
	// 侧栏角标：把 poSvc.SidebarBadges 适配为 render 包的 provider。
	// render 包不反向 import po，避免循环依赖；只通过 provider 闭包注入。
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
	sqlPerfRepo := debug.NewRepo(cfg.Log.Dir)
	sqlPerfSvc := debug.NewService(sqlPerfRepo)
	sqlPerfHandler := debug.NewHandler(sqlPerfSvc)

	testtaskRepo := testtask.NewRepo(dbReadonlyOrPrimary(dbReadonly, db))
	testtaskSvc := testtask.NewService(testtaskRepo, userSvc, zentaopkg.DefaultClient(), zapLog)
	testtaskHandler := testtask.NewHandler(testtaskSvc, zapLog)

	buildRepo := build.NewRepo(dbReadonlyOrPrimary(dbReadonly, db))
	buildSvc := build.NewService(buildRepo, userSvc, zentaopkg.DefaultClient(), zapLog)

	// 个人资料：GET /profile 渲染深链页；顶栏入口同时挂 openProfileModal 弹窗；与整页共用 po-profile.js。
	profileHandler := profile.NewHandler(profile.NewService(profile.NewRepo(db)), zapLog)
	buildHandler := build.NewHandler(buildSvc, zapLog)

	agileTeamRepo := agileteam.NewRepo(db, dbReadonlyOrPrimary(dbReadonly, db))
	agileTeamSvc := agileteam.NewService(agileTeamRepo, zapLog)
	agileTeamHandler := agileteam.NewHandler(agileTeamSvc, zapLog)

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
		ScheduleHandler:     scheduleHandler,
		TesttaskHandler:     testtaskHandler,
		BuildHandler:        buildHandler,
		QueryHandler:        queryHandler,
		MetricsHandler:      metricsHandler,
		ProfileHandler:      profileHandler,
		AgileTeamHandler:    agileTeamHandler,
		SqlPerfHandler:      sqlPerfHandler,
	}

	srv := server.New(cfg, zapLog, db, sessionMgr, limiter, nil, routeDeps)
	return srv.Run()
}

func dbReadonlyOrPrimary(readonly, primary *gorm.DB) *gorm.DB {
	if readonly != nil {
		return readonly
	}
	return primary
}
