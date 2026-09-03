// =============================================================================
// 文件: internal/module/po/repoboardtask.go
// 模块: PO 工作台
// 类型: repo
// 职责: 任务看板只读查询。
// =============================================================================

package po

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"workbench/internal/pkg/zentao"
)

// FindBoardTaskList 查询当前敏捷小组中的真实任务，并按原型归入三列。
func (r *Repo) FindBoardTaskList(ctx context.Context, req BoardTaskReq, displayMap map[string]string) ([]*BoardTaskColumn, BoardTaskSummary, error) {
	columns := newBoardTaskColumns()
	if r == nil || r.db == nil || req.TeamgroupID == 0 {
		return columns, BoardTaskSummary{}, nil
	}

	// 按敏捷小组真实成员过滤任务负责人（小组 ↔ 任务无直接表，采用组内成员口径）。
	members, memberErr := r.FindBoardTeamgroupMembers(ctx, req.TeamgroupID)
	if memberErr != nil {
		return nil, BoardTaskSummary{}, memberErr
	}
	q := r.boardTaskQuery(ctx, req)
	if len(members) > 0 {
		q = q.Where("zt_task.assignedTo IN ?", members)
	}
	var rows []boardTaskRow
	if err := q.Select("zt_task.id, zt_task.name, zt_task.type, zt_task.status, zt_task.pri, zt_task.story, zt_task.assignedTo, zt_task.deadline").
		Order("zt_task.id DESC").Limit(req.PageSize).Find(&rows).Error; err != nil {
		return nil, BoardTaskSummary{}, err
	}
	storyTitles, err := r.findBoardStoryTitles(ctx, rows)
	if err != nil {
		return nil, BoardTaskSummary{}, err
	}

	summary := BoardTaskSummary{Total: int64(len(rows))}
	// 本地零点口径（Truncate 按 UTC 截断，本地 0~8 点窗口会错判超期）
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for _, row := range rows {
		blocked := row.Status == "pause"
		overdue := taskIsOverdue(row.Status, row.Deadline, today)
		if blocked {
			summary.Blocked++
		}
		if overdue {
			summary.Overdue++
		}
		if req.Focus == "blocked" && !blocked {
			continue
		}
		if req.Focus == "overdue" && !overdue {
			continue
		}
		columnKey := taskColumnKey(row.Status)
		item := boardTaskItem(row, storyTitles, displayMap, blocked, overdue)
		for _, column := range columns {
			if column.Key == columnKey {
				column.Items = append(column.Items, item)
				break
			}
		}
	}
	return columns, summary, nil
}

type boardTaskRow struct {
	ID         int64      `gorm:"column:id"`
	Name       string     `gorm:"column:name"`
	Type       string     `gorm:"column:type"`
	Status     string     `gorm:"column:status"`
	Priority   int        `gorm:"column:pri"`
	StoryID    int64      `gorm:"column:story"`
	AssignedTo string     `gorm:"column:assignedTo"`
	Deadline   *time.Time `gorm:"column:deadline"`
}

func (r *Repo) boardTaskQuery(ctx context.Context, req BoardTaskReq) *gorm.DB {
	q := r.db.WithContext(ctx).Table("zt_task").
		Where("zt_task.deleted = ?", "0").
		Where("zt_task.status IN ?", []string{"wait", "doing", "done", "pause"})
	if req.StatusFilter != "" {
		q = q.Where("zt_task.status = ?", req.StatusFilter)
	}
	if req.OwnerAccount != "" {
		q = q.Where("zt_task.assignedTo = ?", req.OwnerAccount)
	}
	if req.StoryID > 0 {
		q = q.Where("zt_task.story = ?", req.StoryID)
	}
	if req.Keyword != "" {
		q = q.Where("(zt_task.name LIKE ? OR CAST(zt_task.id AS CHAR) LIKE ?)", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}
	return q
}

func (r *Repo) findBoardStoryTitles(ctx context.Context, tasks []boardTaskRow) (map[int64]string, error) {
	ids := make([]int64, 0, len(tasks))
	seen := make(map[int64]struct{}, len(tasks))
	for _, task := range tasks {
		if task.StoryID > 0 {
			if _, ok := seen[task.StoryID]; !ok {
				seen[task.StoryID] = struct{}{}
				ids = append(ids, task.StoryID)
			}
		}
	}
	if len(ids) == 0 {
		return map[int64]string{}, nil
	}
	var rows []struct {
		ID    int64  `gorm:"column:id"`
		Title string `gorm:"column:title"`
	}
	if err := r.db.WithContext(ctx).Table("zt_story").Where("id IN ? AND deleted = ?", ids, "0").Select("id, title").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[int64]string, len(rows))
	for _, row := range rows {
		out[row.ID] = row.Title
	}
	return out, nil
}

