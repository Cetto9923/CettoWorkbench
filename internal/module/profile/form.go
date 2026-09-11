// =============================================================================
// 文件: internal/module/profile/form.go
// 模块: 个人资料
// 类型: action
// 职责: 定义当前登录用户自助资料读写的 Req/Resp 与校验。
// 依赖: 无
// =============================================================================

package profile

import (
	"regexp"
	"strings"
	"unicode"
)

// FieldError 字段级验证错误；表单级错误使用 "_form"。
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// RoleOption 本人可自选的工作台角色。
type RoleOption struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// AgileGroupOption 本人已加入的敏捷小组。
type AgileGroupOption struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

// GetResp 当前用户个人资料。
type GetResp struct {
	Account        string             `json:"account"`
	DisplayName    string             `json:"displayName"`
	Email          string             `json:"email"`
	Mobile         string             `json:"mobile"`
	Gender         string             `json:"gender"` // "m" / "f" / ""
	DeptID         uint64             `json:"deptId"`
	DeptName       string             `json:"deptName"`
	AllowedRoles   []RoleOption       `json:"allowedRoles"`   // 可自选工作台角色（已排除 lead/pmo 等组织固定角色）
	PreferredRoles []string           `json:"preferredRoles"` // 已勾选自选角色
	AgileGroups    []AgileGroupOption `json:"agileGroups"`
	MainTeamID     uint64             `json:"mainTeamId"` // 禅道 zt_user.mainTeam
}

// UpdateReq 更新个人资料。
type UpdateReq struct {
	DisplayName    string   `json:"displayName"`
	Email          string   `json:"email"`
	Mobile         string   `json:"mobile"`
	Gender         string   `json:"gender"` // "m" / "f" / ""
	PreferredRoles []string `json:"preferredRoles"`
	MainTeamID     uint64   `json:"mainTeamId"`
}

// UpdateResp 更新结果。
type UpdateResp struct {
	ID             int64    `json:"id"`
	PreferredRoles []string `json:"preferredRoles"`
	MainTeamID     uint64   `json:"mainTeamId"`
}

// ChangePasswordReq 修改本人登录密码。
type ChangePasswordReq struct {
	OldPassword     string `json:"oldPassword"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
}

var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+\\-]+@[a-zA-Z0-9.\\-]+\\.[a-zA-Z]{2,}$`)

// Validate 校验资料更新请求。
func (r *UpdateReq) Validate() []FieldError {
	r.Email = strings.TrimSpace(r.Email)
	r.Mobile = strings.TrimSpace(r.Mobile)
	r.DisplayName = strings.TrimSpace(r.DisplayName)
	r.Gender = strings.TrimSpace(r.Gender)

	var errs []FieldError
	if r.DisplayName != "" && len([]rune(r.DisplayName)) > 100 {
		errs = append(errs, FieldError{Field: "displayName", Message: "姓名长度不能超过 100 个字符"})
	}
	if r.Email != "" {
		if len(r.Email) > 90 {
			errs = append(errs, FieldError{Field: "email", Message: "邮箱长度不能超过 90 个字符"})
		} else if !emailPattern.MatchString(r.Email) {
			errs = append(errs, FieldError{Field: "email", Message: "邮箱格式不正确"})
		}
	}
	if r.Mobile != "" && len(r.Mobile) > 11 {
		errs = append(errs, FieldError{Field: "mobile", Message: "手机号长度不能超过 11 个字符"})
	}
	if r.Gender != "" && r.Gender != "m" && r.Gender != "f" {
		errs = append(errs, FieldError{Field: "gender", Message: "请选择有效性别（男/女）"})
	}

	normalized := make([]string, 0, len(r.PreferredRoles))
	seen := map[string]bool{}
	for _, raw := range r.PreferredRoles {
		key := strings.ToLower(strings.TrimSpace(raw))
		if key == "" || seen[key] {
			continue
		}
		if key == "lead" || key == "pmo" {
			errs = append(errs, FieldError{Field: "preferredRoles", Message: "团队管理 / PMO 视图须由组织统一配置，不可自行勾选"})
			continue
		}
		seen[key] = true
		normalized = append(normalized, key)
	}
	r.PreferredRoles = normalized
	return errs
}

// ApplyGenderSkip 报告 gender 字段是否应该跳过更新（值为空表示"未设置"）。
func (r *UpdateReq) ApplyGenderSkip() bool {
	return r.Gender == ""
}

// Validate 校验改密请求。
func (r *ChangePasswordReq) Validate() []FieldError {
	var errs []FieldError
	if strings.TrimSpace(r.OldPassword) == "" {
		errs = append(errs, FieldError{Field: "oldPassword", Message: "请输入当前密码"})
	}
	if msg := validatePassword(r.NewPassword); msg != "" {
		errs = append(errs, FieldError{Field: "newPassword", Message: msg})
	}
	if strings.TrimSpace(r.ConfirmPassword) == "" {
		errs = append(errs, FieldError{Field: "confirmPassword", Message: "请再次输入新密码"})
	} else if r.NewPassword != r.ConfirmPassword {
		errs = append(errs, FieldError{Field: "confirmPassword", Message: "两次输入的新密码不一致"})
	}
	return errs
}

func validatePassword(p string) string {
	if p == "" {
		return "新密码不能为空"
	}
	if len(p) < 8 || len(p) > 64 {
		return "新密码长度应为 8-64 个字符"
	}
	hasLetter, hasDigit := false, false
	for _, ch := range p {
		if unicode.IsLetter(ch) {
			hasLetter = true
		}
		if unicode.IsDigit(ch) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return "新密码须同时包含字母和数字"
	}
	return ""
}
