// =============================================================================
// 文件: internal/module/schedule/repo_scheduling_date.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期写入用的日期空值归一化与解析。
// 依赖: （无）
// =============================================================================

package schedule

import (
	"strings"
	"time"
)

func nullableSchedulingDate(raw string) interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return raw
}

func parseSchedulingDatePtr(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	t, err := time.ParseInLocation("2006-01-02", raw, time.Local)
	if err != nil {
		return nil
	}
	return &t
}
