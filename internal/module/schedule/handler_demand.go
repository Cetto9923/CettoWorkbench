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
	"errors"
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
	"workbench/internal/pkg/zentao"
)

// IndependentChildRequirement 独立研发需求子行（树形二级）。
type IndependentChildRequirement struct {
	StoryID       uint
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
	StoryID       uint
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
	StoryID     uint
	ID          string
	Title       string
	Priority    string
	PriClass    string
	IsMain      bool
	Stage       string
	StageClass  string
	WindowName  string
	AgileGroup  string
	Owner       string
	TaskCount   int
	ActionLabel string
	ActionClass string
	DetailURL   template.URL
}

// SubBizRequirement 子业务需求行（树形二级）。
type SubBizRequirement struct {
	DemandID         uint
	ID               string
	Title            string
	Priority         string
	PriClass         string
	AgileGroup       string
	Stage            string
	StageClass       string
	WindowPhase      string
	WindowPhaseClass string
	WindowName       string
	Owner            string
	ActionLabel      string
	ActionClass      string
	DetailURL        template.URL
	DevRequirements  []DevRequirement
}

// BizRequirement 业务需求行（树形一级）。
type BizRequirement struct {
	DemandID           uint
	ID                 string
	Title              string
	Priority           string
	PriClass           string
	AgileGroup         string
	Stage              string
	StageClass         string
	WindowPhase        string
	WindowPhaseClass   string
	WindowName         string
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
	ActiveTab               string
	ActiveFilter            string
	SuspendedActive         bool
	SuspendedCount          int64
	BizFilterCounts         FilterCounts
	IndepFilterCounts       FilterCounts
	SelectedGroups          string
	SelectedProducts        string
	SelectedStages          string
	SelectedWindows         string
	SelectedKeyword         string
	SelectedPri             string
	SelectedWindowType      string
	SelectedDevOwner        string
	SelectedTestOwner       string
	SelectedAcceptOwner     string
	SelectedGroupMap        map[uint]bool
	SelectedProductMap      map[uint]bool
	SelectedStageMap        map[string]bool
	SelectedWindowMap       map[uint]bool
}

type scheduleFilterPreserveReq struct {
	filter      string
	suspended   bool
	bizPage     int
	indepPage   int
	tab         string
	groups      string
	products    string
	stages      string
	windows     string
	keyword     string
	pri         string
	windowType  string
	devOwner    string
	testOwner   string
	acceptOwner string
}

func scheduleFilterPreserveParams(req scheduleFilterPreserveReq) map[string]string {
	params := map[string]string{
		"filter": req.filter,
	}
	if req.suspended {
		params["suspended"] = "1"
	}
	if strings.TrimSpace(req.groups) != "" {
		params["groups"] = strings.TrimSpace(req.groups)
	}
	if strings.TrimSpace(req.products) != "" {
		params["products"] = strings.TrimSpace(req.products)
	}
	if strings.TrimSpace(req.stages) != "" {
		params["stages"] = strings.TrimSpace(req.stages)
	}
	if strings.TrimSpace(req.windows) != "" {
		params["windows"] = strings.TrimSpace(req.windows)
	}
	if strings.TrimSpace(req.keyword) != "" {
		params["keyword"] = strings.TrimSpace(req.keyword)
	}
	if strings.TrimSpace(req.pri) != "" {
		params["pri"] = strings.TrimSpace(req.pri)
	}
	if strings.TrimSpace(req.windowType) != "" {
		params["windowType"] = strings.TrimSpace(req.windowType)
	}
	if strings.TrimSpace(req.devOwner) != "" {
		params["dev"] = strings.TrimSpace(req.devOwner)
	}
	if strings.TrimSpace(req.testOwner) != "" {
		params["test"] = strings.TrimSpace(req.testOwner)
	}
	if strings.TrimSpace(req.acceptOwner) != "" {
		params["accept"] = strings.TrimSpace(req.acceptOwner)
	}
	if req.indepPage > 1 {
		params["indepPage"] = strconv.Itoa(req.indepPage)
	}
	if req.bizPage > 1 {
		params["bizPage"] = strconv.Itoa(req.bizPage)
	}
	if req.tab == "indep" {
		params["tab"] = req.tab
	}
	return params
}

