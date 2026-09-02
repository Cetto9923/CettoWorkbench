// =============================================================================
// 文件: internal/module/schedule/formshared.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期模块共享类型(TeamgroupOption / CreateWindowFormData / FieldError 基类型)。
//
//	拆分自 form.go:共用基类型集中放置,避免重复定义。
//
// 依赖: 无
// =============================================================================
package schedule

import (
	"strings"
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
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Begin string `json:"begin"`
	End   string `json:"end"`
}

// MatchingPlansResp 计划匹配查询响应。
type MatchingPlansResp struct {
	Plans    []MatchingPlanItem `json:"plans"`
	HasMatch bool               `json:"has_match"`
}

// CreateReq 新建版本窗口保存请求。
type CreateReq struct {
	ReleaseDate string               `json:"releaseDate"`
	Name        string               `json:"name"`
	StartDate   string               `json:"startDate"`
	TeamgroupID uint                 `json:"teamgroupId"`
	GroupSize   int                  `json:"groupSize"`
	Products    []WindowProductInput `json:"products"`
}

// Validate 校验新建版本窗口请求。
func (r *CreateReq) Validate() []FieldError {
	return validateWindowSaveFields(
		r.ReleaseDate,
		r.Name,
		r.StartDate,
		r.TeamgroupID,
		r.GroupSize,
		r.Products,
	)
}

// UpdateReq 更新版本窗口请求。
type UpdateReq struct {
	ID          uint64               `json:"-"`
	ReleaseDate string               `json:"releaseDate"`
	Name        string               `json:"name"`
	StartDate   string               `json:"startDate"`
	TeamgroupID uint                 `json:"teamgroupId"`
	GroupSize   int                  `json:"groupSize"`
	Products    []WindowProductInput `json:"products"`
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
