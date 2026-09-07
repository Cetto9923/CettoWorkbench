// =============================================================================
// 文件: internal/module/po/repodone_sidebar.go
// 模块: PO 工作台
// 类型: action
// 职责: 已办相关的侧栏角标（count-only 查询，与 /done 列表筛选/分页解耦）。
// 依赖: 无（继承 repodone.go 的 buildFormalDoneScopeSQL / model.User）
// =============================================================================

package po

import (
	"context"
	"strings"
)

// CountRecentDone 仅返回当前账号正式已办动作的总数（侧栏角标用）。
// 与 CountDoneActions 的 Total 同义，但只取 All 字段，节省一次 7 个 CASE 聚合。
// 不带时间筛选；保持与 /done 默认页 (mode=formal, 全部时间) 计数一致。
func (r *Repo) CountRecentDone(ctx context.Context, account string) (int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return 0, nil
	}
	q := r.db.WithContext(ctx).Table("zt_action AS a").
		Where("a.actor = ?", account)
	scopeSQL, scopeArgs := buildFormalDoneScopeSQL()
	q = q.Where(scopeSQL, scopeArgs...)
	var n int64
	if err := q.Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}
