// =============================================================================
// 文件: internal/module/schedule/handler_demand.go
// 模块: 排期工作台
// 类型: action
// 职责: 业需与独立研发需求列表页面数据加载及视图模型。
// 依赖: internal/middleware
//       internal/pkg/pagination
//       internal/pkg/render
// =============================================================================

package schedule

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/middleware"
	"workbench/internal/model"
	"workbench/internal/pkg/pagination"
	"workbench/internal/pkg/render"
)

// IndependentChildRequirement 独立研发需求子行（树形二级）。
type IndependentChildRequirement struct {
	ID            string
	Title         string
	Priority      string
	PriClass      string
	ProductName   string
	Stage         string
	StageClass    string
	WindowName    string
	TeamgroupName string
	Owner         string
	TaskCount     int
	DetailURL     template.URL
}

// IndependentRequirement 独立研发需求行（树形一级）。
type IndependentRequirement struct {
	ID            string
	Title         string
	Priority      string
	PriClass      string
	ProductName   string
	Stage         string
	StageClass    string
	WindowName    string
	TeamgroupName string
	Owner         string
	TaskCount     int
	HasChildren   bool
	DetailURL     template.URL
	Children      []IndependentChildRequirement
}

// DevRequirement 研发需求行（树形三级）。
type DevRequirement struct {
	ID          string
	Title       string
	Priority    string
	PriClass    string
	IsMain      bool
	Owner       string
	TaskCount   int
	ActionLabel string
	ActionClass string
	DetailURL   template.URL
}

// SubBizRequirement 子业务需求行（树形二级）。
type SubBizRequirement struct {
	DemandID        uint
	ID              string
	Title           string
	Priority        string
	PriClass        string
	Owner           string
	ActionLabel     string
	ActionClass     string
	DetailURL       template.URL
	DevRequirements []DevRequirement
}

// BizRequirement 业务需求行（树形一级）。
type BizRequirement struct {
	DemandID           uint
	ID                 string
	Title              string
	Priority           string
	PriClass           string
	WindowStatus       string
	WindowStatusClass  string
	AgileGroup         string
	StageTag           string
	StageTagClass      string
	VersionWindow      string
	Owner              string
	ActionLabel        string
	ActionClass        string
	DetailURL          template.URL
	HasChildren        bool
	SubBizRequirements []SubBizRequirement
	DevRequirements    []DevRequirement
}

type scheduleIndexDemandData struct {
	BizRequirements         []BizRequirement
	BizTotal                int64
	BizPager                *pagination.Pager
	IndependentRequirements []IndependentRequirement
	IndependentTotal        int64
	IndepPager              *pagination.Pager
}

func (h *Handler) loadScheduleIndexDemandData(c *gin.Context, actor *model.User, bizPage, indepPage int) (scheduleIndexDemandData, bool) {
	var listReq ListBizDemandsReq
	if err := c.ShouldBindQuery(&listReq); err != nil {
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return scheduleIndexDemandData{}, false
	}
	listReq.Page = bizPage
	listReq.PageSize = scheduleListPageSize
	listReq.Normalize()

	bizResp, err := h.svc.ListBizDemands(c.Request.Context(), actor, listReq)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load biz demands failed", zap.Error(err))
		}
		bizResp = &ListBizDemandsResp{Total: 0, Items: []BizDemandItem{}}
	}
	bizRequirements := toBizRequirementsView(bizResp.Items, h.zentaoURL)

	indepReq := ListIndependentReq{
		Page:     indepPage,
		PageSize: scheduleListPageSize,
	}
	indepReq.Normalize()

	indepResp, err := h.svc.ListIndependentStories(c.Request.Context(), actor, indepReq)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load independent stories failed", zap.Error(err))
		}
		indepResp = &ListIndependentResp{Total: 0, Items: []IndependentStoryItem{}}
	}
	if indepResp == nil {
		indepResp = &ListIndependentResp{Total: 0, Items: []IndependentStoryItem{}}
	}
	independentRequirements := toIndependentRequirementsView(indepResp.Items, h.zentaoURL)

	bizPager := pagination.New(bizResp.Total, bizPage, scheduleListPageSize)
	bizPager.PageParam = "bizPage"
	bizPager.PreserveParams = map[string]string{"indepPage": strconv.Itoa(indepPage)}

	indepPager := pagination.New(indepResp.Total, indepPage, scheduleListPageSize)
	indepPager.PageParam = "indepPage"
	indepPager.PreserveParams = map[string]string{
		"bizPage": strconv.Itoa(bizPage),
		"tab":     "indep",
	}

	return scheduleIndexDemandData{
		BizRequirements:         bizRequirements,
		BizTotal:                bizResp.Total,
		BizPager:                bizPager,
		IndependentRequirements: independentRequirements,
		IndependentTotal:        indepResp.Total,
		IndepPager:              indepPager,
	}, true
}

func parseDemandID(c *gin.Context) (uint, bool) {
	idStr := strings.TrimSpace(c.Param("id"))
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}

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
		}
	}
	c.JSON(http.StatusOK, out)
}

// SaveScheduling 保存排期一体化弹窗数据并同步禅道（JSON）。
func (h *Handler) SaveScheduling(c *gin.Context) {
	demandID, ok := parseDemandID(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}

	var req SaveSchedulingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}
	if errs := req.Validate(); len(errs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"message": formatFieldErrors(errs),
			"errors":  errs,
		})
		return
	}

	actor := middleware.CurrentUser(c)
	if err := h.svc.SaveScheduling(c.Request.Context(), actor, demandID, &req); err != nil {
		if h.logger != nil {
			h.logger.Error("save demand scheduling failed",
				zap.Error(err),
				zap.Uint("demand_id", demandID),
			)
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
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
