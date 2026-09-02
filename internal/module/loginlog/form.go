// =============================================================================
// 文件: internal/module/loginlog/form.go
// 模块: 登录日志
// 类型: readonly
// 职责: 定义登录日志列表查询请求与校验逻辑。
// 依赖: 无
// =============================================================================

package loginlog

import (
	"errors"
	"strings"
	"time"
)

const (
	timeLayoutDate     = "2006-01-02"
	timeLayoutDateTime = "2006-01-02 15:04:05"
	timeLayoutLocal    = "2006-01-02T15:04"
)

// FieldError 字段级验证错误。
type FieldError struct {
	Field   string
	Message string
}

// ListReq 登录日志列表查询请求。
type ListReq struct {
	Account  string `form:"account"`
	IP       string `form:"ip"`
	StartAt  string `form:"startAt"`
	EndAt    string `form:"endAt"`
	Page     int    `form:"page"`
	PageSize int    `form:"-"`
}

// Normalize 规范化查询参数。
func (r *ListReq) Normalize() {
	r.Account = strings.TrimSpace(r.Account)
	r.IP = strings.TrimSpace(r.IP)
	r.StartAt = strings.TrimSpace(r.StartAt)
	r.EndAt = strings.TrimSpace(r.EndAt)
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 {
		r.PageSize = 20
	}
}

// Validate 校验列表查询参数。
func (r *ListReq) Validate() []FieldError {
	var errs []FieldError
	start, startErr := parseStartTime(r.StartAt)
	if startErr != nil {
		errs = append(errs, FieldError{Field: "startAt", Message: "开始时间格式不正确"})
	}
	end, endErr := parseEndTime(r.EndAt)
	if endErr != nil {
		errs = append(errs, FieldError{Field: "endAt", Message: "结束时间格式不正确"})
	}
	if startErr == nil && endErr == nil && start != nil && end != nil && start.After(*end) {
		errs = append(errs, FieldError{Field: "endAt", Message: "结束时间不能早于开始时间"})
	}
	return errs
}

// ParsedStartAt 返回解析后的开始时间。
func (r *ListReq) ParsedStartAt() (*time.Time, error) {
	return parseStartTime(r.StartAt)
}

// ParsedEndAt 返回解析后的结束时间。
func (r *ListReq) ParsedEndAt() (*time.Time, error) {
	return parseEndTime(r.EndAt)
}

func parseStartTime(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	layouts := []string{timeLayoutDateTime, timeLayoutLocal, timeLayoutDate}
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, raw, time.Local)
		if err == nil {
			return &t, nil
		}
	}
	return nil, errors.New("invalid time format")
}

func parseEndTime(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	layouts := []string{timeLayoutDateTime, timeLayoutLocal, timeLayoutDate}
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, raw, time.Local)
		if err == nil {
			if layout == timeLayoutDate {
				end := t.Add(24*time.Hour - time.Second)
				return &end, nil
			}
			return &t, nil
		}
	}
	return nil, errors.New("invalid time format")
}
