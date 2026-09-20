// =============================================================================
// 文件: internal/module/kanban/repo_count.go
// 模块: 工作看板
// 类型: action
// 职责: 按账号批量统计指派任务，供任务看板负责人行角标。
// 依赖: 无
// =============================================================================

package kanban

import (
	"context"
)

type accCountRow struct {
	Acc string `gorm:"column:acc"`
	Cnt int64  `gorm:"column:cnt"`
}

func scanAccCounts(rows []accCountRow) map[string]int64 {
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		if row.Acc == "" {
			continue
		}
		out[row.Acc] = row.Cnt
	}
	return out
}

// CountMemberTasks 任务看板数量：指派给该账号的未开始 / 进行中任务。
func (r *Repo) CountMemberTasks(ctx context.Context, accounts []string) (map[string]int64, error) {
	cleaned := cleanAccounts(accounts)
	if r == nil || r.db == nil || len(cleaned) == 0 {
		return map[string]int64{}, nil
	}
	var rows []accCountRow
	err := r.db.WithContext(ctx).
		Table("zt_task").
		Select("assignedTo AS acc, COUNT(*) AS cnt").
		Where("deleted = ?", "0").
		Where("status IN ?", []string{"wait", "doing"}).
		Where("assignedTo IN ?", cleaned).
		Group("assignedTo").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return scanAccCounts(rows), nil
}
