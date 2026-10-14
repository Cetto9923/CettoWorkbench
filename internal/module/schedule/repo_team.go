// =============================================================================
// 文件: internal/module/schedule/repo_team.go
// 模块: 排期工作台
// 类型: action
// 职责: 敏捷小组成员只读查询。
// 依赖: 无
// =============================================================================

package schedule

import (
	"context"
	"strings"
)

// ListTeamgroupMemberAccounts 查询敏捷小组（zt_team type=teamgroup）成员账号，按 order 升序。
func (r *Repo) ListTeamgroupMemberAccounts(ctx context.Context, teamgroupID uint) ([]string, error) {
	if teamgroupID == 0 {
		return []string{}, nil
	}
	var rows []struct {
		Account string `gorm:"column:account"`
	}
	err := r.db.WithContext(ctx).
		Table("zt_team").
		Select("account").
		Where("root = ? AND type = ?", teamgroupID, "teamgroup").
		Order("`order` ASC, id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		account := strings.TrimSpace(row.Account)
		if account == "" {
			continue
		}
		if _, ok := seen[account]; ok {
			continue
		}
		seen[account] = struct{}{}
		out = append(out, account)
	}
	return out, nil
}
