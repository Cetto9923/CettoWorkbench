// =============================================================================
// 文件: internal/module/schedule/handler.go
// 模块: 排期工作台
// 类型: action
// 职责: Handler 主入口 + RegisterRoutes + 跨调用方 helper(parseWindowID/formatFieldErrors)。
// 依赖: internal/constants
//       internal/middleware
//       internal/pkg/render
// =============================================================================

package schedule

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
	"workbench/internal/pkg/render"
)

const scheduleRedirectURL = "/schedule"
const scheduleListPageSize = 10

// WindowCard 版本窗口概览卡片展示数据。
type WindowCard struct {
	ID               uint64
	ShortName        string
	Range            string
	ToneClass        string
	AgileGroup       string
	DemandCount      int
	CapacityHours    int
	UsedHours        int
	RemainingHours   int
	BlockedCount     int
	UsedPercent      int
	CanEdit          bool
	CanDelete        bool
	HasLinkedDemands bool
}

// WindowListItem 版本窗口维护列表项（JSON）。
type windowListItemJSON struct {
	ID               uint64 `json:"id"`
	Name             string `json:"name"`
	ReleaseDate      string `json:"releaseDate"`
	Range            string `json:"range"`
	DemandCount      int    `json:"demandCount"`
	CapacityHours    int    `json:"capacityHours"`
	UsedHours        int    `json:"usedHours"`
	RemainingHours   int    `json:"remainingHours"`
	BlockedCount     int    `json:"blockedCount"`
	UsedPercent      int    `json:"usedPercent"`
	CanEdit          bool   `json:"canEdit"`
	CanDelete        bool   `json:"canDelete"`
	HasLinkedDemands bool   `json:"hasLinkedDemands"`
}

func toWindowListItemJSON(item WindowListItem) windowListItemJSON {
	return windowListItemJSON{
		ID:               item.ID,
		Name:             item.Name,
		ReleaseDate:      item.ReleaseDate,
		Range:            item.Range,
		DemandCount:      item.DemandCount,
		CapacityHours:    item.CapacityHours,
		UsedHours:        item.UsedHours,
		RemainingHours:   item.RemainingHours,
		BlockedCount:     item.BlockedCount,
		UsedPercent:      item.UsedPercent,
		CanEdit:          item.CanEdit,
		CanDelete:        item.CanDelete,
		HasLinkedDemands: item.HasLinkedDemands,
	}
}

// Handler 处理排期工作台页面请求。
type Handler struct {
	renderer  *render.Renderer
	logger    *zap.Logger
	svc       *Service
	zentaoURL string
}

// NewHandler 创建排期模块 Handler。
func NewHandler(renderer *render.Renderer, logger *zap.Logger, svc *Service, zentaoURL string) *Handler {
	return &Handler{
		renderer:  renderer,
		logger:    logger,
		svc:       svc,
		zentaoURL: strings.TrimRight(strings.TrimSpace(zentaoURL), "/"),
	}
}

// RegisterRoutes 注册排期工作台路由（挂载在已配置登录与操作日志的中间件组上）。
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/schedule")
	g.Use(middleware.ActiveNav("/schedule"))
	{
		g.GET("", h.Index)
		g.GET("/matching-plans", h.GetMatchingPlans)
		g.GET("/demands/:id/scheduling", h.GetDemandScheduling)
		g.POST("/demands/:id/save-scheduling", h.SaveScheduling)
		g.GET("/stories/:id/scheduling", h.GetStoryScheduling)
		g.POST("/stories/:id/save-scheduling", h.SaveStoryScheduling)
		g.GET("/products/:id/projects", h.GetProductProjects)
		g.GET("/projects/:id/executions", h.GetProjectExecutions)
		g.GET("/stories/:id/tasks", h.GetStoryTasks)
		g.POST("/stories/:id/save-tasks", h.SaveStoryTasks)
		g.POST("/windows", h.CreateWindow)
		g.GET("/windows", h.ListWindows)
		g.GET("/windows/:id", h.GetWindow)
		g.PUT("/windows/:id", h.UpdateWindow)
		g.DELETE("/windows/:id", h.DeleteWindow)
	}
}

// parseWindowID 解析 URL 中的窗口 ID。供 GetWindow / UpdateWindow / DeleteWindow 共用。
func parseWindowID(c *gin.Context) (uint64, bool) {
	idStr := strings.TrimSpace(c.Param("id"))
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return id, true
}

// formatFieldErrors 把校验错误拼接成一行可读字符串。供 CreateWindow / UpdateWindow / DeleteWindow 共用。
func formatFieldErrors(errs []FieldError) string {
	if len(errs) == 0 {
		return "参数校验失败"
	}
	messages := make([]string, 0, len(errs))
	for _, item := range errs {
		msg := strings.TrimSpace(item.Message)
		if msg != "" {
			messages = append(messages, msg)
		}
	}
	if len(messages) == 0 {
		return "参数校验失败"
	}
	return strings.Join(messages, "；")
}
