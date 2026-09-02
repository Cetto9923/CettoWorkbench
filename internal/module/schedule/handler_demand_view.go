// =============================================================================
// 文件: internal/module/schedule/handler_demand_view.go
// 模块: 排期工作台
// 类型: action
// 职责: 业需与独立研发需求列表页面数据加载及视图模型。
// 依赖: internal/middleware
//
//	internal/model
//	internal/pkg/pagination
//	internal/pkg/render
//
// =============================================================================

package schedule

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/model"
	"workbench/internal/pkg/pagination"
	"workbench/internal/pkg/render"
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
