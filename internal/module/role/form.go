// =============================================================================
// 文件: internal/module/role/form.go
// 模块: 角色管理
// 类型: crud
// 职责: 定义角色模块请求/响应结构与校验逻辑。
// 依赖: internal/model
// =============================================================================

package role

import (
	"strings"
	"unicode"

	"workbench/internal/model"
)

// FieldError 字段级验证错误。
type FieldError struct {
	Field   string
	Message string
}

// ListReq 角色列表查询请求。
type ListReq struct {
	Keyword  string `form:"q"`
	Page     int    `form:"page"`
	PageSize int    `form:"-"`
}

// Normalize 规范化分页与关键字。
func (r *ListReq) Normalize() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = 20
	}
	if r.PageSize > 100 {
		r.PageSize = 100
	}
	r.Keyword = strings.TrimSpace(r.Keyword)
}

// ListResp 角色列表查询响应。
type ListResp struct {
	Items []model.Role
	Total int64
}

// CreateReq 新增角色请求。
type CreateReq struct {
	Name   string `form:"name"`
	Remark string `form:"remark"`
}

// Validate 校验新增角色请求。
func (r *CreateReq) Validate() []FieldError {
	var errs []FieldError
	if msg := validateRoleName(r.Name); msg != "" {
		errs = append(errs, FieldError{Field: "name", Message: msg})
	}
	if msg := validateRemark(r.Remark); msg != "" {
		errs = append(errs, FieldError{Field: "remark", Message: msg})
	}
	return errs
}

// CreateResp 新增角色响应。
type CreateResp struct {
	ID int64
}

// UpdateReq 编辑角色请求。
type UpdateReq struct {
	ID     int64  `form:"-"`
	Name   string `form:"name"`
	Remark string `form:"remark"`
}

// Validate 校验编辑角色请求。
func (r *UpdateReq) Validate() []FieldError {
	var errs []FieldError
	if r.ID <= 0 {
		errs = append(errs, FieldError{Field: "id", Message: "无效的角色 ID"})
	}
	if msg := validateRoleName(r.Name); msg != "" {
		errs = append(errs, FieldError{Field: "name", Message: msg})
	}
	if msg := validateRemark(r.Remark); msg != "" {
		errs = append(errs, FieldError{Field: "remark", Message: msg})
	}
	return errs
}

// DeleteReq 删除角色请求。
type DeleteReq struct {
	ID int64
}

// AssignPermsReq 角色权限分配表单。
type AssignPermsReq struct {
	PermCodes []string `form:"permCodes"`
}

// AssignPermsFormDataReq 权限分配页数据查询（菜单树 + 权限勾选）。
type AssignPermsFormDataReq struct {
	RoleID int64
}

// AssignPermsFormDataResp 权限分配页展示数据。
type AssignPermsFormDataResp struct {
	PermTree []*PermTreeNode
}

// PermTreeNode 权限分配页树节点（菜单层级 + 叶子为具体权限码）。
type PermTreeNode struct {
	ID                string
	ParentID          string
	Name              string
	Code              string
	Checked           bool
	Depth             int
	SortOrder         int
	NumericID         int64
	HasBranchChildren bool
	Children          []*PermTreeNode
}

// NewUpdateReqFromRole 从实体构造编辑回填请求。
func NewUpdateReqFromRole(r *model.Role) *UpdateReq {
	if r == nil {
		return &UpdateReq{}
	}
	return &UpdateReq{
		ID:     r.ID,
		Name:   r.Name,
		Remark: r.Remark,
	}
}

func validateRoleName(n string) string {
	n = strings.TrimSpace(n)
	if n == "" {
		return "角色名称不能为空"
	}
	if len([]rune(n)) > 64 {
		return "角色名称长度不能超过 64 个字符"
	}
	return ""
}

func validateRemark(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len([]rune(s)) > 255 {
		return "备注长度不能超过 255 个字符"
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return "备注不能包含控制字符"
		}
	}
	return ""
}
