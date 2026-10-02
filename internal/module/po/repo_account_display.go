// =============================================================================
// 文件: internal/module/po/repo_account_display.go
// 模块: PO 工作台
// 类型: repo
// 职责: 加载 account -> 展示名映射（姓名(账号) 或 账号）。
// =============================================================================

package po

import (
	"context"
	"workbench/internal/module/user"
)

func (r *Repo) loadAccountDisplayMap(ctx context.Context) (map[string]string, error) {
	if r == nil || r.db == nil {
		return map[string]string{}, nil
	}
	return user.NewRepo(r.db).FindAccountDisplayMap(ctx)
}
