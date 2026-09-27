// =============================================================================
// 文件: internal/module/po/handler_personal.go
// 模块: PO 工作台
// 职责: 我的待办 / 通知中心 HTTP 接口。
// =============================================================================

package po

import (
	"net/http"
	"strconv"
	"strings"
	"workbench/internal/pkg/errorx"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/constants"
	"workbench/internal/middleware"
	"workbench/internal/pkg/render"
)

// Todos 渲染"我的待办"页面。
func (h *Handler) Todos(c *gin.Context) {
	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_TODOS, gin.H{
		"Title":           "我的待办",
		"PageTitle":       "我的待办",
		"PageDescription": "个人责任事项 · 办理与跟进",
		"BaseUrl":         "/todos",
	})
}

// TodosItems 返回"我的待办"列表 JSON。
func (h *Handler) TodosItems(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	var req TodoListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数解析失败"})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": "参数校验失败",
			"errors":  errs,
		})
		return
	}
	resp, err := h.svc.TodoList(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("po todo list", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取待办列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"items":    resp.Items,
		"total":    resp.Total,
		"page":     resp.Page,
		"pageSize": resp.PageSize,
		"summary":  resp.Summary,
		"groups":   resp.Groups,
		"facets":   resp.Facets,
	})
}

// Notice 渲染"通知中心"页面。
func (h *Handler) Notice(c *gin.Context) {
	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_NOTICE, gin.H{
		"Title":           "通知中心",
		"PageTitle":       "通知中心",
		"PageDescription": "聚合与我有关的业务动态、待处理事项和风险提醒",
		"BaseUrl":         "/notice",
	})
}

// NoticeItems 返回"通知中心"列表 + quick view 计数 JSON。
func (h *Handler) NoticeItems(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	var req NoticeListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数解析失败"})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": "参数校验失败",
			"errors":  errs,
		})
		return
	}
	resp, err := h.svc.NoticeList(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("po notice list", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取通知列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"items":         resp.Items,
		"total":         resp.Total,
		"filteredTotal": resp.Filtered,
		"unread":        resp.Unread,
		"action":        resp.Action,
		"inform":        resp.Inform,
		"abnormal":      resp.Abnormal,
		"today":         resp.Today,
		"categories":    resp.Categories,
		"page":          resp.Page,
		"pageSize":      resp.PageSize,
	})
}

// NoticeMarkRead 标记单条通知已读。
func (h *Handler) NoticeMarkRead(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效的通知 ID"})
		return
	}
	if err := h.svc.NoticeMarkRead(c.Request.Context(), actor, id); err != nil {
		if bizErr, ok := errorx.IsBizError(err); ok {
			switch bizErr.Code {
			case errorx.ErrCodeNotFound:
				c.JSON(http.StatusNotFound, gin.H{"message": bizErr.Msg})
				return
			case errorx.ErrCodeForbidden:
				c.JSON(http.StatusForbidden, gin.H{"message": bizErr.Msg})
				return
			}
		}
		h.logger.Error("po notice mark read", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "标记已读失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已标记已读", "redirectUrl": "/notice"})
}

// NoticeMarkAllRead 全部标记已读。
func (h *Handler) NoticeMarkAllRead(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	var body struct {
		Filters *NoticeListReq `json:"filters"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Filters == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "必须提供当前筛选条件"})
		return
	}
	req := *body.Filters
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "筛选条件无效", "errors": errs})
		return
	}
	n, err := h.svc.NoticeMarkAllRead(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("po notice mark all read", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "全部标记已读失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "全部已读", "redirectUrl": "/notice", "affected": n})
}
