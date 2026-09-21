package po

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"workbench/internal/middleware"
	"workbench/internal/pkg/render"
)

func (h *Handler) IssueRisk(c *gin.Context) {
	render.Page(c, http.StatusOK, "po/issue-risk", gin.H{
		"Title":           "问题风险",
		"PageTitle":       "问题风险",
		"PageDescription": "统一跟踪和治理跨需求、跨团队的问题与风险项",
	})
}
func (h *Handler) IssueRiskItems(c *gin.Context) {
	var req IssueRiskListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数解析失败"})
		return
	}
	if es := req.Validate(); len(es) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "参数校验失败", "errors": es})
		return
	}
	resp, err := h.svc.IssueRiskList(c.Request.Context(), middleware.CurrentUser(c), req)
	if err != nil {
		h.logger.Error("po issue risk list", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取问题风险失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "items": resp.Items, "total": resp.Total, "kindCounts": resp.KindCounts, "loopCounts": resp.LoopCounts, "overdueCount": resp.OverdueCount, "projects": resp.Projects, "page": req.Page, "pageSize": req.PageSize})
}
