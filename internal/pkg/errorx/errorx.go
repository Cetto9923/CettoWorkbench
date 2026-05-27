// =============================================================================
// 文件: internal/pkg/errorx/errorx.go
// 模块: 基础设施
// 类型: infra
// 职责: 定义统一业务错误类型与工具函数。
// 依赖: 无
// =============================================================================

package errorx

import "errors"

// 通用业务错误码。
const (
	ErrCodeNotFound     = "notfound"
	ErrCodeForbidden    = "forbidden"
	ErrCodeInvalidParam = "invalidparam"
	ErrCodeInternal     = "internal"
	ErrCodeConflict     = "conflict"
)

// BizError 表示可识别的业务错误。
type BizError struct {
	Code  string
	Msg   string
	Cause error
}

// Error 返回固定格式错误信息：[Code] Msg。
func (e *BizError) Error() string {
	if e == nil {
		return ""
	}
	return "[" + e.Code + "] " + e.Msg
}

// Unwrap 返回底层错误，便于 errors.Is/errors.As 透传判断。
func (e *BizError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// New 创建一个无 Cause 的业务错误。
func New(code, msg string) *BizError {
	return &BizError{
		Code: code,
		Msg:  msg,
	}
}

// Wrap 创建一个带 Cause 的业务错误。
func Wrap(code, msg string, cause error) *BizError {
	return &BizError{
		Code:  code,
		Msg:   msg,
		Cause: cause,
	}
}

// IsBizError 判断 err 是否为 BizError（包含错误链中的包裹情况）。
func IsBizError(err error) (*BizError, bool) {
	if err == nil {
		return nil, false
	}
	var bizErr *BizError
	if errors.As(err, &bizErr) {
		return bizErr, true
	}
	return nil, false
}
