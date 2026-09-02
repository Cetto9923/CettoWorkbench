// =============================================================================
// 文件: internal/module/schedule/formwindow.go
// 模块: 排期工作台
// 类型: action
// 职责: 版本窗口管理相关 Request / Response 结构体与字段校验。
//       拆分自 form.go:窗口 CRUD + MatchingPlans + ListWindows 集中。
// 依赖: internal/module/schedule/formshared.go (FieldError)
// =============================================================================

package schedule

import (
	"strconv"
	"strings"
	"time"
)

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
