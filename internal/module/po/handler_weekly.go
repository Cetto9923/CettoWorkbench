// =============================================================================
// 文件: internal/module/po/handler_weekly.go
// 模块: PO 工作台
// 类型: action
// 职责: 项目周报聚合列表 / 详情 / 历史 / 承建团队 HTTP 处理
// 依赖: github.com/gin-gonic/gin
//       go.uber.org/zap
//       workbench/internal/middleware
// =============================================================================

package po

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
)

// ProjectWeeklies 返回项目周报聚合列表与同源统计（scope=watched|all）。
func (h *Handler) ProjectWeeklies(c *gin.Context) {
	var req ProjectWeeklyListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数解析失败"})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "errors": errs})
		return
	}
	resp, err := h.svc.ProjectWeeklies(c.Request.Context(), middleware.CurrentUser(c), req)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("project weeklies", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取项目周报列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

// ProjectWeeklyDetail 侧滑详情。
func (h *Handler) ProjectWeeklyDetail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "项目 ID 无效"})
		return
	}
	var req ProjectWeeklyItemReq
	_ = c.ShouldBindQuery(&req)
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "errors": errs})
		return
	}
	resp, err := h.svc.ProjectWeeklyDetail(c.Request.Context(), middleware.CurrentUser(c), uint(id), req.Scope)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("project weekly detail", zap.Error(err))
		}
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "未找到项目周报"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

// ProjectWeeklyHistory 历史周报。
func (h *Handler) ProjectWeeklyHistory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "项目 ID 无效"})
		return
	}
	var req ProjectWeeklyItemReq
	_ = c.ShouldBindQuery(&req)
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "errors": errs})
		return
	}
	items, err := h.svc.ProjectWeeklyHistory(c.Request.Context(), middleware.CurrentUser(c), uint(id), req.Scope)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("project weekly history", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取历史周报失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "items": items})
}
