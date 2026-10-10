// =============================================================================
// 文件: internal/module/teamleader/handler.go
// 模块: 团队长工作台
// 类型: handler
// 职责: 团队长工作台 HTTP 控制器与路由注册。
// 依赖: github.com/gin-gonic/gin
//       go.uber.org/zap
//       workbench/internal/middleware
//       workbench/internal/pkg/perm
// =============================================================================

package teamleader

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/perm"
)

// Handler 团队长工作台控制器。
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler 创建 Handler 实例。
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes 注册团队长工作台相关路由。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	// 挂载至 /team 路由组，绑定既有看板与工作台权限，并在服务层实施严格的对象级与数据范围鉴权
	g := rg.Group("/team")
	g.Use(middleware.RequireAnyPerm(perm.KanbanStory, perm.AgileTeamList, perm.PoHomeList))
	{
		g.GET("/hierarchy", h.GetHierarchy)
	}
}

// GetHierarchy 获取团队、小组、成员层级数据。
func (h *Handler) GetHierarchy(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	var req TeamHierarchyReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数解析失败"})
		return
	}

	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "message": "参数校验失败", "errors": errs})
		return
	}

	resp, err := h.svc.GetTeamHierarchy(c.Request.Context(), actor, req)
	if err != nil {
		var bizErr *errorx.BizError
		if errors.As(err, &bizErr) {
			if bizErr.Code == errorx.ErrCodeForbidden {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "message": bizErr.Msg})
				return
			}
			if bizErr.Code == errorx.ErrCodeNotFound {
				c.JSON(http.StatusNotFound, gin.H{"success": false, "message": bizErr.Msg})
				return
			}
		}
		if h.logger != nil {
			h.logger.Error("team hierarchy error", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取团队层级数据失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    resp,
	})
}
