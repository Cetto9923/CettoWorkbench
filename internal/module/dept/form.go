// =============================================================================
// 文件: internal/module/dept/form.go
// 模块: 部门管理
// 类型: crud
// 职责: 定义部门模块请求/响应结构体和校验逻辑。
// 依赖: internal/model
// =============================================================================

package dept

import (
	"strings"

	"workbench/internal/model"
)

// FieldError 字段级验证错误。
type FieldError struct {
	Field   string
	Message string
}

// ListReq 部门列表请求。
type ListReq struct{}

// ListResp 部门列表响应。
type ListResp struct {
	Items []*DeptNode
}

// CreateReq 新增部门请求。
type CreateReq struct {
	ParentID        uint64 `form:"parentId"`
	Name            string `form:"name" binding:"required,max=64"`
	Leader          string `form:"leader" binding:"max=64"`
	Phone           string `form:"phone" binding:"max=32"`
	Email           string `form:"email" binding:"omitempty,email,max=64"`
	Status          uint8  `form:"status" binding:"oneof=0 1"`
	Sort            int    `form:"sort"`
	EnableAncestors bool   `form:"enableAncestors"`
}

// Validate 校验新增部门请求。
func (r *CreateReq) Validate() []FieldError {
	errs := make([]FieldError, 0)
	name := strings.TrimSpace(r.Name)
	if name == "" {
		errs = append(errs, FieldError{Field: "name", Message: "部门名称不能为空"})
	}
	if len([]rune(name)) > 64 {
		errs = append(errs, FieldError{Field: "name", Message: "部门名称长度不能超过 64"})
	}
	if len([]rune(strings.TrimSpace(r.Leader))) > 64 {
		errs = append(errs, FieldError{Field: "leader", Message: "负责人长度不能超过 64"})
	}
	if len([]rune(strings.TrimSpace(r.Phone))) > 32 {
		errs = append(errs, FieldError{Field: "phone", Message: "联系电话长度不能超过 32"})
	}
	if len([]rune(strings.TrimSpace(r.Email))) > 64 {
		errs = append(errs, FieldError{Field: "email", Message: "邮箱长度不能超过 64"})
	}
	if r.Status != 0 && r.Status != 1 {
		errs = append(errs, FieldError{Field: "status", Message: "状态值不合法"})
	}
	return errs
}

// CreateResp 新增部门响应。
type CreateResp struct {
	ID uint64
}

// UpdateReq 编辑部门请求。
type UpdateReq struct {
	ID       uint64 `form:"-"`
	ParentID uint64 `form:"parentId"`
	Name     string `form:"name" binding:"required,max=64"`
	Leader   string `form:"leader" binding:"max=64"`
	Phone    string `form:"phone" binding:"max=32"`
	Email    string `form:"email" binding:"omitempty,email,max=64"`
	Status   uint8  `form:"status" binding:"oneof=0 1"`
	Sort     int    `form:"sort"`
}

// UpdateStatusReq 更新部门状态请求。
type UpdateStatusReq struct {
	ID            uint64 `form:"-"`
	Status        uint8  `form:"status" binding:"oneof=0 1"`
	WithAncestors bool   `form:"withAncestors"`
}

// Validate 校验状态更新请求。
func (r *UpdateStatusReq) Validate() []FieldError {
	errs := make([]FieldError, 0)
	if r.ID == 0 {
		errs = append(errs, FieldError{Field: "id", Message: "无效的部门 ID"})
	}
	if r.Status != 0 && r.Status != 1 {
		errs = append(errs, FieldError{Field: "status", Message: "状态值不合法"})
	}
	return errs
}

// Validate 校验编辑部门请求。
func (r *UpdateReq) Validate() []FieldError {
	errs := make([]FieldError, 0)
	if r.ID == 0 {
		errs = append(errs, FieldError{Field: "id", Message: "无效的部门 ID"})
	}
	name := strings.TrimSpace(r.Name)
	if name == "" {
		errs = append(errs, FieldError{Field: "name", Message: "部门名称不能为空"})
	}
	if len([]rune(name)) > 64 {
		errs = append(errs, FieldError{Field: "name", Message: "部门名称长度不能超过 64"})
	}
	if len([]rune(strings.TrimSpace(r.Leader))) > 64 {
		errs = append(errs, FieldError{Field: "leader", Message: "负责人长度不能超过 64"})
	}
	if len([]rune(strings.TrimSpace(r.Phone))) > 32 {
		errs = append(errs, FieldError{Field: "phone", Message: "联系电话长度不能超过 32"})
	}
	if len([]rune(strings.TrimSpace(r.Email))) > 64 {
		errs = append(errs, FieldError{Field: "email", Message: "邮箱长度不能超过 64"})
	}
	if r.Status != 0 && r.Status != 1 {
		errs = append(errs, FieldError{Field: "status", Message: "状态值不合法"})
	}
	return errs
}

// DeleteReq 删除部门请求。
type DeleteReq struct {
	ID uint64
}

// NewUpdateReqFromDept 从实体构造编辑请求。
func NewUpdateReqFromDept(d *model.Dept) *UpdateReq {
	if d == nil {
		return &UpdateReq{}
	}
	return &UpdateReq{
		ID:       d.ID,
		ParentID: d.ParentID,
		Name:     d.Name,
		Leader:   d.Leader,
		Phone:    d.Phone,
		Email:    d.Email,
		Status:   d.Status,
		Sort:     d.Sort,
	}
}
