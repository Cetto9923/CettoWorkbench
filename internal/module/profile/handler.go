// =============================================================================
// 文件: internal/module/profile/handler.go
// 模块: 个人资料
// 类型: action
// 职责: 个人资料 HTTP 入口：GET /profile 渲染模板，GET /profile/data 返回 JSON，
//       PUT /profile 与 PUT /profile/password 为写接口。
// 依赖: internal/middleware
//       internal/pkg/errorx
//       internal/pkg/perm
//       internal/pkg/render
// =============================================================================

package profile

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

// Handler 个人资料 HTTP 入口。
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler 创建 Handler。
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes 注册 /profile 与 API 路由。
//
// 权限：沿用现有 PoHomeList（个人资料任何已登录用户可访问，不新增 perm）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/profile")
	g.GET("", middleware.RequirePerm(perm.PoHomeList), h.Index)
	g.GET("/data", middleware.RequirePerm(perm.PoHomeList), h.GetData)
	g.PUT("", middleware.RequirePerm(perm.PoHomeList), h.Update)
	g.PUT("/password", middleware.RequirePerm(perm.PoHomeList), h.ChangePassword)
}

// Index 渲染个人资料页模板。
//
// Service 失败时仍渲染模板（空数据 + 页面错误提示），避免 500。
func (h *Handler) Index(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	resp, pageErr := "", ""
	if actor != nil && actor.ID > 0 {
		if _, err := h.svc.Get(c.Request.Context(), actor); err != nil {
			if h.logger != nil {
				h.logger.Warn("profile index get failed", zap.Error(err))
			}
			pageErr = "个人资料暂不可用"
		} else {
			// Index 不需要把数据塞进模板；前端 JS 走 /profile/data 异步加载。
			_ = resp
		}
	}
	render.Page(c, http.StatusOK, constants.TEMPLATE_PROFILE_INDEX, gin.H{
		"Title":     "个人资料",
		"PageTitle": "个人资料",
		"BaseUrl":   "/profile",
		"PageError": pageErr,
	})
}

// GetData 返回当前用户资料 JSON。
func (h *Handler) GetData(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	if actor == nil || actor.ID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "请先登录",
		})
		return
	}
	resp, err := h.svc.Get(c.Request.Context(), actor)
	if err != nil {
		writeProfileErr(c, h.logger, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

// Update 保存当前用户资料。
func (h *Handler) Update(c *gin.Context) {
	var req UpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求格式错误",
		})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"errors":  errs,
		})
		return
	}
	actor := middleware.CurrentUser(c)
	if err := h.svc.Update(c.Request.Context(), actor, req); err != nil {
		writeProfileErr(c, h.logger, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "个人资料已保存",
	})
}

// ChangePassword 修改当前用户密码。
func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求格式错误",
		})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"errors":  errs,
		})
		return
	}
	actor := middleware.CurrentUser(c)
	if err := h.svc.ChangePassword(c.Request.Context(), actor, req); err != nil {
		writeProfileErr(c, h.logger, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "密码已更新",
	})
}

// writeProfileErr 把 service 错误映射为合适的 HTTP 状态码。
func writeProfileErr(c *gin.Context, logger *zap.Logger, err error) {
	if biz, ok := errorx.IsBizError(err); ok {
		status := http.StatusBadRequest
		switch biz.Code {
		case "unauthorized":
			status = http.StatusUnauthorized
		case "not_found":
			status = http.StatusNotFound
		case "invalid_team", "bad_password":
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"success": false, "message": biz.Msg})
		return
	}
	if logger != nil {
		logger.Error("profile request failed", zap.Error(err))
	}
	c.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"message": "操作失败，请稍后重试",
	})
}
