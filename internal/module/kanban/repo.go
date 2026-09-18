// =============================================================================
// 文件: internal/module/kanban/repo.go
// 模块: 工作看板
// 类型: action
// 职责: 看板数据访问（敏捷小组/成员、任务三列查询、任务状态读取）。
// 依赖: internal/model/zentao
// =============================================================================

package kanban

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	ztmodel "workbench/internal/model/zentao"
)

const kanbanTaskLimit = 200

// Repo 封装看板数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// teamgroupRow 用户所属小组原始行（含 PO / 敏捷教练）。
type teamgroupRow struct {
	ID      uint
	Name    string
	PO      string
	Manager string
}

// teamMemberRow 小组成员原始行。
type teamMemberRow struct {
	Root    uint
	Account string
	Order   int8
}

// ListUserTeamgroups 查询账号所属敏捷小组，按加入时间倒序。
func (r *Repo) ListUserTeamgroups(ctx context.Context, account string) ([]teamgroupRow, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return []teamgroupRow{}, nil
	}

	var rows []teamgroupRow
	err := r.db.WithContext(ctx).
		Table((ztmodel.ZtTeam{}).TableName()+" AS t").
		Select("g.id, g.name, g.PO, g.manager").
		Joins("LEFT JOIN "+(ztmodel.ZtTeamgroup{}).TableName()+" AS g ON t.root = g.id").
		Where("t.account = ? AND t.type = ? AND g.deleted = ?", account, "teamgroup", "0").
		Order("t.`join` DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []teamgroupRow{}, nil
	}
	return rows, nil
}

// ListTeamMembersByRoots 批量查询敏捷小组成员账号（按 order 升序）。
func (r *Repo) ListTeamMembersByRoots(ctx context.Context, roots []uint) ([]teamMemberRow, error) {
	if len(roots) == 0 {
		return []teamMemberRow{}, nil
	}

	var rows []teamMemberRow
	err := r.db.WithContext(ctx).
		Table((ztmodel.ZtTeam{}).TableName()).
		Select("root, account, `order`").
		Where("root IN ? AND type = ?", roots, "teamgroup").
		Order("`order` ASC, id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []teamMemberRow{}, nil
	}
	return rows, nil
}

// FindKanbanTasks 查询选中账号（可多个）的看板任务。
// wait/doing：assignedTo IN accounts；done：finishedBy IN accounts；不含 pause。
func (r *Repo) FindKanbanTasks(ctx context.Context, accounts []string) ([]taskRow, error) {
	if r == nil || r.db == nil {
		return []taskRow{}, nil
	}
	cleaned := make([]string, 0, len(accounts))
	seen := make(map[string]struct{}, len(accounts))
	for _, a := range accounts {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if _, ok := seen[a]; ok {
			continue
		}
		seen[a] = struct{}{}
		cleaned = append(cleaned, a)
	}
	if len(cleaned) == 0 {
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
			(t.assignedTo IN ? AND t.status IN (?, ?))
			OR (t.finishedBy IN ? AND t.status = ?)
		)`, cleaned, "wait", "doing", cleaned, "done").
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

// taskStatusRow 任务状态更新所需最小字段。
type taskStatusRow struct {
	ID         int64
	Status     string
	AssignedTo string
}

// FindTaskStatusByID 按 ID 查询未删除任务的 status / assignedTo。
func (r *Repo) FindTaskStatusByID(ctx context.Context, id int64) (*taskStatusRow, error) {
	if r == nil || r.db == nil || id <= 0 {
		return nil, gorm.ErrRecordNotFound
	}
	type rawRow struct {
		ID         int64  `gorm:"column:id"`
		Status     string `gorm:"column:status"`
		AssignedTo string `gorm:"column:assignedTo"`
	}
	var row rawRow
	err := r.db.WithContext(ctx).
		Table("zt_task").
		Select("id, status, assignedTo").
		Where("id = ? AND deleted = ?", id, "0").
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &taskStatusRow{
		ID:         row.ID,
		Status:     row.Status,
		AssignedTo: row.AssignedTo,
	}, nil
}

// FindStoryCountsByDemands 批量统计业务需求下关联的未删除研发需求数（走 fromDemand 索引）。
func (r *Repo) FindStoryCountsByDemands(ctx context.Context, demandIDs []int64) (map[int64]int, error) {
	if len(demandIDs) == 0 || r == nil || r.db == nil {
		return map[int64]int{}, nil
	}
	type countRow struct {
		FromDemand int64 `gorm:"column:fromDemand"`
		Cnt        int   `gorm:"column:cnt"`
	}
	var rows []countRow
	err := r.db.WithContext(ctx).Table("zt_story").
		Select("fromDemand, COUNT(1) AS cnt").
		Where("fromDemand IN ? AND deleted = ?", demandIDs, "0").
		Group("fromDemand").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	res := make(map[int64]int, len(rows))
	for _, row := range rows {
		res[row.FromDemand] = row.Cnt
	}
	return res, nil
}

// FindTaskStatsByStories 批量统计研发需求下的任务总量与已完成任务数（走 story 索引）。
func (r *Repo) FindTaskStatsByStories(ctx context.Context, storyIDs []int64) (map[int64][2]int, error) {
	if len(storyIDs) == 0 || r == nil || r.db == nil {
		return map[int64][2]int{}, nil
	}
	type taskStatRow struct {
		Story int64 `gorm:"column:story"`
		Total int   `gorm:"column:total"`
		Done  int   `gorm:"column:done"`
	}
	var rows []taskStatRow
	err := r.db.WithContext(ctx).Table("zt_task").
		Select("story, COUNT(1) AS total, SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) AS done", "done").
		Where("story IN ? AND deleted = ?", storyIDs, "0").
		Group("story").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	res := make(map[int64][2]int, len(rows))
	for _, row := range rows {
		res[row.Story] = [2]int{row.Done, row.Total}
	}
	return res, nil
}

// IndependentStoryRow 独立研需（zt_story）供看板行展示。
type IndependentStoryRow struct {
	ID         int64  `gorm:"column:id"`
	Title      string `gorm:"column:title"`
	Status     string `gorm:"column:status"`
	Stage      string `gorm:"column:stage"`
	Pri        string `gorm:"column:pri"`
	AssignedTo string `gorm:"column:assignedTo"`
	OpenedBy   string `gorm:"column:openedBy"`
	Product    int64  `gorm:"column:product"`
	SourceType string `gorm:"column:sourceType"`
}

// FindIndependentStoriesByAccounts 查询指定负责人的独立研发需求（非需求池来源且无关联业务需求）。
func (r *Repo) FindIndependentStoriesByAccounts(ctx context.Context, accounts []string) ([]IndependentStoryRow, error) {
	if len(accounts) == 0 || r == nil || r.db == nil {
		return nil, nil
	}
	var rows []IndependentStoryRow
	excludedSource := []string{"", "demandpool", "demandlib", "feedback"}
	err := r.db.WithContext(ctx).Table("zt_story").
		Where("(fromDemand IS NULL OR fromDemand = 0)").
		Where("deleted = ?", "0").
		Where("status != ?", "closed").
		Where("sourceType NOT IN ?", excludedSource).
		Where("assignedTo IN ?", accounts).
		Select("id, title, status, stage, pri, assignedTo, openedBy, product, sourceType").
		Order("id DESC").
		Limit(100).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
