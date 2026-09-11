// =============================================================================
// 文件: internal/module/po/form_demand_edit.go
// 模块: PO 工作台
// 类型: form
// 职责: 业务需求草稿/驳回状态编辑与删除入参表单及元数据结构。
// =============================================================================

package po

import (
	"strings"
	"time"
)

// UpdateDemandReq 业务需求编辑更新入参。
type UpdateDemandReq struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Category       string `json:"category"`
	Pri            string `json:"pri"`
	Pool           int    `json:"pool"`
	Product        string `json:"product"`
	Reviewer       string `json:"reviewer"`
	EstimateLaunch string `json:"estimateLaunch"`
	Source         string `json:"source"`
	SourceNote     string `json:"sourceNote"`
	Desc           string `json:"desc"`
	VerifyPlan     string `json:"verifyPlan"`
	SubmitReview   bool   `json:"submitReview"` // 是否保存后直接提交评审
	Comment        string `json:"comment"`
}

// Validate 校验更新入参。
func (r *UpdateDemandReq) Validate() []FieldError {
	var errs []FieldError
	if r.ID <= 0 {
		errs = append(errs, FieldError{Field: "id", Message: "需求 ID 无效"})
	}
	name := strings.TrimSpace(r.Name)
	if name == "" {
		errs = append(errs, FieldError{Field: "name", Message: "需求名称不能为空"})
	} else if len([]rune(name)) > 200 {
		errs = append(errs, FieldError{Field: "name", Message: "需求名称长度不能超过 200 个字"})
	}

	if r.SubmitReview {
		if strings.TrimSpace(r.Reviewer) == "" {
			errs = append(errs, FieldError{Field: "reviewer", Message: "提交业务评审时必须指定业务评审人"})
		}
		if strings.TrimSpace(r.Desc) == "" {
			errs = append(errs, FieldError{Field: "desc", Message: "提交业务评审时需求描述不能为空"})
		}
	}

	el := strings.TrimSpace(r.EstimateLaunch)
	if el != "" && el != "—" {
		if _, err := time.Parse("2006-01-02", el); err != nil {
			errs = append(errs, FieldError{Field: "estimateLaunch", Message: "期望上线日期格式无效，应为 YYYY-MM-DD"})
		}
	}
	return errs
}

// DeleteDemandReq 业务需求删除入参。
type DeleteDemandReq struct {
	ID      int64  `json:"id"`
	Comment string `json:"comment"`
}

// Validate 校验删除入参。
func (r *DeleteDemandReq) Validate() []FieldError {
	if r.ID <= 0 {
		return []FieldError{{Field: "id", Message: "需求 ID 无效"}}
	}
	return nil
}

// DemandEditOptionItem 通用选项项。
type DemandEditOptionItem struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// DemandEditOptions 编辑表单下拉字典选项。
type DemandEditOptions struct {
	Categories []DemandEditOptionItem `json:"categories"`
	Priorities []DemandEditOptionItem `json:"priorities"`
	Pools      []DemandEditOptionItem `json:"pools"`
	Products   []DemandEditOptionItem `json:"products"`
	Reviewers  []DemandEditOptionItem `json:"reviewers"`
	Sources    []DemandEditOptionItem `json:"sources"`
}

// DemandEditFields 待编辑的需求字段。
type DemandEditFields struct {
	ID             int64  `json:"id"`
	Code           string `json:"code"`
	Name           string `json:"name"`
	Status         string `json:"status"`
	StatusLabel    string `json:"statusLabel"`
	Category       string `json:"category"`
	Pri            string `json:"pri"`
	Pool           int    `json:"pool"`
	PoolName       string `json:"poolName"`
	Product        string `json:"product"`
	ProductName    string `json:"productName"`
	Reviewer       string `json:"reviewer"`
	ReviewerName   string `json:"reviewerName"`
	EstimateLaunch string `json:"estimateLaunch"`
	Source         string `json:"source"`
	SourceNote     string `json:"sourceNote"`
	Desc           string `json:"desc"`
	VerifyPlan     string `json:"verifyPlan"`
	CreatedBy      string `json:"createdBy"`
	CreatedByName  string `json:"createdByName"`
	IsCreator      bool   `json:"isCreator"`
}

// DemandEditDataResp 编辑表单初始化上下文响应。
type DemandEditDataResp struct {
	Success   bool              `json:"success"`
	Demand    DemandEditFields  `json:"demand"`
	Options   DemandEditOptions `json:"options"`
	CanEdit   bool              `json:"canEdit"`
	CanDelete bool              `json:"canDelete"`
}
