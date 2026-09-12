// =============================================================================
// 文件: internal/module/agileteam/repo_user.go
// 模块: 敏捷小组治理
// 类型: repository
// 职责: 对成员调整请求中的禅道账号做精确存在性校验。
// =============================================================================

package agileteam

import (
	"context"
	"strings"
)

// ExistingUserAccounts 返回仍有效的禅道用户账号集合。
// 调整单只能引用 zt_user 中未删除的真实账号，避免确认后向 zt_team 写入伪造账号。
func (r *Repo) ExistingUserAccounts(ctx context.Context, accounts []string) (map[string]bool, error) {
	out := map[string]bool{}
	uniq := make([]string, 0, len(accounts))
	seen := map[string]bool{}
	for _, raw := range accounts {
		account := strings.TrimSpace(raw)
		if account == "" || seen[account] {
			continue
		}
		seen[account] = true
		uniq = append(uniq, account)
	}
	if len(uniq) == 0 {
		return out, nil
	}
	type row struct {
		Account string `gorm:"column:account"`
	}
	var rows []row
	if err := r.read().WithContext(ctx).
		Table("zt_user").
		Select("account").
		Where("account IN ? AND deleted = '0'", uniq).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, item := range rows {
		account := strings.TrimSpace(item.Account)
		if account != "" {
			out[account] = true
		}
	}
	return out, nil
}
