// =============================================================================
// 文件: internal/module/schedule/form.go
// 模块: 排期工作台
// 类型: action
// 职责: 定义排期模块请求/响应结构体。
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
	Teamgroups      []TeamgroupOption
	Products        []ZtProduct
	WindowTemplates []WindowTemplateItem
}

// WindowTemplateItem 版本窗口组织模板（新建弹窗「跟随组织窗口」用）。
type WindowTemplateItem struct {
	ID     uint64 `json:"id"`
	Label  string `json:"label"`
	Start  string `json:"start"`
	End    string `json:"end"`
	Online string `json:"online"`
}

// MatchingPlansReq 计划匹配查询请求。
type MatchingPlansReq struct {
	ProductID uint   `form:"product_id"`
	EndDate   string `form:"end_date"`
}

// Validate 校验计划匹配查询参数。
func (r MatchingPlansReq) Validate() []FieldError {
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

// CreateWindowForm 新建版本窗口保存请求。
type CreateWindowForm struct {
	ReleaseDate string               `json:"releaseDate"`
	Name        string               `json:"name"`
	StartDate   string               `json:"startDate"`
	TeamgroupID uint                 `json:"teamgroupId"`
	GroupSize   int                  `json:"groupSize"`
	Products    []WindowProductInput `json:"products"`
}

// Validate 校验新建版本窗口请求。
func (f CreateWindowForm) Validate() []FieldError {
	var errs []FieldError
	releaseDate := strings.TrimSpace(f.ReleaseDate)
	if releaseDate == "" {
		errs = append(errs, FieldError{Field: "releaseDate", Message: "预计上线日期不能为空"})
	} else if _, err := time.Parse("2006-01-02", releaseDate); err != nil {
		errs = append(errs, FieldError{Field: "releaseDate", Message: "预计上线日期格式无效"})
	}
	if strings.TrimSpace(f.Name) == "" {
		errs = append(errs, FieldError{Field: "name", Message: "窗口名称不能为空"})
	}
	startDate := strings.TrimSpace(f.StartDate)
	if startDate != "" {
		if _, err := time.Parse("2006-01-02", startDate); err != nil {
			errs = append(errs, FieldError{Field: "startDate", Message: "窗口开始日期格式无效"})
		}
	}
	if f.TeamgroupID == 0 {
		errs = append(errs, FieldError{Field: "teamgroupId", Message: "敏捷小组不能为空"})
	}
	if f.GroupSize < 0 {
		errs = append(errs, FieldError{Field: "groupSize", Message: "小组人数不能为负数"})
	}
	for i, product := range f.Products {
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

// WindowProductInput 版本窗口关联系统及计划同步选项。
type WindowProductInput struct {
	ProductID uint   `json:"productId"`
	SyncPlan  bool   `json:"syncPlan"`
	PlanTitle string `json:"planTitle"`
}

// FieldError 表单字段错误。
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
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
	ID          uint64                `json:"id"`
	ReleaseDate string                `json:"releaseDate"`
	Name        string                `json:"name"`
	StartDate   string                `json:"startDate"`
	TeamgroupID uint                  `json:"teamgroupId"`
	GroupSize   uint                  `json:"groupSize"`
	Products    []WindowProductDetail `json:"products"`
}
