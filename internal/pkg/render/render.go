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
	"errors"
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

// Renderer 模板渲染器：dev 每次 ParseFiles，prod 启动时缓存 layout×page 组合。
type Renderer struct {
	templateDir string
	staticDir   string
	isDev       bool
	cache       map[string]*template.Template
	appName     string
	layoutNav   string
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
	if cfg != nil {
		layoutNav = cfg.Layout.Nav
	}
	r := &Renderer{
		templateDir: templateDir,
		staticDir:   staticDir,
		isDev:       isDev,
		cache:       make(map[string]*template.Template),
		appName:     appName,
		layoutNav:   layoutNav,
	}
	if isDev {
		return r, nil
	}
	if err := r.warmCache(); err != nil {
		return nil, err
	}
	return r, nil
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
		data["CSRFToken"] = nosurf.Token(c.Request)
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
		data["CurrentPath"] = c.Request.URL.Path
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

func dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}
	result := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		result[key] = values[i+1]
	}
	return result, nil
}

func (r *Renderer) asset(path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		return path
	}
	rel := strings.TrimPrefix(p, "/")
	rel = strings.TrimPrefix(rel, "static/")
	rel = filepath.FromSlash(rel)
	clean := filepath.Clean(rel)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return path
	}
	full := filepath.Join(r.staticDir, clean)
	base := filepath.Clean(r.staticDir)
	fullClean := filepath.Clean(full)
	relFromBase, err := filepath.Rel(base, fullClean)
	if err != nil || relFromBase == ".." || strings.HasPrefix(relFromBase, ".."+string(filepath.Separator)) {
		return path
	}
	st, err := os.Stat(full)
	if err != nil {
		return path
	}
	v := st.ModTime().Unix()
	if strings.HasPrefix(p, "/static/") {
		return fmt.Sprintf("%s?v=%d", p, v)
	}
	return fmt.Sprintf("/static/%s?v=%d", filepath.ToSlash(clean), v)
}

func add(a, b interface{}) int {
	return toInt(a) + toInt(b)
}

func sub(a, b interface{}) int {
	return toInt(a) - toInt(b)
}

func alertClass(level interface{}) string {
	s := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", level)))
	switch s {
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
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int8:
		return int(n)
	case int16:
		return int(n)
	case int32:
		return int(n)
	case int64:
		return int(n)
	case uint:
		return int(n)
	case uint8:
		return int(n)
	case uint16:
		return int(n)
	case uint32:
		return int(n)
	case uint64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	default:
		return 0
	}
}
