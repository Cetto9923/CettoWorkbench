// =============================================================================
// 文件: internal/module/schedule/handler_demand_read.go
// 模块: 排期工作台
// 类型: action
// 职责: 业需排期弹窗详情与产品/项目联动查询的 JSON GET 接口。
// 依赖: internal/middleware
//
// =============================================================================

package schedule

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
)

// GetDemandScheduling 返回排期一体化弹窗业需详情（JSON）。
func (h *Handler) GetDemandScheduling(c *gin.Context) {
	demandID, ok := parseDemandID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "业需 ID 无效",
		})
		return
	}

	actor := middleware.CurrentUser(c)
	resp, err := h.svc.GetDemandScheduling(c.Request.Context(), actor, demandID)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("get demand scheduling detail failed",
				zap.Error(err),
				zap.Uint("demand_id", demandID),
			)
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

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
	if resp != nil {
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
		if resp.UserStories != nil {
			out["userStories"] = resp.UserStories
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
	}
	out["zentaoUrl"] = h.zentaoURL
	c.JSON(http.StatusOK, out)
}

// GetProjectExecutions 返回项目下的执行列表（JSON）。
func (h *Handler) GetProjectExecutions(c *gin.Context) {
	projectID, ok := parseProjectID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "项目 ID 无效",
		})
		return
	}

	actor := middleware.CurrentUser(c)
	executions, err := h.svc.GetProjectExecutions(c.Request.Context(), actor, projectID)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("get project executions failed",
				zap.Error(err),
				zap.Uint("project_id", projectID),
			)
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "加载执行列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"executions": executions,
	})
}

// GetProductProjects 返回产品关联的项目列表（JSON）。
func (h *Handler) GetProductProjects(c *gin.Context) {
	productID, ok := parseProductID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "产品 ID 无效",
		})
		return
	}

	actor := middleware.CurrentUser(c)
	projects, err := h.svc.GetProductProjects(c.Request.Context(), actor, productID)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("get product projects failed",
				zap.Error(err),
				zap.Uint("product_id", productID),
			)
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "加载项目列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"projects": projects,
	})
}

func parseProductID(c *gin.Context) (uint, bool) {
	idStr := strings.TrimSpace(c.Param("id"))
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}

func parseProjectID(c *gin.Context) (uint, bool) {
	return parseProductID(c)
}