func newBoardTaskColumns() []*BoardTaskColumn {
	return []*BoardTaskColumn{
		{Key: "wait", Name: "未开始", Items: []*BoardTaskItem{}},
		{Key: "doing", Name: "进行中", Items: []*BoardTaskItem{}},
		{Key: "done", Name: "已完成", Items: []*BoardTaskItem{}},
	}
}

func taskColumnKey(status string) string {
	if status == "done" {
		return "done"
	}
	if status == "doing" || status == "pause" {
		return "doing"
	}
	return "wait"
}

func taskIsOverdue(status string, deadline *time.Time, today time.Time) bool {
	return status != "done" && deadline != nil && !deadline.IsZero() && deadline.Before(today)
}

func boardTaskItem(row boardTaskRow, stories map[int64]string, displayMap map[string]string, blocked, overdue bool) *BoardTaskItem {
	owner := displayMap[row.AssignedTo]
	if owner == "" {
		owner = row.AssignedTo
	}
	deadline := ""
	if row.Deadline != nil && !row.Deadline.IsZero() {
		deadline = row.Deadline.Format("2006-01-02")
	}
	priority := ""
	if row.Priority > 0 {
		priority = fmt.Sprintf("P%d", row.Priority)
	}
	return &BoardTaskItem{
		ID: row.ID, DisplayID: fmt.Sprintf("TASK-%d", row.ID), Title: row.Name, Type: row.Type,
		Status: row.Status, Priority: priority, StoryID: row.StoryID, StoryTitle: stories[row.StoryID],
		StoryURL: zentao.StoryViewURL(uint(row.StoryID)), Owner: owner, Deadline: deadline,
		Blocked: blocked, Overdue: overdue, URL: zentao.TaskViewURL(uint(row.ID)),
	}
}

// FindBoardTaskOwners 查询当前小组实际拥有任务的负责人及各自任务数。
func (r *Repo) FindBoardTaskOwners(ctx context.Context, teamgroupID uint, displayMap map[string]string) ([]BoardOwnerOption, error) {
	if r == nil || r.db == nil || teamgroupID == 0 {
		return []BoardOwnerOption{}, nil
	}
	var rows []struct {
		AssignedTo string `gorm:"column:assignedTo"`
		Count      int64  `gorm:"column:cnt"`
	}
	q := r.db.WithContext(ctx).Table("zt_task").
		Where("deleted = ? AND status IN ?", "0", []string{"wait", "doing", "done", "pause"}).
		Where("assignedTo <> ''")
	members, memberErr := r.FindBoardTeamgroupMembers(ctx, teamgroupID)
	if memberErr != nil {
		return nil, memberErr
	}
	if len(members) > 0 {
		q = q.Where("assignedTo IN ?", members)
	}
	if err := q.Select("assignedTo, COUNT(*) AS cnt").Group("assignedTo").Order("cnt DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]BoardOwnerOption, 0, len(rows))
	for _, row := range rows {
		display := displayMap[row.AssignedTo]
		if display == "" {
			display = row.AssignedTo
		}
		out = append(out, BoardOwnerOption{Account: row.AssignedTo, Display: display, Count: row.Count})
	}
	return out, nil
}
