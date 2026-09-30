// =============================================================================
// 文件: internal/module/po/handler_version_follow.go
// 模块: PO 工作台
// 类型: handler
// 职责: 版本跟进 HTTP 控制器方法。
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

// VersionFollow 渲染"版本跟进"页面。
func (h *Handler) VersionFollow(c *gin.Context) {
	render.Page(c, http.StatusOK, constants.TEMPLATE_PO_VERSION_FOLLOW, gin.H{
		"Title":           "版本跟进",
		"PageTitle":       "版本跟进",
		"PageDescription": "按版本窗口判断上线风险，逐条需求给出唯一下一步",
		"BaseUrl":         "/version-follow",
	})
}

// VersionFollowItems 返回"版本跟进"列表 JSON。
func (h *Handler) VersionFollowItems(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	var req VersionFollowListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数解析失败"})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "参数校验失败", "errors": errs})
		return
	}
	resp, err := h.svc.VersionFollowList(c.Request.Context(), actor, req)
	if err != nil {
		h.logger.Error("po version follow list", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取版本跟进数据失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

// VersionFollowOrphanEvidence 返回「已到测试后阶段但未挂版本窗口」的依据清单。
// 仅 AI 试点开启时有内容；关闭时返回空数组，前端据此不渲染 AI 块。
func (h *Handler) VersionFollowOrphanEvidence(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	if !aiPilotEnabled {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": []VersionFollowAIFinding{}})
		return
	}
	findings, err := h.svc.VersionFollowOrphanEvidence(c.Request.Context(), actor)
	if err != nil {
		h.logger.Error("po version follow orphan evidence", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取依据失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": findings})
}