func selectedUintMap(raw string) map[uint]bool {
	ids := ParseCommaSeparatedUints(raw)
	out := make(map[uint]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out
}

func selectedStageMap(raw string) map[string]bool {
	stages := ParseCommaSeparatedStages(raw)
	out := make(map[string]bool, len(stages))
	for _, stage := range stages {
		out[stage] = true
	}
	return out
}

func (h *Handler) loadScheduleIndexDemandData(c *gin.Context, actor *model.User, bizPage, indepPage int) (scheduleIndexDemandData, bool) {
	tab := strings.TrimSpace(c.Query("tab"))
	if tab != "indep" {
		tab = "biz"
	}

	var listReq ListBizDemandsReq
	if err := c.ShouldBindQuery(&listReq); err != nil {
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return scheduleIndexDemandData{}, false
	}
	listReq.Page = bizPage
	listReq.PageSize = scheduleListPageSize
	listReq.Normalize()
	rawStages := listReq.Stages
	bizStages := NormalizeStageFilterForTab(rawStages, "biz")
	indepStages := NormalizeStageFilterForTab(rawStages, "indep")
	listReq.Stages = bizStages
	activeFilter := listReq.Filter
	suspendedActive := listReq.Suspended

	bizResp, err := h.svc.ListBizDemands(c.Request.Context(), actor, listReq)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load biz demands failed", zap.Error(err))
		}
		bizResp = &ListBizDemandsResp{Total: 0, Items: []BizDemandItem{}}
	}
	bizRequirements := toBizRequirementsView(bizResp.Items, h.zentaoURL)

	var indepReq ListIndependentReq
	if err := c.ShouldBindQuery(&indepReq); err != nil {
		render.Error(c, http.StatusBadRequest, "参数解析失败", err)
		return scheduleIndexDemandData{}, false
	}
	indepReq.Page = indepPage
	indepReq.PageSize = scheduleListPageSize
	indepReq.Filter = activeFilter
	indepReq.Suspended = false
	indepReq.Groups = listReq.Groups
	indepReq.Products = listReq.Products
	indepReq.Stages = indepStages
	indepReq.Windows = listReq.Windows
	indepReq.Keyword = listReq.Keyword
	indepReq.Pri = listReq.Pri
	indepReq.WindowType = listReq.WindowType
	indepReq.DevOwner = listReq.DevOwner
	indepReq.TestOwner = listReq.TestOwner
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

	bizFilterCounts, err := h.svc.GetBizDemandFilterCounts(c.Request.Context(), actor, activeFilter)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load biz filter counts failed", zap.Error(err))
		}
		bizFilterCounts = FilterCounts{}
	}
	indepFilterCounts, err := h.svc.GetIndependentFilterCounts(c.Request.Context(), actor)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("load independent filter counts failed", zap.Error(err))
		}
		indepFilterCounts = FilterCounts{}
	}
	selectedStages := bizStages
	if tab == "indep" {
		selectedStages = indepStages
	}

	bizPager := pagination.New(bizResp.Total, bizPage, scheduleListPageSize)
	bizPager.PageParam = "bizPage"
	bizPager.PreserveParams = scheduleFilterPreserveParams(scheduleFilterPreserveReq{
		filter: activeFilter, suspended: suspendedActive, bizPage: bizPage, indepPage: indepPage, tab: tab,
		groups: listReq.Groups, products: listReq.Products, stages: bizStages, windows: listReq.Windows,
		keyword: listReq.Keyword, pri: listReq.Pri, windowType: listReq.WindowType,
		devOwner: listReq.DevOwner, testOwner: listReq.TestOwner, acceptOwner: listReq.AcceptOwner,
	})

	indepPager := pagination.New(indepResp.Total, indepPage, scheduleListPageSize)
	indepPager.PageParam = "indepPage"
	indepPager.PreserveParams = scheduleFilterPreserveParams(scheduleFilterPreserveReq{
		filter: activeFilter, suspended: suspendedActive, bizPage: bizPage, indepPage: indepPage, tab: "indep",
		groups: listReq.Groups, products: listReq.Products, stages: indepStages, windows: listReq.Windows,
		keyword: listReq.Keyword, pri: listReq.Pri, windowType: listReq.WindowType,
		devOwner: listReq.DevOwner, testOwner: listReq.TestOwner, acceptOwner: listReq.AcceptOwner,
	})

	return scheduleIndexDemandData{
		BizRequirements:         bizRequirements,
		BizTotal:                bizResp.Total,
		BizPager:                bizPager,
		IndependentRequirements: independentRequirements,
		IndependentTotal:        indepResp.Total,
		IndepPager:              indepPager,
		ActiveTab:               tab,
		ActiveFilter:            activeFilter,
		SuspendedActive:         suspendedActive,
		SuspendedCount:          bizFilterCounts.Suspended,
		BizFilterCounts:         bizFilterCounts,
		IndepFilterCounts:       indepFilterCounts,
		SelectedGroups:          listReq.Groups,
		SelectedProducts:        listReq.Products,
		SelectedStages:          selectedStages,
		SelectedWindows:         listReq.Windows,
		SelectedKeyword:         listReq.Keyword,
		SelectedPri:             listReq.Pri,
		SelectedWindowType:      listReq.WindowType,
		SelectedDevOwner:        listReq.DevOwner,
		SelectedTestOwner:       listReq.TestOwner,
		SelectedAcceptOwner:     listReq.AcceptOwner,
		SelectedGroupMap:        selectedUintMap(listReq.Groups),
		SelectedProductMap:      selectedUintMap(listReq.Products),
		SelectedStageMap:        selectedStageMap(selectedStages),
		SelectedWindowMap:       selectedUintMap(listReq.Windows),
	}, true
}

