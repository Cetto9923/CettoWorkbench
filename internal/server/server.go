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
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
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
	t, err := loadTemplates("web/templates")
	if err != nil {
		zapLog.Panic("load templates failed", zap.Error(err))
	}
	r.SetHTMLTemplate(t)

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

// loadTemplates 递归加载目录下全部 html 模板，并注入基础 FuncMap。
func loadTemplates(templatesDir string) (*template.Template, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(templatesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".html") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("no html template found in %s", templatesDir)
	}
	funcMap := template.FuncMap{
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
		"asset": func(path string) string {
			return path
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"alertclass": func(level string) string {
			switch strings.ToLower(strings.TrimSpace(level)) {
			case "success":
				return "success"
			case "error":
				return "danger"
			case "warning":
				return "warning"
			case "info":
				return "info"
			default:
				return "secondary"
			}
		},
		"menuNavActive": menu.MenuNavActive,
	}
	return template.New("").Funcs(funcMap).ParseFiles(files...)
}

// Run 启动 HTTP 服务，并在收到 SIGINT/SIGTERM 时优雅关闭（最长等待 30 秒）。
func (s *Server) Run() error {
	s.engine.Use(func(c *gin.Context) {
		c.Set("db", s.db)
		c.Set("sessionMgr", s.sessionMgr)
		c.Set("menus", s.menus)
		c.Next()
	})

	s.engine.Use(middleware.Recovery(s.logger))
	s.engine.Use(ratelimitpkg.NewGlobalLimiter(s.globalRPS))
	s.engine.Use(middleware.RequestLogger(s.logger))
	s.engine.Use(middleware.SecureHeaders())
	s.engine.Use(middleware.MethodOverride())
	if s.sessionMgr != nil {
		s.engine.Use(wrapStdMiddleware(s.sessionMgr.LoadAndSave))
	}
	// s.engine.Use(wrapStdMiddleware(middleware.CSRF()))

	registerRoutes(s.engine, s.routeDeps)

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

func wrapStdMiddleware(m func(http.Handler) http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		nextCalled := false
		var next http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			c.Request = r
			c.Next()
		}
		m(next).ServeHTTP(c.Writer, c.Request)
		if !nextCalled {
			// 标准库中间件已直接完成响应（如 CSRF 校验失败），
			// 需要显式中止 Gin 后续处理器，避免重复写 Header。
			c.Abort()
		}
	}
}
