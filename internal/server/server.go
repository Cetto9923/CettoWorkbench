// =============================================================================
// 文件: internal/server/server.go
// 模块: 基础设施
// 类型: infra
// 职责: 初始化 Gin 与 HTTP Server 并管理服务生命周期。
// 依赖: internal/config
//       internal/middleware
//       internal/pkg/menu
//       internal/pkg/ratelimit
// =============================================================================

package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"workbench/internal/config"
	"workbench/internal/middleware"
	"workbench/internal/pkg/menu"
	ratelimitpkg "workbench/internal/pkg/ratelimit"
)

// Server 封装 Gin 与 HTTP Server。
type Server struct {
	httpServer *http.Server
	engine     *gin.Engine
	logger     *zap.Logger
	db         *gorm.DB
	sessionMgr *scs.SessionManager
	limiter    *ratelimitpkg.Limiter
	globalRPS  int
	menus      []menu.Menu
	routeDeps  RouteDeps
}

// New 创建 Server；根据 App.Env 设置 Gin 模式并注册路由。
func New(
	cfg *config.Config,
	zapLog *zap.Logger,
	db *gorm.DB,
	sessionMgr *scs.SessionManager,
	limiter *ratelimitpkg.Limiter,
	menus []menu.Menu,
	routeDeps RouteDeps,
) *Server {
	if strings.EqualFold(cfg.App.Env, "prod") {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Static("/static", "web/static")

	return &Server{
		httpServer: &http.Server{
			Addr:    cfg.App.Addr,
			Handler: r,
		},
		engine:     r,
		logger:     zapLog,
		db:         db,
		sessionMgr: sessionMgr,
		limiter:    limiter,
		globalRPS:  cfg.RateLimit.GlobalRPS,
		menus:      menus,
		routeDeps:  routeDeps,
	}
}

// Run 启动 HTTP 服务，并在收到 SIGINT/SIGTERM 时优雅关闭（最长等待 30 秒）。
func (s *Server) Run() error {
	s.engine.Use(func(c *gin.Context) {
		c.Set("db", s.db)
		c.Set("sessionMgr", s.sessionMgr)
		c.Set("menus", s.menus)
		c.Next()
	})

	s.engine.Use(middleware.SQLRequestContext())
	s.engine.Use(middleware.Recovery(s.logger))
	s.engine.Use(ratelimitpkg.NewGlobalLimiter(s.globalRPS))
	s.engine.Use(middleware.RequestLogger(s.logger))
	s.engine.Use(middleware.SecureHeaders())
	s.engine.Use(middleware.MethodOverride())

	registerRoutes(s.engine, s.routeDeps)

	var handler http.Handler = s.engine
	if s.sessionMgr != nil {
		// 必须包在 gin.Engine 外层：SCS 才能拦截 gin 的 WriteHeader，登录 303 才会带上 Set-Cookie。
		handler = s.sessionMgr.LoadAndSave(s.engine)
	}
	s.httpServer.Handler = handler

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-quit:
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}
