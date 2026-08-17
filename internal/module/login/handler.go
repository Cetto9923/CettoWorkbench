// =============================================================================
// 文件: internal/module/login/handler.go
// 模块: 登录
// 类型: action
// 职责: 处理登录页展示、登录提交与登出请求。
// 依赖: internal/pkg/errorx
//       internal/pkg/flash
//       internal/pkg/render
//       internal/pkg/perm
//       internal/pkg/ratelimit
//       internal/middleware
// =============================================================================

package login

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/flash"
	"workbench/internal/pkg/perm"
	ratelimitpkg "workbench/internal/pkg/ratelimit"
	"workbench/internal/pkg/render"
)

// Handler 处理认证 HTTP 请求。
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler 创建 Handler。
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger,
	}
}

// RegisterRoutes 注册 auth 模块路由。
func (h *Handler) RegisterRoutes(
	r *gin.Engine,
	requireLogin gin.HandlerFunc,
	redirectIfLoggedIn gin.HandlerFunc,
	loginLimiter *ratelimitpkg.Limiter,
) {
	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/login")
	})

	r.GET("/login", redirectIfLoggedIn, h.ShowLoginPage)
	r.POST("/login",
		middleware.RateLimit(loginLimiter, middleware.KeyByIP),
		h.DoLogin,
	)

	authed := r.Group("")
	authed.Use(requireLogin)
	{
		authed.POST("/logout",
			middleware.RequirePerm(perm.AuthLogout),
			h.DoLogout,
		)
	}
}

func (h *Handler) renderLoginPage(c *gin.Context, status int, redirectTo, account string, errors []FieldError) {
	if wantsJSON(c) {
		c.JSON(status, gin.H{
			"success": false,
			"errors":  jsonFieldErrors(errors),
		})
		return
	}
	// html/template 中 {{ if .Errors }} 对 nil 和空切片行为不同（nil 为 false，空切片为 true）。
	// 统一转为空切片，保证模板逻辑一致，避免“无错误时表单错误区域意外显示”的渲染 Bug。
	if errors == nil {
		errors = []FieldError{}
	}
	render.Page(c, status, constants.TEMPLATE_AUTH_LOGIN, gin.H{
		"Title":      "登录",
		"RedirectTo": redirectTo,
		"Form":       &LoginForm{Account: account},
		"Errors":     errors,
	})
}

// ShowLoginPage 显示登录页。
func (h *Handler) ShowLoginPage(c *gin.Context) {
	var req ShowLoginReq
	if err := c.ShouldBindQuery(&req); err != nil {
		h.renderLoginPage(c, http.StatusUnprocessableEntity, "", "", []FieldError{
			{Field: "_form", Message: "请求参数解析失败"},
		})
		return
	}
	redirectTo := strings.TrimSpace(req.Redirect)
	h.renderLoginPage(c, http.StatusOK, redirectTo, "", nil)
}

// DoLogin 执行登录。
func (h *Handler) DoLogin(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBind(&req); err != nil {
		h.logger.Warn("login bind failed",
			zap.String("ip", c.ClientIP()),
			zap.String("userAgent", c.GetHeader("User-Agent")),
			zap.Error(err),
		)
		h.renderLoginPage(c, http.StatusUnprocessableEntity, "", "", []FieldError{
			{Field: "_form", Message: "请求参数解析失败"},
		})
		return
	}
	req.Redirect = strings.TrimSpace(req.Redirect)
	req.IP = c.ClientIP()
	req.UserAgent = c.GetHeader("User-Agent")
	if fieldErrs := req.Validate(); len(fieldErrs) > 0 {
		h.logger.Warn("login validate failed",
			zap.String("account", req.Account),
			zap.String("ip", req.IP),
			zap.Any("fieldErrors", fieldErrs),
		)
		h.renderLoginPage(c, http.StatusUnprocessableEntity, req.Redirect, req.Account, fieldErrs)
		return
	}

	_, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		errMsg := err.Error()
		errCode := ""
		if bizErr, ok := errorx.IsBizError(err); ok {
			errMsg = bizErr.Msg
			errCode = bizErr.Code
		}
		h.logger.Warn("login failed",
			zap.String("account", req.Account),
			zap.String("ip", req.IP),
			zap.String("code", errCode),
			zap.String("message", errMsg),
		)
		h.renderLoginPage(c, http.StatusUnprocessableEntity, req.Redirect, req.Account, []FieldError{
			{Field: "_form", Message: errMsg},
		})
		return
	}

	flash.Success(c, "登录成功")

	redirectTo := "/home"
	if isSafeInternalPath(req.Redirect) {
		redirectTo = req.Redirect
	}
	if wantsJSON(c) {
		c.JSON(http.StatusOK, gin.H{
			"success":     true,
			"message":     "登录成功",
			"redirectUrl": redirectTo,
		})
		return
	}
	render.Redirect(c, redirectTo)
}

// DoLogout 执行登出。
func (h *Handler) DoLogout(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	if err := h.svc.Logout(c.Request.Context(), actor); err != nil {
		h.logger.Error("logout failed", zap.Error(err))
		render.Error(c, http.StatusInternalServerError, "登出失败", err)
		return
	}

	flash.Success(c, "已登出")
	render.Redirect(c, "/login")
}

func isSafeInternalPath(path string) bool {
	if path == "" {
		return false
	}
	// 必须以 "/" 开头，确保是站内相对路径而非外部 URL。
	// 禁止 "//" 开头：防止 "//evil.com" 协议相对 URL 被浏览器解析为外部跳转（开放重定向漏洞）。
	// 局限性：不防御路径穿越（如 "/../etc/passwd"）；如需更严格校验，可结合 path.Clean 或白名单。
	return strings.HasPrefix(path, "/") && !strings.Contains(path, "//")
}

func wantsJSON(c *gin.Context) bool {
	accept := strings.ToLower(c.GetHeader("Accept"))
	requestedWith := strings.ToLower(c.GetHeader("X-Requested-With"))
	return strings.Contains(accept, "application/json") || requestedWith == "xmlhttprequest"
}

func jsonFieldErrors(errors []FieldError) []FieldError {
	if errors == nil {
		return []FieldError{}
	}
	return errors
}
