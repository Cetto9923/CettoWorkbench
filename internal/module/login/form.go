// =============================================================================
// 文件: internal/module/login/form.go
// 模块: 登录
// 类型: action
// 职责: 定义认证页面表单回填、登录请求结构与校验。
// 依赖: 无
// =============================================================================

package login

import "strings"

// FieldError 表示表单字段错误；表单级错误使用 "_form"。
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// LoginForm 登录页数据回填。
type LoginForm struct {
	Account string
}

// ShowLoginReq 登录成功后重定向地址。
type ShowLoginReq struct {
	Redirect string `form:"redirect"`
}

// LoginReq 登录请求（Handler 绑定 POST 字段；Redirect 优先来自 query，其次隐藏域；IP/UserAgent 由 Handler 注入）。
type LoginReq struct {
	Account   string `form:"account"`
	Password  string `form:"password"`
	Redirect  string `form:"redirect"`
	IP        string // 由 Handler 注入，不来自表单。
	UserAgent string // 由 Handler 注入，不来自表单。
}

// normalizeLoginAccount 规范化登录账号（与 DB 列 account 一致）。
func (r *LoginReq) normalizeLoginAccount() {
	r.Account = strings.TrimSpace(r.Account)
}

// Validate 校验登录必填字段，无错误返回 nil。
func (r *LoginReq) Validate() []FieldError {
	r.normalizeLoginAccount()
	if r.Account == "" || r.Password == "" {
		return []FieldError{{Field: "_form", Message: "账号和密码不能为空"}}
	}
	return nil
}
