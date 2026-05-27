// =============================================================================
// 文件: internal/module/dictitem/form.go
// 模块: 字典值
// 类型: crud
// 职责: 定义字典值模块请求/响应结构与校验逻辑。
// 依赖: internal/model
// =============================================================================

package dictitem

import (
	"strings"

	"workbrench/internal/model"
)

// FieldError 字段级验证错误。
type FieldError struct {
	Field   string
	Message string
}

// ListReq 字典值列表请求。
type ListReq struct {
	TypeCode string `form:"typeCode"`
}

// Normalize 规范化查询参数。
func (r *ListReq) Normalize() {
	r.TypeCode = strings.TrimSpace(r.TypeCode)
}

// ListResp 字典值列表响应。
type ListResp struct {
	Items []model.DictItem
}

// CreateReq 新增字典值请求。
type CreateReq struct {
	TypeCode string `form:"typeCode"`
	Label    string `form:"label"`
	Value    string `form:"value"`
	Sort     int    `form:"sort"`
}

// Validate 校验新增请求。
func (r *CreateReq) Validate() []FieldError {
	errs := make([]FieldError, 0)
	if strings.TrimSpace(r.TypeCode) == "" {
		errs = append(errs, FieldError{Field: "typeCode", Message: "字典类型编码不能为空"})
	}
	if len([]rune(strings.TrimSpace(r.TypeCode))) > 64 {
		errs = append(errs, FieldError{Field: "typeCode", Message: "字典类型编码长度不能超过 64"})
	}
	if strings.TrimSpace(r.Label) == "" {
		errs = append(errs, FieldError{Field: "label", Message: "字典项名称不能为空"})
	}
	if len([]rune(strings.TrimSpace(r.Label))) > 64 {
		errs = append(errs, FieldError{Field: "label", Message: "字典项名称长度不能超过 64"})
	}
	if strings.TrimSpace(r.Value) == "" {
		errs = append(errs, FieldError{Field: "value", Message: "字典项值不能为空"})
	}
	if len([]rune(strings.TrimSpace(r.Value))) > 64 {
		errs = append(errs, FieldError{Field: "value", Message: "字典项值长度不能超过 64"})
	}
	return errs
}

// CreateResp 新增字典值响应。
type CreateResp struct {
	ID uint64
}

// UpdateReq 编辑字典值请求。
type UpdateReq struct {
	ID       uint64 `form:"-"`
	TypeCode string `form:"typeCode"`
	Label    string `form:"label"`
	Value    string `form:"value"`
	Sort     int    `form:"sort"`
}

// Validate 校验编辑请求。
func (r *UpdateReq) Validate() []FieldError {
	errs := make([]FieldError, 0)
	if r.ID == 0 {
		errs = append(errs, FieldError{Field: "id", Message: "无效的字典值 ID"})
	}
	if strings.TrimSpace(r.TypeCode) == "" {
		errs = append(errs, FieldError{Field: "typeCode", Message: "字典类型编码不能为空"})
	}
	if len([]rune(strings.TrimSpace(r.TypeCode))) > 64 {
		errs = append(errs, FieldError{Field: "typeCode", Message: "字典类型编码长度不能超过 64"})
	}
	if strings.TrimSpace(r.Label) == "" {
		errs = append(errs, FieldError{Field: "label", Message: "字典项名称不能为空"})
	}
	if len([]rune(strings.TrimSpace(r.Label))) > 64 {
		errs = append(errs, FieldError{Field: "label", Message: "字典项名称长度不能超过 64"})
	}
	if strings.TrimSpace(r.Value) == "" {
		errs = append(errs, FieldError{Field: "value", Message: "字典项值不能为空"})
	}
	if len([]rune(strings.TrimSpace(r.Value))) > 64 {
		errs = append(errs, FieldError{Field: "value", Message: "字典项值长度不能超过 64"})
	}
	return errs
}

// DeleteReq 删除字典值请求。
type DeleteReq struct {
	ID uint64
}

// NewUpdateReqFromModel 从实体构造编辑请求。
func NewUpdateReqFromModel(m *model.DictItem) *UpdateReq {
	if m == nil {
		return &UpdateReq{}
	}
	return &UpdateReq{
		ID:       m.ID,
		TypeCode: m.TypeCode,
		Label:    m.Label,
		Value:    m.Value,
		Sort:     m.Sort,
	}
}
