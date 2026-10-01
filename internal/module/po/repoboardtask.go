// =============================================================================
// 文件: internal/module/po/repoboardtask.go
// 模块: PO 工作台
// 类型: repo
// 职责: 任务看板只读查询。
// =============================================================================

package po

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"workbench/internal/pkg/datefmt"
	"workbench/internal/pkg/zentao"
)

// FindBoardTaskList 查询当前敏捷小组中的真实任务，并按原型归入三列。
func (r *Repo) FindBoardTaskList(ctx context.Context, req BoardTaskReq, displayMap map[string]string) ([]*BoardTaskColumn, BoardTaskSummary, error) {
	columns := newBoardTaskColumns()
	if r == nil || r.db == nil || (req.TeamgroupID == 0 && req.StoryID == 0) {
		return columns, BoardTaskSummary{}, nil
	}

	q := r.boardTaskQuery(ctx, req)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var summary BoardTaskSummary
	if err := q.Session(&gorm.Session{}).Select(`COUNT(*) AS total,
		COALESCE(SUM(zt_task.status = 'pause'), 0) AS blocked,
		COALESCE(SUM(zt_task.status <> 'done' AND zt_task.deadline > '1970-01-01' AND zt_task.deadline < ?), 0) AS overdue`, today).
		Scan(&summary).Error; err != nil {
		return nil, summary, err
	}
	summary.FilteredTotal = summary.Total
	switch req.Focus {
	case "blocked":
		q = q.Where("zt_task.status = ?", "pause")
		summary.FilteredTotal = summary.Blocked
	case "overdue":
		q = q.Where("zt_task.status <> ? AND zt_task.deadline > '1970-01-01' AND zt_task.deadline < ?", "done", today)
		summary.FilteredTotal = summary.Overdue
	}
	var rows []boardTaskRow
	if err := q.Select("zt_task.id, zt_task.name, zt_task.type, zt_task.status, zt_task.pri, zt_task.story, zt_task.assignedTo, zt_task.finishedBy, zt_task.deadline").
		Order("zt_task.id DESC").Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize).Find(&rows).Error; err != nil {
		return nil, BoardTaskSummary{}, err
	}
	storyTitles, err := r.findBoardStoryTitles(ctx, rows)
	if err != nil {
		return nil, BoardTaskSummary{}, err
	}

	for _, row := range rows {
		blocked := row.Status == "pause"
		overdue := taskIsOverdue(row.Status, row.Deadline, today)
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
	FinishedBy string     `gorm:"column:finishedBy"`
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
		q = q.Where("("+boardTaskOwnerSQL+") = ?", req.OwnerAccount)
	}
	if req.StoryID > 0 {
		q = q.Where("zt_task.story = ?", req.StoryID)
	} else if len(req.members) == 0 {
		q = q.Where("1 = 0")
	} else {
		q = q.Where("((zt_task.status <> 'done' AND zt_task.assignedTo IN ?) OR (zt_task.status = 'done' AND zt_task.finishedBy IN ?))", req.members, req.members)
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
	return status != "done" && deadline != nil && deadline.Year() > 1970 && deadline.Before(today)
}

func boardTaskItem(row boardTaskRow, stories map[int64]string, displayMap map[string]string, blocked, overdue bool) *BoardTaskItem {
	ownerAccount := row.AssignedTo
	if row.Status == "done" {
		ownerAccount = row.FinishedBy
	}
	owner := displayMap[ownerAccount]
	if owner == "" {
		owner = ownerAccount
	}
	deadline := datefmt.Date(row.Deadline)
	priority := ""
	if row.Priority > 0 {
		priority = fmt.Sprintf("P%d", row.Priority)
	}
	return &BoardTaskItem{
		ID: row.ID, DisplayID: fmt.Sprintf("%d", row.ID), Title: row.Name, Type: row.Type,
		Status: row.Status, Priority: priority, StoryID: row.StoryID, StoryTitle: stories[row.StoryID],
		StoryURL: zentao.StoryViewURL(uint(row.StoryID)), Owner: owner, OwnerAccount: ownerAccount, Deadline: deadline,
		Blocked: blocked, Overdue: overdue, URL: zentao.TaskViewURL(uint(row.ID)),
	}
}

