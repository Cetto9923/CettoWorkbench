// =============================================================================
// 文件: internal/module/po/handler_follow.go
// 模块: PO 工作台
// 类型: handler
// 职责: 我的关注 HTTP 控制器方法。
// =============================================================================

package po

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

// Follow 渲染"我的关注"页面。
func (h *Handler) Follow(c *gin.Context) {
	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_FOLLOW, gin.H{
		"Title":     "我的关注",
		"PageTitle": "我的关注",
		"BaseUrl":   "/follow",
	})
}

// FollowItems 返回"我的关注"列表 JSON。
func (h *Handler) FollowItems(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	var req FollowListReq
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
	resp, err := h.svc.FollowList(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("po follow list", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取关注列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"items":    resp.Items,
		"total":    resp.Total,
		"page":     resp.Page,
		"pageSize": resp.PageSize,
	})
}

// FollowSetDemand 切换对业务需求的关注状态。
func (h *Handler) FollowSetDemand(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效的业务需求 ID"})
		return
	}
	req := FollowSetReq{ID: id}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数解析失败"})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "参数校验失败", "errors": errs})
		return
	}
	if err := h.svc.FollowSetDemand(c.Request.Context(), actor, req); err != nil {
		h.logger.Error("po follow set", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "更新关注失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已更新关注", "redirectUrl": "/follow"})
}

// FollowRemoveProjectReport 解除当前用户对项目周报的关注。
func (h *Handler) FollowRemoveProjectReport(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效的项目 ID"})
		return
	}
	if err := h.svc.FollowRemoveProjectReport(c.Request.Context(), actor, id); err != nil {
		h.logger.Error("po project report unfollow", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "取消项目周报关注失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已取消项目周报关注", "redirectUrl": "/follow"})
}
