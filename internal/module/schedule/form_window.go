// =============================================================================
// 文件: internal/module/schedule/form_window.go
// 模块: 排期工作台
// 类型: action
// 职责: 版本窗口模块请求/响应结构体与校验。
// 依赖: 无
// =============================================================================

package schedule

import (
	"strconv"
	"strings"
	"time"
)

// TeamgroupOption 敏捷小组下拉选项。
type TeamgroupOption struct {
	ID          uint
	DisplayName string
}

// CreateWindowFormData 新建版本窗口弹窗表单数据。
type CreateWindowFormData struct {
	Teamgroups []TeamgroupOption
	Products   []ZtProduct
}

// MatchingPlansReq 计划匹配查询请求。
type MatchingPlansReq struct {
	ProductID uint   `form:"product_id"`
	EndDate   string `form:"end_date"`
}

// Validate 校验计划匹配查询参数。
func (r *MatchingPlansReq) Validate() []FieldError {
	var errs []FieldError
	if r.ProductID == 0 {
		errs = append(errs, FieldError{Field: "product_id", Message: "产品 ID 不能为空"})
	}
	endDate := strings.TrimSpace(r.EndDate)
	if endDate == "" {
		errs = append(errs, FieldError{Field: "end_date", Message: "结束日期不能为空"})
	} else if len(endDate) != 10 || endDate[4] != '-' || endDate[7] != '-' {
		errs = append(errs, FieldError{Field: "end_date", Message: "结束日期格式无效"})
	}
	return errs
}

// MatchingPlanItem 计划匹配结果项。
type MatchingPlanItem struct {
	ID     uint   `json:"id"`
	Title  string `json:"title"`
	Begin  string `json:"begin"`
	End    string `json:"end"`
	Status string `json:"status"`
}

// MatchingPlansResp 计划匹配查询响应。
type MatchingPlansResp struct {
	Plans    []MatchingPlanItem `json:"plans"`
	HasMatch bool               `json:"has_match"`
}

// CreateReq 新建版本窗口保存请求。
type CreateReq struct {
	ReleaseDate  string               `json:"releaseDate"`
	Name         string               `json:"name"`
	StartDate    string               `json:"startDate"`
	PlanTestDone string               `json:"planTestDone"`
	TestDone     string               `json:"testDone"`
	AcceptDone   string               `json:"acceptDone"`
	TeamgroupID  uint                 `json:"teamgroupId"`
	GroupSize    int                  `json:"groupSize"`
	Products     []WindowProductInput `json:"products"`
}

// Validate 校验新建版本窗口请求。
func (r *CreateReq) Validate() []FieldError {
	errs := validateWindowSaveFields(
		r.ReleaseDate,
		r.Name,
		r.StartDate,
		r.TeamgroupID,
		r.GroupSize,
		r.Products,
	)
	errs = append(errs, validateWindowMilestoneFields(r.PlanTestDone, r.TestDone, r.AcceptDone)...)
	return errs
}

// UpdateReq 更新版本窗口请求。
type UpdateReq struct {
	ID           uint64               `json:"-"`
	ReleaseDate  string               `json:"releaseDate"`
	Name         string               `json:"name"`
	StartDate    string               `json:"startDate"`
	PlanTestDone string               `json:"planTestDone"`
	TestDone     string               `json:"testDone"`
	AcceptDone   string               `json:"acceptDone"`
	TeamgroupID  uint                 `json:"teamgroupId"`
	GroupSize    int                  `json:"groupSize"`
	Products     []WindowProductInput `json:"products"`
}

// Validate 校验更新版本窗口请求。
func (r *UpdateReq) Validate() []FieldError {
	var errs []FieldError
	if r.ID == 0 {
		errs = append(errs, FieldError{Field: "id", Message: "窗口 ID 无效"})
	}
	errs = append(errs, validateWindowSaveFields(
		r.ReleaseDate,
		r.Name,
		r.StartDate,
		r.TeamgroupID,
		r.GroupSize,
		r.Products,
	)...)
	errs = append(errs, validateWindowMilestoneFields(r.PlanTestDone, r.TestDone, r.AcceptDone)...)
	return errs
}

// DeleteReq 删除版本窗口请求。
type DeleteReq struct {
	ID uint64
}

// Validate 校验删除版本窗口请求。
func (r *DeleteReq) Validate() []FieldError {
	if r.ID == 0 {
		return []FieldError{{Field: "id", Message: "窗口 ID 无效"}}
	}
	return nil
}

// ListWindowsResp 版本窗口维护列表响应。
type ListWindowsResp struct {
	Windows []WindowListItem
}

