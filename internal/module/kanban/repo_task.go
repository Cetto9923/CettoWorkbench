// =============================================================================
// 文件: internal/module/kanban/repo_task.go
// 模块: 工作看板
// 类型: readonly
// 职责: 任务看板只读查询（指派未开始/进行中 + 完成者已完成）。
// 依赖: 无
// =============================================================================

package kanban

import (
	"context"
	"strings"
	"time"
)

const kanbanTaskLimit = 200

// FindKanbanTasks 查询选中账号的看板任务。
// wait/doing：assignedTo=account；done：finishedBy=account；不含 pause。
func (r *Repo) FindKanbanTasks(ctx context.Context, account string) ([]taskRow, error) {
	account = strings.TrimSpace(account)
	if account == "" || r == nil || r.db == nil {
		return []taskRow{}, nil
	}

	type rawRow struct {
		ID         int64      `gorm:"column:id"`
		Name       string     `gorm:"column:name"`
		Type       string     `gorm:"column:type"`
		Status     string     `gorm:"column:status"`
		StoryID    int64      `gorm:"column:story"`
		AssignedTo string     `gorm:"column:assignedTo"`
		FinishedBy string     `gorm:"column:finishedBy"`
		Deadline   *time.Time `gorm:"column:deadline"`
		StoryTitle string     `gorm:"column:storyTitle"`
	}

	var rows []rawRow
	err := r.db.WithContext(ctx).
		Table("zt_task AS t").
		Select(`t.id, t.name, t.type, t.status, t.story, t.assignedTo, t.finishedBy, t.deadline,
			IFNULL(s.title, '') AS storyTitle`).
		Joins("LEFT JOIN zt_story AS s ON s.id = t.story AND s.deleted = ?", "0").
		Where("t.deleted = ?", "0").
		Where(`(
			(t.assignedTo = ? AND t.status IN (?, ?))
			OR (t.finishedBy = ? AND t.status = ?)
		)`, account, "wait", "doing", account, "done").
		Order("t.id DESC").
		Limit(kanbanTaskLimit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []taskRow{}, nil
	}

	out := make([]taskRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, taskRow{
			ID:         row.ID,
			Name:       row.Name,
			Type:       row.Type,
			Status:     row.Status,
			StoryID:    row.StoryID,
			AssignedTo: row.AssignedTo,
			FinishedBy: row.FinishedBy,
			Deadline:   row.Deadline,
			StoryTitle: row.StoryTitle,
		})
	}
	return out, nil
}
