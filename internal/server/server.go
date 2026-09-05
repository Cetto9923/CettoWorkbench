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
	"sync"
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
	httpServer   *http.Server
	engine       *gin.Engine
	logger       *zap.Logger
	db           *gorm.DB
	sessionMgr   *scs.SessionManager
	limiter      *ratelimitpkg.Limiter
	globalRPS    int
	menus        []menu.Menu
	routeDeps    RouteDeps
	cookieSecure bool
	initOnce     sync.Once
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
	if cfg != nil && strings.EqualFold(cfg.App.Env, "prod") {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Static("/static", "web/static")
	t, err := loadTemplates(findTemplatesDir())
	if err != nil {
		zapLog.Panic("load templates failed", zap.Error(err))
	}
	r.SetHTMLTemplate(t)

	addr := ""
	globalRPS := 0
	cookieSecure := false
	if cfg != nil {
		addr = cfg.App.Addr
		globalRPS = cfg.RateLimit.GlobalRPS
		cookieSecure = cfg.Session.CookieSecure
	}

	return &Server{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: r,
		},
		engine:       r,
		logger:       zapLog,
		db:           db,
		sessionMgr:   sessionMgr,
		limiter:      limiter,
		globalRPS:    globalRPS,
		menus:        menus,
		routeDeps:    routeDeps,
		cookieSecure: cookieSecure,
	}
}

func findTemplatesDir() string {
	candidate := filepath.Clean("web/templates")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	dir, err := os.Getwd()
	if err == nil {
		for i := 0; i < 5; i++ {
			p := filepath.Join(dir, "web/templates")
			if _, err := os.Stat(p); err == nil {
				return p
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return candidate
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

// Setup 注册全局中间件与业务路由（幂等）。
func (s *Server) Setup() {
	s.initOnce.Do(func() {
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
	})
}

// BuildHandler 组装生产使用的完整 HTTP 中间件链：
// SCS LoadAndSave -> CSRF -> Gin Engine。
// 外部测试必须直接测试此方法返回的 Handler，保证测试与生产完全同一构造。
func (s *Server) BuildHandler() http.Handler {
	s.Setup()
	var handler http.Handler = s.engine
	handler = middleware.CSRF(s.cookieSecure)(handler)
	if s.sessionMgr != nil {
		// 必须包在最外层：SCS 才能拦截 gin 的 WriteHeader，登录 303 才会带上 Set-Cookie。
		handler = s.sessionMgr.LoadAndSave(handler)
	}
	return handler
}

// Run 启动 HTTP 服务，并在收到 SIGINT/SIGTERM 时优雅关闭（最长等待 30 秒）。
func (s *Server) Run() error {
	s.httpServer.Handler = s.BuildHandler()

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