// WindowListItem 版本窗口维护列表项。
type WindowListItem struct {
	ID               uint64
	Name             string
	ReleaseDate      string
	PlanTestDone     string
	TestDone         string
	AcceptDone       string
	Range            string
	DemandCount      int
	CapacityHours    int
	UsedHours        int
	RemainingHours   int
	BlockedCount     int
	UsedPercent      int
	CanEdit          bool
	CanDelete        bool
	HasLinkedDemands bool
}

// WindowProductInput 版本窗口关联系统及计划同步选项。
type WindowProductInput struct {
	ProductID uint   `json:"productId"`
	SyncPlan  bool   `json:"syncPlan"`
	PlanTitle string `json:"planTitle"`
}

// FieldError 字段级验证错误。
type FieldError struct {
	Field   string
	Message string
}

// WindowProductDetail 版本窗口关联产品及计划详情。
type WindowProductDetail struct {
	ProductID   uint               `json:"productId"`
	ProductName string             `json:"productName"`
	SyncPlan    bool               `json:"syncPlan"`
	PlanTitle   string             `json:"planTitle"`
	PlanID      *uint              `json:"planId,omitempty"`
	HasMatch    bool               `json:"hasMatch"`
	Plans       []MatchingPlanItem `json:"plans,omitempty"`
}

// WindowDetailResp 版本窗口详情响应。
type WindowDetailResp struct {
	ID           uint64                `json:"id"`
	ReleaseDate  string                `json:"releaseDate"`
	Name         string                `json:"name"`
	StartDate    string                `json:"startDate"`
	PlanTestDone string                `json:"planTestDone"`
	TestDone     string                `json:"testDone"`
	AcceptDone   string                `json:"acceptDone"`
	TeamgroupID  uint                  `json:"teamgroupId"`
	GroupSize    uint                  `json:"groupSize"`
	Products     []WindowProductDetail `json:"products"`
}

func validateWindowSaveFields(
	releaseDate, name, startDate string,
	teamgroupID uint,
	groupSize int,
	products []WindowProductInput,
) []FieldError {
	var errs []FieldError
	releaseDate = strings.TrimSpace(releaseDate)
	if releaseDate == "" {
		errs = append(errs, FieldError{Field: "releaseDate", Message: "预计上线日期不能为空"})
	} else if _, err := time.Parse("2006-01-02", releaseDate); err != nil {
		errs = append(errs, FieldError{Field: "releaseDate", Message: "预计上线日期格式无效"})
	}
	if strings.TrimSpace(name) == "" {
		errs = append(errs, FieldError{Field: "name", Message: "窗口名称不能为空"})
	}
	startDate = strings.TrimSpace(startDate)
	if startDate == "" {
		errs = append(errs, FieldError{Field: "startDate", Message: "窗口开始日期不能为空"})
	} else if _, err := time.Parse("2006-01-02", startDate); err != nil {
		errs = append(errs, FieldError{Field: "startDate", Message: "窗口开始日期格式无效"})
	}
	if teamgroupID == 0 {
		errs = append(errs, FieldError{Field: "teamgroupId", Message: "敏捷小组不能为空"})
	}
	if groupSize < 0 {
		errs = append(errs, FieldError{Field: "groupSize", Message: "小组人数不能为负数"})
	}
	for i, product := range products {
		if product.ProductID == 0 {
			errs = append(errs, FieldError{
				Field:   "products",
				Message: "第 " + strconv.Itoa(i+1) + " 个关联系统 ID 不能为空",
			})
		}
		if product.SyncPlan && strings.TrimSpace(product.PlanTitle) == "" {
			errs = append(errs, FieldError{
				Field:   "products",
				Message: "第 " + strconv.Itoa(i+1) + " 个系统勾选同步创建计划时，计划名称不能为空",
			})
		}
	}
	return errs
}

func validateWindowMilestoneFields(planTestDone, testDone, acceptDone string) []FieldError {
	fields := []struct {
		field string
		value string
		label string
	}{
		{field: "planTestDone", value: planTestDone, label: "预计提测/开发完成日期"},
		{field: "testDone", value: testDone, label: "预计测试完成日期"},
		{field: "acceptDone", value: acceptDone, label: "预计验收完成日期"},
	}
	var errs []FieldError
	for _, f := range fields {
		value := strings.TrimSpace(f.value)
		if value == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", value); err != nil {
			errs = append(errs, FieldError{Field: f.field, Message: f.label + "格式无效"})
		}
	}
	return errs
}
