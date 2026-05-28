// =============================================================================
// 文件: internal/module/user/form.go
// 模块: 用户管理
// 类型: crud
// 职责: 定义 user 模块所有表单/请求/响应结构体和验证逻辑
// 依赖: internal/model
// =============================================================================

package user

import (
	"regexp"
	"strings"
	"unicode"

	"workbench/internal/model"
)

// accountPattern / emailPattern 目前仅 user 模块使用，保留在本文件中。
// 若其他模块需要相同规则，统一迁移至 internal/pkg/validate/patterns.go。
var (
	accountPattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	emailPattern   = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// FieldError 字段级验证错误。
type FieldError struct {
	Field   string
	Message string
}

// ListReq 用户列表查询请求。
type ListReq struct {
	Account     string `form:"account"`     // 按账号模糊搜索
	Email       string `form:"email"`       // 按邮箱模糊搜索
	DisplayName string `form:"displayName"` // 按姓名模糊搜索
	Status      string `form:"status"`      // "" 表示全部，"active"/"inactive" 表示过滤状态
	Page        int    `form:"page"`
	PageSize    int    `form:"-"` // 不接受前端传入
}

// ListResp 用户列表查询响应。
type ListResp struct {
	Items []model.User
	Total int64
}

// CreateReq 新增用户请求。
type CreateReq struct {
	Account         string  `form:"account" json:"account"`
	Email           string  `form:"email" json:"email"`
	DisplayName     string  `form:"displayName" json:"displayName"`
	Gender          string  `form:"gender" json:"gender"`
	Password        string  `form:"password" json:"password"`
	PasswordConfirm string  `form:"passwordConfirm" json:"passwordConfirm"`
	IsActive        bool    `form:"isActive" json:"isActive"`
	DeptID          uint64  `form:"deptID" json:"deptID"`
	RoleIDs         []int64 `form:"roleIds" json:"roleIds"`
}

// CreateResp 新增用户响应。
type CreateResp struct {
	ID int64
}

// BatchCreateReq 批量新增用户请求。
type BatchCreateReq struct {
	Users []CreateReq `form:"users" json:"users"`
}

// BatchFieldError 批量新增的行级字段错误。
type BatchFieldError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

// BatchCreateResp 批量新增用户响应。
type BatchCreateResp struct {
	Count int `json:"count"`
}

// ExportReq 用户列表导出请求。
type ExportReq struct {
	FileName string `form:"fileName"`
	FileType string `form:"fileType"`
	Scope    string `form:"scope"`
}

// UpdateReq 编辑用户请求。
type UpdateReq struct {
	ID int64 `form:"-"` // 从路径参数注入
	// Account 账号创建后不可修改。前端 readonly 无法阻止恶意请求，
	// 故在绑定层即忽略，Service 从数据库加载原值。
	Email           string  `form:"email"`
	DisplayName     string  `form:"displayName"`
	Gender          string  `form:"gender"`
	Password        string  `form:"password"`
	PasswordConfirm string  `form:"passwordConfirm"`
	IsActive        bool    `form:"isActive"`
	DeptID          uint64  `form:"deptID"`
	RoleIDs         []int64 `form:"roleIds"`
}

// UpdateResp 编辑用户响应。
type UpdateResp struct {
	ID int64
}

// DeleteReq 删除用户请求。
type DeleteReq struct {
	ID int64
}

// ToggleStatusReq 切换用户启用状态请求。
type ToggleStatusReq struct {
	ID int64 `form:"id" uri:"id"`
}

// ResetPasswordReq 重置密码请求。
type ResetPasswordReq struct {
	ID              uint64 `form:"id" uri:"id"`
	NewPassword     string `form:"newPassword"`
	ConfirmPassword string `form:"confirmPassword"`
}

// Normalize 规范化分页参数、状态值与筛选字段。
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
	if r.Status != "active" && r.Status != "inactive" {
		r.Status = ""
	}
	r.Account = strings.TrimSpace(r.Account)
	r.Email = strings.TrimSpace(r.Email)
	r.DisplayName = strings.TrimSpace(r.DisplayName)
}

// IsActiveFilter 返回 Repo 可使用的激活状态过滤值。
// nil 表示不过滤；true 表示仅激活；false 表示仅未激活。
func (r *ListReq) IsActiveFilter() *bool {
	switch r.Status {
	case "active":
		v := true
		return &v
	case "inactive":
		v := false
		return &v
	default:
		return nil
	}
}

// Validate 校验新增用户请求参数。
func (r *CreateReq) Validate() []FieldError {
	var errs []FieldError

	if err := validateAccount(r.Account); err != "" {
		errs = append(errs, FieldError{Field: "account", Message: err})
	}

	if strings.TrimSpace(r.Email) != "" {
		if err := validateEmail(r.Email); err != "" {
			errs = append(errs, FieldError{Field: "email", Message: err})
		}
	}

	if err := validateDisplayName(r.DisplayName); err != "" {
		errs = append(errs, FieldError{Field: "displayName", Message: err})
	}
	if err := validateGender(r.Gender); err != "" {
		errs = append(errs, FieldError{Field: "gender", Message: err})
	}

	if err := validatePassword(r.Password); err != "" {
		errs = append(errs, FieldError{Field: "password", Message: err})
	}
	if r.PasswordConfirm == "" {
		errs = append(errs, FieldError{Field: "passwordConfirm", Message: "请再次输入密码"})
	} else if r.Password != r.PasswordConfirm {
		errs = append(errs, FieldError{Field: "passwordConfirm", Message: "两次输入的密码不一致"})
	}
	if len(r.RoleIDs) == 0 {
		errs = append(errs, FieldError{Field: "roleIds", Message: "请至少选择一个角色"})
	}

	return errs
}

