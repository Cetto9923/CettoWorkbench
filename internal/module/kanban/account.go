// =============================================================================
// 文件: internal/module/kanban/account.go
// 模块: 工作看板
// 类型: action
// 职责: 解析看板负责人筛选（单账号 / 当前敏捷小组全部）。
// 依赖: internal/pkg/errorx
// =============================================================================

package kanban

import (
	"strings"

	"workbench/internal/pkg/errorx"
)

// AccountFilterAll 负责人行「全部」tab 的固定值。
const AccountFilterAll = "all"

// resolveKanbanAccounts 解析负责人筛选。
// account=all 时返回指定敏捷小组全员（去重）；否则返回单账号。
func resolveKanbanAccounts(actorAccount, reqAccount string, teamgroupID uint, groups []TeamgroupItem) ([]string, error) {
	actorAccount = strings.TrimSpace(actorAccount)
	acc := strings.TrimSpace(reqAccount)
	if acc == "" {
		acc = actorAccount
	}
	if acc == "" {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "账号不能为空")
	}

	if acc == AccountFilterAll {
		if teamgroupID == 0 {
			return nil, errorx.New(errorx.ErrCodeInvalidParam, "请选择敏捷小组")
		}
		var found *TeamgroupItem
		for i := range groups {
			if groups[i].ID == teamgroupID {
				found = &groups[i]
				break
			}
		}
		if found == nil {
			return nil, errorx.New(errorx.ErrCodeForbidden, "无权查看该敏捷小组")
		}
		out := make([]string, 0, len(found.Members))
		seen := make(map[string]struct{}, len(found.Members))
		for _, m := range found.Members {
			a := strings.TrimSpace(m.Account)
			if a == "" {
				continue
			}
			if _, ok := seen[a]; ok {
				continue
			}
			seen[a] = struct{}{}
			out = append(out, a)
		}
		return out, nil
	}

	allowed := collectMemberAccounts(groups)
	if acc == actorAccount {
		return []string{acc}, nil
	}
	if _, ok := allowed[acc]; !ok {
		return nil, errorx.New(errorx.ErrCodeForbidden, "无权查看该成员")
	}
	return []string{acc}, nil
}

func collectMemberAccounts(groups []TeamgroupItem) map[string]struct{} {
	out := make(map[string]struct{})
	for _, g := range groups {
		for _, m := range g.Members {
			acc := strings.TrimSpace(m.Account)
			if acc == "" {
				continue
			}
			out[acc] = struct{}{}
		}
	}
	return out
}
