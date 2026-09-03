// =============================================================================
// 文件: internal/module/schedule/repo_scheduling.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期一体化弹窗跨文件共用 helper:account 去重 / 禅道日期格式化。
//       本文件只放 helper,具体只读查询见 repo_scheduling_detail.go 与 repo_scheduling_item.go。
// 依赖: (无项目内部包)
// =============================================================================

package schedule

import (
	"strings"
)

func collectNonEmptyAccounts(accounts ...string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(accounts))
	for _, account := range accounts {
		account = strings.TrimSpace(account)
		if account == "" {
			continue
		}
		if _, exists := seen[account]; exists {
			continue
		}
		seen[account] = struct{}{}
		out = append(out, account)
	}
	return out
}

func formatZenTaoDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "0000-00-00") {
		return ""
	}
	if len(raw) >= 10 {
		return raw[:10]
	}
	return raw
}
