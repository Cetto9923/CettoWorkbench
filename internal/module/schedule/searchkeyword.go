// =============================================================================
// 文件: internal/module/schedule/searchkeyword.go
// 模块: 排期工作台
// 类型: action
// 职责: 解析排期页面搜索框中的展示编号，兼容 REQ/RD/SUB 前缀转真实 ID。
// 依赖: 无
// =============================================================================

package schedule

import (
	"regexp"
	"strings"
)

var scheduleDisplayIDPattern = regexp.MustCompile(`(?i)^(?:REQ|SUB|RD)\s*-?\s*(\d+)$`)
var schedulePlainIDPattern = regexp.MustCompile(`^\d+$`)

func extractScheduleSearchID(keyword string) string {
	trimmed := strings.TrimSpace(keyword)
	if trimmed == "" {
		return ""
	}
	if schedulePlainIDPattern.MatchString(trimmed) {
		return trimmed
	}
	matches := scheduleDisplayIDPattern.FindStringSubmatch(trimmed)
	if len(matches) != 2 {
		return ""
	}
	return matches[1]
}