// FindBoardTaskForTransition 返回任务拖拽状态变更所需的最小 ZenTao 字段。
func (r *Repo) FindBoardTaskForTransition(ctx context.Context, taskID uint) (*boardTaskTransitionRow, error) {
	if r == nil || r.db == nil || taskID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var row boardTaskTransitionRow
	err := r.db.WithContext(ctx).Table("zt_task").
		Where("id = ? AND deleted = ?", taskID, "0").
		Select("id, status, assignedTo").
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

type boardTaskTransitionRow struct {
	ID         uint   `gorm:"column:id"`
	Status     string `gorm:"column:status"`
	AssignedTo string `gorm:"column:assignedTo"`
}

// boardTaskOwnerSQL 使负责人选项、筛选和卡片的完成归属一致。
const boardTaskOwnerSQL = "CASE WHEN zt_task.status = 'done' THEN zt_task.finishedBy ELSE zt_task.assignedTo END"

// FindBoardTaskOwners 复用 Service 已验证的小组成员，不重复读取成员表。
func (r *Repo) FindBoardTaskOwners(ctx context.Context, req BoardTaskReq, displayMap map[string]string) ([]BoardOwnerOption, error) {
	req.OwnerAccount, req.Focus = "", ""
	var rows []struct {
		Account string
		Count   int64
	}
	if err := r.boardTaskQuery(ctx, req).
		Select(boardTaskOwnerSQL + " AS account, COUNT(*) AS count").
		Group(boardTaskOwnerSQL).Order("count DESC, account ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]BoardOwnerOption, 0, len(rows))
	for _, row := range rows {
		if row.Account == "" {
			continue
		}
		display := displayMap[row.Account]
		if display == "" {
			display = row.Account
		}
		out = append(out, BoardOwnerOption{Account: row.Account, Display: display, Count: row.Count})
	}
	return out, nil
}

// CheckStoryAccess 校验指定账号对故事是否具备访问权限（对象级鉴权）。
// 适用范围：
// 1. 故事指派给该用户或由该用户创建；
// 2. 故事派生自需求：用户为该需求的 PM / QD / RD / BRA / assignedTo 或需求所属敏捷小组成员；
// 3. 用户承接或创建了该故事下的任意任务。
func (r *Repo) CheckStoryAccess(ctx context.Context, account string, storyID uint) (bool, error) {
	if r == nil || r.db == nil || account == "" || storyID == 0 {
		return false, nil
	}

	var story struct {
		ID         int64  `gorm:"column:id"`
		FromDemand int64  `gorm:"column:fromDemand"`
		AssignedTo string `gorm:"column:assignedTo"`
		OpenedBy   string `gorm:"column:openedBy"`
	}
	err := r.db.WithContext(ctx).Table("zt_story").
		Where("id = ? AND deleted = ?", storyID, "0").
		Select("id, fromDemand, assignedTo, openedBy").
		Take(&story).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	if story.AssignedTo == account || story.OpenedBy == account {
		return true, nil
	}

	if story.FromDemand > 0 {
		var demandCount int64
		err = r.db.WithContext(ctx).Table("zt_demand AS d").
			Where("d.id = ? AND d.deleted = ?", story.FromDemand, "0").
			Where(`d.assignedTo = ? OR d.QD = ? OR d.RD = ? OR d.BRA = ?
				OR d.id IN (SELECT demand FROM zt_demandclarify WHERE PM = ?)
				OR (d.teamGroup > 0 AND d.teamGroup IN (
					SELECT root FROM zt_team WHERE type = 'teamgroup' AND account = ?
				))`, account, account, account, account, account, account).
			Count(&demandCount).Error
		if err != nil {
			return false, err
		}
		if demandCount > 0 {
			return true, nil
		}
	}

	var taskCount int64
	err = r.db.WithContext(ctx).Table("zt_task").
		Where("story = ? AND deleted = ?", storyID, "0").
		Where("assignedTo = ? OR openedBy = ?", account, account).
		Count(&taskCount).Error
	if err != nil {
		return false, err
	}
	if taskCount > 0 {
		return true, nil
	}

	return false, nil
}
