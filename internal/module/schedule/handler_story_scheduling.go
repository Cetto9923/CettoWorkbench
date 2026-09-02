// =============================================================================
// 文件: internal/module/schedule/handler_story_scheduling.go
// 模块: 排期工作台
// 类型: action
// 职责: 处理独立研发需求排期弹窗 HTTP 请求。
// 依赖: internal/module/schedule/service_independent.go
//       internal/middleware
// =============================================================================

package schedule

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
)

// GetStoryScheduling 返回独立研发需求排期弹窗详情（JSON）。
func (h *Handler) GetStoryScheduling(c *gin.Context) {
	storyID, ok := parseStoryID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "研发需求 ID 无效",
		})
		return
	}

	actor := middleware.CurrentUser(c)
	resp, err := h.svc.GetStoryScheduling(c.Request.Context(), actor, storyID)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("get story scheduling detail failed",
				zap.Error(err),
				zap.Uint("story_id", storyID),
			)
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	out := buildSchedulingDetailJSON(resp)
	out["zentaoUrl"] = h.zentaoURL
	c.JSON(http.StatusOK, out)
}

func buildSchedulingDetailJSON(resp *DemandSchedulingResp) gin.H {
	out := gin.H{
		"success":           true,
		"involvedProducts":  []ZtProductOption{},
		"productProjects":   gin.H{},
		"projectExecutions": gin.H{},
		"stories":           []DemandSchedulingStoryItem{},
		"userStories":       []UserStoryItem{},
		"windows":           []SchedulingWindowOption{},
		"users":             []SchedulingUserOption{},
	}
	if resp == nil {
		return out
	}
	if resp.InvolvedProducts != nil {
		out["involvedProducts"] = resp.InvolvedProducts
	}
	if resp.ProductProjects != nil {
		out["productProjects"] = resp.ProductProjects
	}
	if resp.ProjectExecutions != nil {
		out["projectExecutions"] = resp.ProjectExecutions
	}
	if resp.Stories != nil {
		out["stories"] = resp.Stories
	}
	if resp.Windows != nil {
		out["windows"] = resp.Windows
	}
	if resp.Users != nil {
		out["users"] = resp.Users
	}
	if resp.DemandSchedulingDetail != nil {
		detail := resp.DemandSchedulingDetail
		out["id"] = detail.ID
		out["name"] = detail.Name
		out["pri"] = detail.Pri
		out["bra"] = detail.BRA
		out["braName"] = detail.BRAName
		out["rd"] = detail.RD
		out["rdName"] = detail.RDName
		out["qd"] = detail.QD
		out["qdName"] = detail.QDName
		out["accepter"] = detail.Accepter
		out["accepterName"] = detail.AccepterName
		out["mainSystemId"] = detail.MainSystemID
		out["mainSystemName"] = detail.MainSystemName
		out["schedulePlanDate"] = detail.SchedulePlanDate
		out["developFinish"] = detail.DevelopFinish
		out["testFinish"] = detail.TestFinish
		out["acceptancedDate"] = detail.AcceptancedDate
		out["windowId"] = detail.WindowID
		out["windowName"] = detail.WindowName
		out["windowPhase"] = detail.WindowPhase
		out["canEditWindow"] = detail.CanEditWindow
	}
	return out
}
