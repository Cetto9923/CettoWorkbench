// =============================================================================
// 文件: internal/module/menu/form.go
// 模块: 菜单管理
// 类型: crud
// 职责: 定义菜单模块请求/响应结构体与校验逻辑。
// 依赖: internal/model
// =============================================================================

package menu

import (
	"strings"

	"workbench/internal/model"
)

// FieldError 表示字段级验证错误。
type FieldError struct {
	Field   string
	Message string
}

// ListReq 菜单列表请求。
type ListReq struct {
	Page     int `form:"page"`
	PageSize int `form:"pageSize"`
}

// ListResp 菜单列表响应。
type ListResp struct {
	Items []model.Menu
	Total int64
}

// CreateReq 新增菜单请求。
type CreateReq struct {
	ParentID uint64 `form:"parentId"`
	Type     string `form:"menuType"`
	Title    string `form:"title"`
	Icon     string `form:"icon"`
	Path     string `form:"path"`
	Perm     string `form:"perm"`
	Sort     int    `form:"sort"`
}

// Validate 校验新增菜单请求。
func (r *CreateReq) Validate() []FieldError {
	errs := make([]FieldError, 0)
	if strings.TrimSpace(r.Title) == "" {
		errs = append(errs, FieldError{Field: "title", Message: "菜单标题不能为空"})
	}
	if len([]rune(strings.TrimSpace(r.Title))) > 64 {
		errs = append(errs, FieldError{Field: "title", Message: "菜单标题长度不能超过 64"})
	}
	if len([]rune(strings.TrimSpace(r.Icon))) > 64 {
		errs = append(errs, FieldError{Field: "icon", Message: "图标长度不能超过 64"})
	}
	if len([]rune(strings.TrimSpace(r.Path))) > 255 {
		errs = append(errs, FieldError{Field: "path", Message: "路径长度不能超过 255"})
	}
	if len([]rune(strings.TrimSpace(r.Perm))) > 64 {
		errs = append(errs, FieldError{Field: "perm", Message: "权限标识长度不能超过 64"})
	}
	return errs
}

// CreateResp 新增菜单响应。
type CreateResp struct {
	ID uint64
}

// UpdateReq 编辑菜单请求。
type UpdateReq struct {
	ID       uint64 `form:"-"`
	ParentID uint64 `form:"parentId"`
	Type     string `form:"menuType"`
	Title    string `form:"title"`
	Icon     string `form:"icon"`
	Path     string `form:"path"`
	Perm     string `form:"perm"`
	Sort     int    `form:"sort"`
}

// Validate 校验编辑菜单请求。
func (r *UpdateReq) Validate() []FieldError {
	errs := make([]FieldError, 0)
	if r.ID == 0 {
		errs = append(errs, FieldError{Field: "id", Message: "无效的菜单 ID"})
	}
	if strings.TrimSpace(r.Title) == "" {
		errs = append(errs, FieldError{Field: "title", Message: "菜单标题不能为空"})
	}
	if len([]rune(strings.TrimSpace(r.Title))) > 64 {
		errs = append(errs, FieldError{Field: "title", Message: "菜单标题长度不能超过 64"})
	}
	if len([]rune(strings.TrimSpace(r.Icon))) > 64 {
		errs = append(errs, FieldError{Field: "icon", Message: "图标长度不能超过 64"})
	}
	if len([]rune(strings.TrimSpace(r.Path))) > 255 {
		errs = append(errs, FieldError{Field: "path", Message: "路径长度不能超过 255"})
	}
	if len([]rune(strings.TrimSpace(r.Perm))) > 64 {
		errs = append(errs, FieldError{Field: "perm", Message: "权限标识长度不能超过 64"})
	}
	return errs
}

// UpdateResp 编辑菜单响应。
type UpdateResp struct {
	ID uint64
}

// DeleteReq 删除菜单请求。
type DeleteReq struct {
	ID uint64
}

// RepoFindAllReq 仓储层菜单列表请求。
type RepoFindAllReq struct {
	Page     int
	PageSize int
}

// NewUpdateReqFromModel 从实体构造编辑请求。
func NewUpdateReqFromModel(m *model.Menu) *UpdateReq {
	if m == nil {
		return &UpdateReq{}
	}
	return &UpdateReq{
		ID:       m.ID,
		ParentID: m.ParentID,
		Type:     m.Type,
		Title:    m.Title,
		Icon:     m.Icon,
		Path:     m.Path,
		Perm:     m.Perm,
		Sort:     m.Sort,
	}
}
