// =============================================================================
// 文件: internal/module/po/repo_account_display.go
// 模块: PO 工作台
// 类型: repo
// 职责: 加载 account -> 展示名映射（姓名(账号) 或 账号）。
// =============================================================================

package po

import (
	"context"
	"strings"

	"workbench/internal/pkg/personlabel"
)

// loadAccountDisplayMap 加载 account → 展示名映射（zhentao 兼容）。
func (r *Repo) loadAccountDisplayMap(ctx context.Context) (map[string]string, error) {
	if r == nil || r.db == nil {
		return make(map[string]string), nil
	}
	type row struct {
		Account  string `gorm:"column:account"`
		Realname string `gorm:"column:realname"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).Table("zt_user").
		Where("deleted = ?", "0").
		Select("account, realname").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Account) == "" {
			continue
		}
		if strings.TrimSpace(row.Realname) == "" {
			out[row.Account] = row.Account
		} else {
			out[row.Account] = personlabel.Format(row.Account, row.Realname)
		}
	}
	return out, nil
}
