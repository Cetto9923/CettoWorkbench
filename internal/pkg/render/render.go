// =============================================================================
// 文件: internal/pkg/render/render.go
// 模块: 基础设施
// 类型: infra
// 职责: 提供统一页面渲染、错误渲染与重定向能力。
// 依赖: internal/config
//       internal/constants
//       internal/model
//       internal/pkg/flash
//       internal/pkg/logger
//       internal/pkg/menu
// =============================================================================

package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/justinas/nosurf"
	"go.uber.org/zap"

	"workbench/internal/config"
	"workbench/internal/constants"
	"workbench/internal/model"
	"workbench/internal/pkg/flash"
	"workbench/internal/pkg/logger"
	"workbench/internal/pkg/menu"
)

const (
	layoutDir = "layout"
	html500   = `<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>服务器错误</title></head><body><p>服务器暂时无法处理请求，请稍后再试。</p></body></html>`
)

var (
	defaultRendererMu sync.RWMutex
	defaultRenderer   *Renderer
)

// SidebarBadgesProvider 与 SidebarBadges 类型定义见 sidebar_badges.go（独立文件以
// 保持本文件在职责分离下不超过 500 行上限）。

// Renderer 模板渲染器：dev 每次 ParseFiles，prod 启动时缓存 layout×page 组合。
type Renderer struct {
	templateDir       string
	staticDir         string
	isDev             bool
	cache             map[string]*template.Template
	appName           string
	layoutNav         string
	zentaoURL         string
	zentaoRequestType string
	sidebarBadges     SidebarBadgesProvider
}

// New 创建 Renderer。
func New(cfg *config.Config, isDev bool) (*Renderer, error) {
	templateDir := filepath.Clean("web/templates")
	staticDir := filepath.Join(filepath.Dir(templateDir), "static")
	appName := ""
	if cfg != nil && strings.TrimSpace(cfg.App.Name) != "" {
		appName = strings.TrimSpace(cfg.App.Name)
	}
	layoutNav := "sidebar"
	zentaoURL := ""
	zentaoRequestType := ""
	if cfg != nil {
		layoutNav = cfg.Layout.Nav
		zentaoURL = strings.TrimRight(strings.TrimSpace(cfg.Zentao.URL), "/")
		zentaoRequestType = strings.TrimSpace(cfg.Zentao.RequestType)
	}
	r := &Renderer{
		templateDir:       templateDir,
		staticDir:         staticDir,
		isDev:             isDev,
		cache:             make(map[string]*template.Template),
		appName:           appName,
		layoutNav:         layoutNav,
		zentaoURL:         zentaoURL,
		zentaoRequestType: zentaoRequestType,
	}
	if isDev {
		return r, nil
	}
	if err := r.warmCache(); err != nil {
		return nil, err
	}
	return r, nil
}

// SetSidebarBadgesProvider 注册侧栏角标数据源；不强制要求（缺省为 nil 表示不显示）。
func (r *Renderer) SetSidebarBadgesProvider(p SidebarBadgesProvider) {
	r.sidebarBadges = p
}

// SetDefault 注册默认渲染器实例。
func SetDefault(r *Renderer) {
	defaultRendererMu.Lock()
	defaultRenderer = r
	defaultRendererMu.Unlock()
}

// Page 渲染页面。
func Page(c *gin.Context, status int, page string, data gin.H) {
	r := rendererFromContext(c)
	if r == nil {
		c.String(http.StatusInternalServerError, "renderer not initialized")
		return
	}
	if err := r.renderPage(c, status, page, data); err != nil {
		r.failRender(c, err)
	}
}

// Error 渲染统一错误页。
func Error(c *gin.Context, status int, userMsg string, err error) {
	r := rendererFromContext(c)
	log := logger.FromContext(c.Request.Context())
	log.Error("request failed",
		zap.Error(err),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.Int64("userID", currentUserID(c)),
	)
	if r == nil {
		c.String(status, userMsg)
		return
	}

	detail := ""
	if r.isDev && err != nil {
		detail = err.Error()
	}
	data := gin.H{
		"Title":   "出错了",
		"Status":  status,
		"Message": userMsg,
		"Detail":  detail,
	}
	if renderErr := r.renderPage(c, status, "error", data); renderErr != nil {
		log.Error("render error page failed", zap.Error(renderErr))
		c.String(status, userMsg)
	}
}

// Redirect 使用 303 See Other 而非 302 Found，
// 确保浏览器在重定向后以 GET 方法请求目标，防止 POST 表单刷新时重复提交（PRG 模式）。
func Redirect(c *gin.Context, path string) {
	c.Redirect(http.StatusSeeOther, path)
}

func rendererFromContext(c *gin.Context) *Renderer {
	if v, ok := c.Get("renderer"); ok {
		if r, ok := v.(*Renderer); ok && r != nil {
			return r
		}
	}
	defaultRendererMu.RLock()
	r := defaultRenderer
	defaultRendererMu.RUnlock()
	return r
}

func (r *Renderer) warmCache() error {
	return filepath.WalkDir(r.templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".html") {
			return nil
		}
		rel, err := filepath.Rel(r.templateDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, layoutDir+"/") || strings.HasPrefix(rel, "components/") {
			return nil
		}
		page := strings.TrimSuffix(rel, ".html")
		tpl, err := r.parseTemplates(page)
		if err != nil {
			return fmt.Errorf("parse template %s: %w", page, err)
		}
		r.cache[page] = tpl
		return nil
	})
}

func resolveLayout(page string) string {
	if strings.HasPrefix(page, "auth/") {
		return "auth"
	}
	return "base"
}

