// =============================================================================
// 文件: internal/bootstrap/bootstrap.go
// 模块: 基础设施
// 类型: infra
// 职责: 编排应用启动依赖并启动 HTTP 服务。
// 依赖: internal/config
//       internal/pkg/database
//       internal/pkg/flash
//       internal/pkg/logger
//       internal/pkg/menu
//       internal/pkg/ratelimit
//       internal/pkg/render
//       internal/pkg/session
//       internal/server
// =============================================================================

package bootstrap

import (
	"fmt"

	"workbrench/internal/config"
	"workbrench/internal/middleware"
	"workbrench/internal/model"
	"workbrench/internal/module/backup"
	"workbrench/internal/module/cronjob"
	"workbrench/internal/module/dept"
	"workbrench/internal/module/dictitem"
	"workbrench/internal/module/dicttype"
	"workbrench/internal/module/login"
	"workbrench/internal/module/loginlog"
	"workbrench/internal/module/menu"
	"workbrench/internal/module/operationlog"
	"workbrench/internal/module/role"
	"workbrench/internal/module/user"
	backuppkg "workbrench/internal/pkg/backup"
	"workbrench/internal/pkg/cron"
	"workbrench/internal/pkg/database"
	"workbrench/internal/pkg/flash"
	"workbrench/internal/pkg/logger"
	"workbrench/internal/pkg/ratelimit"
	"workbrench/internal/pkg/render"
	"workbrench/internal/pkg/session"
	"workbrench/internal/server"

	"go.uber.org/zap"
)

// Run 加载配置、初始化日志、启动 HTTP 服务。
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	zapLog, err := logger.Init(cfg)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer func() { _ = zapLog.Sync() }()

	db, err := database.New(cfg, zapLog)
	if err != nil {
		return fmt.Errorf("init database: %w", err)
	}
	defer func() { _ = database.Close(db) }()
	if err := db.AutoMigrate(&model.OperationLog{}); err != nil {
		return fmt.Errorf("ensure zt_operation_logs: %w", err)
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
	dictTypeRepo := dicttype.NewRepo(db)
	dictTypeSvc := dicttype.NewService(dictTypeRepo)
	dictTypeHandler := dicttype.NewHandler(rend, zapLog, dictTypeSvc)
	dictItemRepo := dictitem.NewRepo(db)
	dictItemSvc := dictitem.NewService(dictItemRepo)
	dictItemHandler := dictitem.NewHandler(rend, zapLog, dictItemSvc)
	roleRepo := role.NewRepo(db)
	roleSvc := role.NewService(roleRepo)
	roleHandler := role.NewHandler(rend, zapLog, roleSvc)

	cronMgr := cron.New(db)
	if err := cronMgr.Register("builtin_auto_backup", cfg.Backup.AutoCron, func() {
		if _, backupErr := backuppkg.Run(backuppkg.Config{
			DB:       db,
			Host:     cfg.Database.Host,
			Port:     cfg.Database.Port,
			User:     cfg.Database.User,
			Password: cfg.Database.Password,
			DBName:   cfg.Database.DBName,
			Dir:      cfg.Backup.Dir,
			KeepDays: cfg.Backup.KeepDays,
		}); backupErr != nil {
			zapLog.Error("auto backup failed", zap.Error(backupErr))
		}
	}); err != nil {
		return fmt.Errorf("register auto backup cron: %w", err)
	}
	cronJobRepo := cronjob.NewRepo(db)
	cronJobSvc := cronjob.NewService(cronJobRepo, cronMgr)
	cronJobHandler := cronjob.NewHandler(rend, zapLog, cronJobSvc)
	backupRepo := backup.NewRepo(db)
	backupSvc := backup.NewService(backupRepo, cfg, db)
	backupHandler := backup.NewHandler(zapLog, backupSvc)

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
		DictTypeHandler:     dictTypeHandler,
		DictItemHandler:     dictItemHandler,
		RoleHandler:         roleHandler,
		CronJobHandler:      cronJobHandler,
		BackupHandler:       backupHandler,
	}

	cronMgr.Start()
	defer cronMgr.Stop()

	srv := server.New(cfg, zapLog, db, sessionMgr, limiter, nil, routeDeps)
	return srv.Run()
}
