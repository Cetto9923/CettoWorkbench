// =============================================================================
// 文件: internal/module/schedule/handler_window_page.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期工作台首页 Index 渲染 + 仅 Index 使用的 helper(parseSchedulePage/bindRenderer)。
// 依赖: internal/constants
//       internal/middleware
//       internal/pkg/render
// =============================================================================

package schedule

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/render"
)

// Index 渲染排期工作台页面。
func (h *Handler) Index(c *gin.Context) {
	h.bindRenderer(c)
	actor := middleware.CurrentUser(c)

	bizPage := parseSchedulePage(c.DefaultQuery("bizPage", "1"))
	indepPage := parseSchedulePage(c.DefaultQuery("indepPage", "1"))

	demandData, ok := h.loadScheduleIndexDemandData(c, actor, bizPage, indepPage)
	if !ok {
		return
	}

	formData, err := h.svc.GetCreateWindowFormData(c.Request.Context(), actor)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load create window form data failed", zap.Error(err))
		}
		formData = &CreateWindowFormData{
			Teamgroups: []TeamgroupOption{},
			Products:   []ZtProduct{},
		}
	}

	windows, err := h.svc.ListWindowCards(c.Request.Context(), actor)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load version windows failed", zap.Error(err))
		}
		windows = []WindowCard{}
	}

	filterProducts, err := h.svc.ListFilterProducts(c.Request.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load filter products failed", zap.Error(err))
		}
		filterProducts = []ZtProduct{}
	}

	filterWindows, err := h.svc.ListFilterWindows(c.Request.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load filter windows failed", zap.Error(err))
		}
		filterWindows = []WindowFilterOption{}
	}

	filterUsers, err := h.svc.ListScheduleUsers(c.Request.Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load filter users failed", zap.Error(err))
		}
		filterUsers = []SchedulingUserOption{}
	}

	render.Page(c, http.StatusOK, constants.TEMPLATE_SCHEDULE_INDEX, gin.H{
		"Title":                   "排期工作台",
		"PageTitle":               "排期工作台",
		"Windows":                 windows,
		"BizRequirements":         demandData.BizRequirements,
		"BizTotal":                demandData.BizTotal,
		"IndependentRequirements": demandData.IndependentRequirements,
		"IndependentTotal":        demandData.IndependentTotal,
		"Teamgroups":              formData.Teamgroups,
		"Products":                formData.Products,
		"FilterProducts":          filterProducts,
		"FilterWindows":           filterWindows,
		"FilterUsers":             filterUsers,
		"StageFilterOptions":      ScheduleStageFilterOptionsForTab(demandData.ActiveTab),
		"WindowTypeFilterOptions": ScheduleWindowTypeFilterOptions,
		"SelectedGroups":          demandData.SelectedGroups,
		"SelectedProducts":        demandData.SelectedProducts,
		"SelectedStages":          demandData.SelectedStages,
		"SelectedWindows":         demandData.SelectedWindows,
		"SelectedKeyword":         demandData.SelectedKeyword,
		"SelectedPri":             demandData.SelectedPri,
		"SelectedWindowType":      demandData.SelectedWindowType,
		"SelectedDevOwner":        demandData.SelectedDevOwner,
		"SelectedTestOwner":       demandData.SelectedTestOwner,
		"SelectedAcceptOwner":     demandData.SelectedAcceptOwner,
		"SelectedGroupMap":        demandData.SelectedGroupMap,
		"SelectedProductMap":      demandData.SelectedProductMap,
		"SelectedStageMap":        demandData.SelectedStageMap,
		"SelectedWindowMap":       demandData.SelectedWindowMap,
		"BizPager":                demandData.BizPager,
		"IndepPager":              demandData.IndepPager,
		"ActiveFilter":            demandData.ActiveFilter,
		"SuspendedActive":         demandData.SuspendedActive,
		"SuspendedCount":          demandData.SuspendedCount,
		"BizFilterCounts":         demandData.BizFilterCounts,
		"IndepFilterCounts":       demandData.IndepFilterCounts,
	})
}

// parseSchedulePage 解析分页查询参数,无效时回落到 1。仅 Index 使用。
func parseSchedulePage(raw string) int {
	page, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

// bindRenderer 把渲染器放进 gin.Context,供模板渲染使用。仅 Index 使用。
func (h *Handler) bindRenderer(c *gin.Context) {
	if h.renderer != nil {
		c.Set("renderer", h.renderer)
	}
}
