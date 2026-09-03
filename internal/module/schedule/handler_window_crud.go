// =============================================================================
// 文件: internal/module/schedule/handler_window_crud.go
// 模块: 排期工作台
// 类型: action
// 职责: 版本窗口维护 CRUD 5 个接口 + 匹配计划查询 GetMatchingPlans。
// 依赖: internal/middleware
// =============================================================================

package schedule

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
)

// ListWindows 返回版本窗口维护列表（JSON）。
func (h *Handler) ListWindows(c *gin.Context) {
	actor := middleware.CurrentUser(c)

	resp, err := h.svc.ListWindows(c.Request.Context(), actor)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("list version windows failed", zap.Error(err))
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "加载版本窗口列表失败",
		})
		return
	}

	windows := make([]windowListItemJSON, 0, len(resp.Windows))
	for _, item := range resp.Windows {
		windows = append(windows, toWindowListItemJSON(item))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"windows": windows,
	})
}

// CreateWindow 保存新建版本窗口（JSON）。
func (h *Handler) CreateWindow(c *gin.Context) {
	var req CreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "请求参数解析失败",
		})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"message": "参数校验失败",
			"errors":  errs,
			"error":   formatFieldErrors(errs),
		})
		return
	}

	actor := middleware.CurrentUser(c)
	if actorAccount(actor) == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "未登录或无法识别当前用户",
		})
		return
	}

	if err := h.svc.Create(c.Request.Context(), actor, req); err != nil {
		if h.logger != nil {
			h.logger.Error("save version window failed",
				zap.Error(err),
				zap.String("name", strings.TrimSpace(req.Name)),
			)
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "保存版本窗口失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "版本窗口保存成功",
		"redirectUrl": scheduleRedirectURL,
	})
}

// GetWindow 获取版本窗口详情（JSON）。
func (h *Handler) GetWindow(c *gin.Context) {
	id, ok := parseWindowID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "窗口 ID 无效",
		})
		return
	}

	actor := middleware.CurrentUser(c)
	detail, err := h.svc.GetByID(c.Request.Context(), actor, id)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("get version window detail failed", zap.Error(err), zap.Uint64("window_id", id))
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	resp := gin.H{"success": true}
	if detail != nil {
		resp["id"] = detail.ID
		resp["releaseDate"] = detail.ReleaseDate
		resp["name"] = detail.Name
		resp["startDate"] = detail.StartDate
		resp["teamgroupId"] = detail.TeamgroupID
		resp["groupSize"] = detail.GroupSize
		resp["products"] = detail.Products
	}
	c.JSON(http.StatusOK, resp)
}

// UpdateWindow 更新版本窗口（JSON）。
func (h *Handler) UpdateWindow(c *gin.Context) {
	id, ok := parseWindowID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "窗口 ID 无效",
		})
		return
	}

	var req UpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "请求参数解析失败",
		})
		return
	}
	req.ID = id
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"message": "参数校验失败",
			"errors":  errs,
			"error":   formatFieldErrors(errs),
		})
		return
	}

	actor := middleware.CurrentUser(c)
	if actorAccount(actor) == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "未登录或无法识别当前用户",
		})
		return
	}

	if err := h.svc.Update(c.Request.Context(), actor, req); err != nil {
		if h.logger != nil {
			h.logger.Error("update version window failed",
				zap.Error(err),
				zap.Uint64("window_id", id),
			)
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "更新版本窗口失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "版本窗口更新成功",
		"redirectUrl": scheduleRedirectURL,
	})
}

// DeleteWindow 软删除版本窗口（JSON）。
func (h *Handler) DeleteWindow(c *gin.Context) {
	id, ok := parseWindowID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "窗口 ID 无效",
		})
		return
	}

	actor := middleware.CurrentUser(c)
	if actorAccount(actor) == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "未登录或无法识别当前用户",
		})
		return
	}

	deleteReq := DeleteReq{ID: id}
	if errs := deleteReq.Validate(); len(errs) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   formatFieldErrors(errs),
		})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), actor, deleteReq); err != nil {
		if h.logger != nil {
			h.logger.Error("delete version window failed",
				zap.Error(err),
				zap.Uint64("window_id", id),
			)
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "版本窗口已删除",
		"redirectUrl": scheduleRedirectURL,
	})
}

// GetMatchingPlans 根据产品 ID 和结束日期返回匹配计划（JSON）。
func (h *Handler) GetMatchingPlans(c *gin.Context) {
	var req MatchingPlansReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "参数解析失败",
		})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": "参数校验失败",
			"errors":  errs,
		})
		return
	}

	actor := middleware.CurrentUser(c)
	resp, err := h.svc.GetMatchingPlans(c.Request.Context(), actor, req)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("get matching plans failed",
				zap.Error(err),
				zap.Uint("product_id", req.ProductID),
				zap.String("end_date", req.EndDate),
			)
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "查询匹配计划失败",
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}
