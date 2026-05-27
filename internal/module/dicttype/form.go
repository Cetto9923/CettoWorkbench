// =============================================================================
// 文件: internal/module/dicttype/form.go
// 模块: 字典类型
// 类型: crud
// 职责: 定义字典类型模块请求/响应结构与校验逻辑。
// 依赖: internal/model
// =============================================================================

package dicttype

import (
	"strings"

	"goframework/internal/model"
)

// FieldError 字段级验证错误。
type FieldError struct {
	Field   string
	Message string
}

// ListReq 字典类型列表请求。
type ListReq struct {
	Keyword string `form:"q"`
}

// Normalize 规范化查询参数。
func (r *ListReq) Normalize() {
	r.Keyword = strings.TrimSpace(r.Keyword)
}

// ListResp 字典类型列表响应。
type ListResp struct {
	Items []model.DictType
}

// CreateReq 新增字典类型请求。
type CreateReq struct {
	Code   string `form:"code"`
	Name   string `form:"name"`
	Remark string `form:"remark"`
}

// Validate 校验新增请求。
func (r *CreateReq) Validate() []FieldError {
	errs := make([]FieldError, 0)
	if strings.TrimSpace(r.Code) == "" {
		errs = append(errs, FieldError{Field: "code", Message: "类型编码不能为空"})
	}
	if len([]rune(strings.TrimSpace(r.Code))) > 64 {
		errs = append(errs, FieldError{Field: "code", Message: "类型编码长度不能超过 64"})
	}
	if strings.TrimSpace(r.Name) == "" {
		errs = append(errs, FieldError{Field: "name", Message: "类型名称不能为空"})
	}
	if len([]rune(strings.TrimSpace(r.Name))) > 64 {
		errs = append(errs, FieldError{Field: "name", Message: "类型名称长度不能超过 64"})
	}
	if len([]rune(strings.TrimSpace(r.Remark))) > 255 {
		errs = append(errs, FieldError{Field: "remark", Message: "备注长度不能超过 255"})
	}
	return errs
}

// CreateResp 新增字典类型响应。
type CreateResp struct {
	ID uint64
}

// UpdateReq 编辑字典类型请求。
type UpdateReq struct {
	ID     uint64 `form:"-"`
	Code   string `form:"code"`
	Name   string `form:"name"`
	Remark string `form:"remark"`
}

// Validate 校验编辑请求。
func (r *UpdateReq) Validate() []FieldError {
	errs := make([]FieldError, 0)
	if r.ID == 0 {
		errs = append(errs, FieldError{Field: "id", Message: "无效的字典类型 ID"})
	}
	if strings.TrimSpace(r.Code) == "" {
		errs = append(errs, FieldError{Field: "code", Message: "类型编码不能为空"})
	}
	if len([]rune(strings.TrimSpace(r.Code))) > 64 {
		errs = append(errs, FieldError{Field: "code", Message: "类型编码长度不能超过 64"})
	}
	if strings.TrimSpace(r.Name) == "" {
		errs = append(errs, FieldError{Field: "name", Message: "类型名称不能为空"})
	}
	if len([]rune(strings.TrimSpace(r.Name))) > 64 {
		errs = append(errs, FieldError{Field: "name", Message: "类型名称长度不能超过 64"})
	}
	if len([]rune(strings.TrimSpace(r.Remark))) > 255 {
		errs = append(errs, FieldError{Field: "remark", Message: "备注长度不能超过 255"})
	}
	return errs
}

// DeleteReq 删除字典类型请求。
type DeleteReq struct {
	ID uint64
}

// NewUpdateReqFromModel 从实体构造编辑请求。
func NewUpdateReqFromModel(m *model.DictType) *UpdateReq {
	if m == nil {
		return &UpdateReq{}
	}
	return &UpdateReq{
		ID:     m.ID,
		Code:   m.Code,
		Name:   m.Name,
		Remark: m.Remark,
	}
}