func parseDemandID(c *gin.Context) (uint, bool) {
	return parseStoryID(c)
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
		// 业务前置校验拦截：零写入，前端弹警告框引导去禅道维护。
		var notice *ProductAccessNoticeError
		if errors.As(err, &notice) {
			for i := range notice.Products {
				notice.Products[i].ViewURL = zentao.ProductViewURLWithBase(h.zentaoURL, notice.Products[i].ID)
			}
			c.JSON(http.StatusOK, gin.H{
				"success":  false,
				"code":     "PRODUCT_ACCESS_NOTICE",
				"products": notice.Products,
			})
			return
		}
		var businessErr *SchedulingBusinessError
		if errors.As(err, &businessErr) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"success": false,
				"message": businessErr.Error(),
			})
			return
		}
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

// SaveStoryScheduling 保存独立研发需求排期并同步禅道（JSON）。
func (h *Handler) SaveStoryScheduling(c *gin.Context) {
	storyID, ok := parseStoryID(c)
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
	if err := h.svc.SaveStoryScheduling(c.Request.Context(), actor, storyID, &req); err != nil {
		// 业务前置校验拦截（PRODUCT_ACCESS_NOTICE）：零写入，前端弹警告引导去禅道。
		var notice *ProductAccessNoticeError
		if errors.As(err, &notice) {
			for i := range notice.Products {
				notice.Products[i].ViewURL = zentao.ProductViewURLWithBase(h.zentaoURL, notice.Products[i].ID)
			}
			c.JSON(http.StatusOK, gin.H{
				"success":  false,
				"code":     "PRODUCT_ACCESS_NOTICE",
				"products": notice.Products,
			})
			return
		}
		if h.logger != nil {
			h.logger.Error("save story scheduling failed",
				zap.Error(err),
				zap.Uint("story_id", storyID),
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