func (r *Renderer) parseTemplates(page string) (*template.Template, error) {
	layout := resolveLayout(page)
	layoutFile := filepath.Join(r.templateDir, layoutDir, layout+".html")
	pageFile := filepath.Join(r.templateDir, filepath.FromSlash(page)+".html")
	files := []string{layoutFile, pageFile}
	layoutFiles, err := collectTemplateFiles(filepath.Join(r.templateDir, layoutDir))
	if err != nil {
		return nil, err
	}
	componentFiles, err := collectTemplateFiles(filepath.Join(r.templateDir, "components"))
	if err != nil {
		return nil, err
	}
	files = append(files, layoutFiles...)
	files = append(files, componentFiles...)
	files = uniquePaths(files)
	return template.New("").Funcs(r.funcMap()).ParseFiles(files...)
}

func collectTemplateFiles(root string) ([]string, error) {
	entries := make([]string, 0)
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return entries, nil
		}
		return nil, err
	}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".html") {
			entries = append(entries, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(entries)
	return entries, nil
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		cleanPath := filepath.Clean(path)
		if _, ok := seen[cleanPath]; ok {
			continue
		}
		seen[cleanPath] = struct{}{}
		result = append(result, cleanPath)
	}
	return result
}

func (r *Renderer) renderPage(c *gin.Context, status int, page string, data gin.H) error {
	if data == nil {
		data = gin.H{}
	}
	r.enrichData(c, page, data)

	var (
		tpl *template.Template
		err error
	)
	if r.isDev {
		tpl, err = r.parseTemplates(page)
	} else {
		tpl = r.cache[page]
		if tpl == nil {
			tpl, err = r.parseTemplates(page)
		}
	}
	if err != nil {
		return err
	}
	layoutName := resolveLayout(page) + ".html"
	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, layoutName, data); err != nil {
		return err
	}
	c.Status(status)
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(c.Writer)
	return nil
}

func (r *Renderer) failRender(c *gin.Context, err error) {
	log := logger.FromContext(c.Request.Context())
	log.Error("template render failed", zap.Error(err))
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Writer.WriteHeader(http.StatusInternalServerError)
	_, _ = c.Writer.WriteString(html500)
}

func currentUserID(c *gin.Context) int64 {
	if v, ok := c.Get("currentUser"); ok {
		if u, ok := v.(*model.User); ok && u != nil {
			return u.ID
		}
	}
	return 0
}

func (r *Renderer) enrichData(c *gin.Context, page string, data gin.H) {
	if _, ok := data["CurrentUser"]; !ok {
		if v, exists := c.Get("currentUser"); exists {
			data["CurrentUser"] = v
		} else {
			data["CurrentUser"] = nil
		}
	}
	if _, ok := data["CSRFToken"]; !ok {
		if c.Request != nil {
			data["CSRFToken"] = nosurf.Token(c.Request)
		} else {
			data["CSRFToken"] = ""
		}
	}
	if _, ok := data["Menus"]; !ok {
		if v, exists := c.Get("currentMenus"); exists {
			data["Menus"] = v
		} else {
			data["Menus"] = []menu.Menu{}
		}
	}
	if _, ok := data["CurrentMenus"]; !ok {
		if v, exists := data["Menus"]; exists {
			data["CurrentMenus"] = v
		} else {
			data["CurrentMenus"] = []menu.Menu{}
		}
	}
	if _, ok := data["AppName"]; !ok {
		data["AppName"] = r.appName
	}
	if _, ok := data["HideChrome"]; !ok {
		data["HideChrome"] = constants.PageHidesChrome(page)
	}
	if _, ok := data["LayoutNav"]; !ok {
		data["LayoutNav"] = r.layoutNav
	}
	if _, ok := data["CurrentPath"]; !ok {
		if c.Request != nil && c.Request.URL != nil {
			data["CurrentPath"] = c.Request.URL.Path
		} else {
			data["CurrentPath"] = ""
		}
	}
	if _, ok := data["ActiveNavKey"]; !ok {
		if v, exists := c.Get(menu.ContextActiveNavKey); exists {
			if s, ok := v.(string); ok {
				data["ActiveNavKey"] = s
			} else {
				data["ActiveNavKey"] = ""
			}
		} else {
			data["ActiveNavKey"] = ""
		}
	}
	if _, ok := data["ZentaoURL"]; !ok {
		data["ZentaoURL"] = r.zentaoURL
	}
	if _, ok := data["ZentaoRequestType"]; !ok {
		data["ZentaoRequestType"] = r.zentaoRequestType
	}
	if _, ok := data["Flash"]; !ok {
		data["Flash"] = flash.Pop(c)
	}
	if _, ok := data["FlashMessages"]; !ok {
		msg := data["Flash"]
		if m, ok := msg.(*flash.Message); ok && m != nil {
			data["FlashMessages"] = []flash.Message{*m}
		} else {
			data["FlashMessages"] = []flash.Message{}
		}
	}
	// 侧栏角标：每个页面渲染时拉一次，失败或未登录返零值。
	if _, ok := data["SidebarBadges"]; !ok {
		badges := SidebarBadges{}
		if r.sidebarBadges != nil {
			if v, exists := c.Get("currentUser"); exists {
				if u, ok := v.(*model.User); ok && u != nil {
					if b, err := r.sidebarBadges(c); err == nil {
						badges = b
					}
				}
			}
		}
		data["SidebarBadges"] = badges
	}
}

func (r *Renderer) funcMap() template.FuncMap {
	return template.FuncMap{
		"asset":         r.asset,
		"add":           add,
		"sub":           sub,
		"alertclass":    alertClass,
		"dict":          dict,
		"menuNavActive": menu.MenuNavActive,
	}
}

// asset / dict / add / sub / alertClass / toInt 的实现见 helpers.go（独立文件
// 以保持 render.go 在职责分离下不超过 500 行上限）。
