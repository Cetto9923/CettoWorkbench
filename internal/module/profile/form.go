// =============================================================================
// 文件: internal/module/profile/form.go
// 模块: 个人资料
// 类型: action
// 职责: 定义当前登录用户自助资料读写的 Req/Resp 与校验。
//       本轮实现基本信息编辑 + 默认小组 + 改密；自选视图（preferredRoles）等高级
//       功能暂不实现，留给 workbenchroles 包后续接入。
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

// AgileGroupOption 本人已加入的敏捷小组。
type AgileGroupOption struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

// GetResp 当前用户个人资料。
type GetResp struct {
	Account     string             `json:"account"`
	DisplayName string             `json:"displayName"`
	Email       string             `json:"email"`
	Mobile      string             `json:"mobile"`
	Gender      string             `json:"gender"` // "m" / "f"
	DeptID      uint64             `json:"deptId"`
	DeptName    string             `json:"deptName"`
	AgileGroups []AgileGroupOption `json:"agileGroups"`
	MainTeamID  uint64             `json:"mainTeamId"` // 禅道 zt_user.mainTeam
}

// UpdateReq 更新个人资料。
//
// 注：性别字段在 ZenTao schema 中是 enum('f','m')；前端"未设置"选项通过清空
// request body 中 Gender 字段并在 Service 层跳过更新处理（保持库内原值）。
type UpdateReq struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Mobile      string `json:"mobile"`
	Gender      string `json:"gender"` // "m" / "f" / "" (空 = 不更新)
	MainTeamID  uint64 `json:"mainTeamId"`
}

var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Validate 校验资料更新请求。
// gender == "" 表示前端未选择（前端 radio 选项包含"未设置"），此时跳过性别校验，
// Service 端用 "skip gender" 语义，避免向 enum 字段写空字符串触发 GORM 错误。
func (r *UpdateReq) Validate() []FieldError {
	r.Email = strings.TrimSpace(r.Email)
	r.Mobile = strings.TrimSpace(r.Mobile)
	r.DisplayName = strings.TrimSpace(r.DisplayName)
	r.Gender = strings.TrimSpace(r.Gender)

	var errs []FieldError
	if r.DisplayName == "" {
		errs = append(errs, FieldError{Field: "displayName", Message: "姓名不能为空"})
	} else if len([]rune(r.DisplayName)) > 100 {
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
		errs = append(errs, FieldError{Field: "gender", Message: "性别必须是 m 或 f"})
	}
	return errs
}

// ApplyGenderSkip 报告 gender 字段是否应该跳过更新（值为空表示"未设置"）。
func (r *UpdateReq) ApplyGenderSkip() bool {
	return r.Gender == ""
}

// ChangePasswordReq 修改本人登录密码。
type ChangePasswordReq struct {
	OldPassword     string `json:"oldPassword"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
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
