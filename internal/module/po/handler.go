// =============================================================================
// 文件: internal/module/po/handler.go
// 模块: PO 工作台
// 类型: action
// 职责: PO 工作台页面 HTTP 请求。
// 依赖: internal/middleware
//       internal/pkg/render
// =============================================================================

package po

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/render"
)

// Handler 处理 PO 工作台页面请求。
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler 创建 PO 模块 Handler。
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// RegisterRoutes 注册 PO 工作台路由（挂载在已配置登录与操作日志的中间件组上）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("")
	g.Use(middleware.ActiveNav("/po/home"))

	g.GET("/home", h.Home)
}

// Home 渲染 PO 工作台首页。
func (h *Handler) Home(c *gin.Context) {
	resp, err := h.svc.Home(c.Request.Context(), middleware.CurrentUser(c))
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("po home value stream", zap.Error(err))
		}
		resp = &HomeResp{Stages: emptyValueStreamStages()}
	}

	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_HOME, gin.H{
		"Title":             "工作台首页",
		"PageTitle":         "工作台首页",
		"ValueStreamStages": resp.Stages,
	})
}

func emptyValueStreamStages() []ValueStreamStage {
	stages := make([]ValueStreamStage, 0, len(valueStreamStages))
	for _, def := range valueStreamStages {
		stages = append(stages, ValueStreamStage{
			Label:  def.label,
			Status: def.status,
			IsAll:  def.isAll,
		})
	}
	return stages
}