// Validate 校验编辑用户请求参数。
func (r *UpdateReq) Validate() []FieldError {
	var errs []FieldError

	if strings.TrimSpace(r.Email) != "" {
		if err := validateEmail(r.Email); err != "" {
			errs = append(errs, FieldError{Field: "email", Message: err})
		}
	}

	if err := validateDisplayName(r.DisplayName); err != "" {
		errs = append(errs, FieldError{Field: "displayName", Message: err})
	}
	if err := validateGender(r.Gender); err != "" {
		errs = append(errs, FieldError{Field: "gender", Message: err})
	}
	// 编辑场景下密码允许留空；填写其一时要求两项同时合法且一致。
	if strings.TrimSpace(r.Password) != "" || strings.TrimSpace(r.PasswordConfirm) != "" {
		if err := validatePassword(r.Password); err != "" {
			errs = append(errs, FieldError{Field: "password", Message: err})
		}
		if strings.TrimSpace(r.PasswordConfirm) == "" {
			errs = append(errs, FieldError{Field: "passwordConfirm", Message: "请再次输入密码"})
		} else if r.Password != r.PasswordConfirm {
			errs = append(errs, FieldError{Field: "passwordConfirm", Message: "两次输入的密码不一致"})
		}
	}

	return errs
}

// NewUpdateReqFromUser 从实体构造编辑回填请求。
func NewUpdateReqFromUser(u *model.User) *UpdateReq {
	if u == nil {
		return &UpdateReq{}
	}
	return &UpdateReq{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Gender:      u.Gender,
		IsActive:    u.IsActive,
		DeptID:      u.DeptID,
		RoleIDs:     []int64{},
	}
}

// Validate 校验重置密码请求参数。
func (r *ResetPasswordReq) Validate() []FieldError {
	var errs []FieldError

	if strings.TrimSpace(r.NewPassword) == "" {
		errs = append(errs, FieldError{Field: "newPassword", Message: "新密码不能为空"})
	} else if len(r.NewPassword) < 8 {
		errs = append(errs, FieldError{Field: "newPassword", Message: "新密码长度不能小于 8 位"})
	}

	if strings.TrimSpace(r.ConfirmPassword) == "" {
		errs = append(errs, FieldError{Field: "confirmPassword", Message: "请再次输入新密码"})
	} else if r.NewPassword != r.ConfirmPassword {
		errs = append(errs, FieldError{Field: "confirmPassword", Message: "两次输入的密码不一致"})
	}

	return errs
}

// Validate 校验用户导出请求。
func (r *ExportReq) Validate() []FieldError {
	var errs []FieldError

	r.FileName = strings.TrimSpace(r.FileName)
	r.FileType = strings.ToLower(strings.TrimSpace(r.FileType))
	r.Scope = strings.ToLower(strings.TrimSpace(r.Scope))

	if r.FileName == "" {
		errs = append(errs, FieldError{Field: "fileName", Message: "文件名不能为空"})
	}
	if len([]rune(r.FileName)) > 128 {
		errs = append(errs, FieldError{Field: "fileName", Message: "文件名长度不能超过 128 个字符"})
	}

	switch r.FileType {
	case "xlsx", "xls", "csv", "xml", "html":
	default:
		errs = append(errs, FieldError{Field: "fileType", Message: "文件类型不支持"})
	}

	if r.Scope != "all" {
		errs = append(errs, FieldError{Field: "scope", Message: "导出范围不支持"})
	}

	return errs
}

// validateAccount 校验账号。
// 规则：3-32 字符，字母数字下划线。
func validateAccount(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return "用户名不能为空"
	}
	if len(u) < 3 || len(u) > 32 {
		return "用户名长度应为 3-32 个字符"
	}
	if !accountPattern.MatchString(u) {
		return "用户名只能包含字母、数字和下划线"
	}
	return ""
}

// validateEmail 校验邮箱格式（仅调用方已确认非空时使用）。
func validateEmail(e string) string {
	e = strings.TrimSpace(e)
	if len(e) > 128 {
		return "邮箱长度不能超过 128 个字符"
	}
	if !emailPattern.MatchString(e) {
		return "邮箱格式不正确"
	}
	return ""
}

// validateDisplayName 校验显示名。
func validateDisplayName(n string) string {
	n = strings.TrimSpace(n)
	if n == "" {
		return "显示名不能为空"
	}
	if len([]rune(n)) > 32 {
		return "显示名长度不能超过 32 个字符"
	}
	return ""
}

// validateGender 校验性别。
// 规则：仅允许 m/f。
func validateGender(g string) string {
	g = strings.TrimSpace(g)
	if g == "" {
		return "请选择性别"
	}
	if g != "m" && g != "f" {
		return "性别取值不合法"
	}
	return ""
}

// validatePassword 校验密码强度。
// 规则：8-64 字符，至少包含字母和数字各一个。
// ⚠️ 前端 create.html 的 minlength 必须与此处下限 (8) 保持一致。
func validatePassword(p string) string {
	if p == "" {
		return "密码不能为空"
	}
	if len(p) < 8 || len(p) > 64 {
		return "密码长度应为 8-64 个字符"
	}

	var hasLetter bool
	var hasDigit bool
	for _, c := range p {
		switch {
		case unicode.IsLetter(c):
			hasLetter = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}

	if !hasLetter || !hasDigit {
		return "密码必须同时包含字母和数字"
	}
	return ""
}
