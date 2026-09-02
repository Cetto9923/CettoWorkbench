// =============================================================================
// 文件: internal/module/user/handler.go
// 模块: 用户管理
// 类型: crud
// 职责: 用户模块 Handler 主入口：构造、路由注册及共用 ID 解析工具。
// 依赖: internal/middleware
//
//	internal/pkg/perm
//	internal/pkg/render
//
// =============================================================================

package user

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/render"
)

// Handler 处理用户管理页面请求。
type Handler struct {
	renderer *render.Renderer
	logger   *zap.Logger
	svc      *Service
}

// NewHandler 创建用户模块 Handler。
func NewHandler(renderer *render.Renderer, logger *zap.Logger, svc *Service) *Handler {
	return &Handler{
		renderer: renderer,
		logger:   logger,
		svc:      svc,
	}
}

// RegisterRoutes 注册用户模块路由。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	users.Use(middleware.ActiveNav("/admin/users"))
	users.GET("", middleware.RequirePerm(perm.UserList), h.List)
	users.GET("/new", middleware.RequirePerm(perm.UserCreate), h.NewForm)
	users.GET("/:id/edit", middleware.RequirePerm(perm.UserUpdate), h.EditForm)
	users.GET("/batch/create", middleware.RequirePerm(perm.UserCreate), h.BatchPage)

	users.POST("", middleware.RequirePerm(perm.UserCreate), h.Create)
	users.POST("/export", middleware.RequirePerm(perm.UserList), h.Export)
	users.POST("/batch/create", middleware.RequirePerm(perm.UserCreate), h.BatchCreate)
	users.POST("/:id", middleware.RequirePerm(perm.UserUpdate), h.Update)
	users.POST("/:id/toggle-status", middleware.RequirePerm(perm.UserUpdate), h.ToggleStatus)
	users.POST("/:id/reset-password", middleware.RequirePerm(perm.UserResetPassword), h.ResetPassword)

	users.DELETE("/:id", middleware.RequirePerm(perm.UserDelete), h.Delete)
}

// parseID 解析 :id 路径参数为 int64。
// 多处 handler 共用,集中在此以避免重复定义。
func parseID(raw string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
