// =============================================================================
// 文件: internal/module/kanban/repo.go
// 模块: 工作看板
// 类型: action
// 职责: 看板数据访问（敏捷小组/成员、任务三列查询、任务状态读取、独立研需查询）。
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

const (
	kanbanTaskLimit       = 200
	independentStoryLimit = 200
)

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

// IndependentStoryRow 独立研需（zt_story）供看板行展示。
type IndependentStoryRow struct {
	ID            int64      `gorm:"column:id"`
	Title         string     `gorm:"column:title"`
	Status        string     `gorm:"column:status"`
	Stage         string     `gorm:"column:stage"`
	Pri           int        `gorm:"column:pri"`
	AssignedTo    string     `gorm:"column:assignedTo"`
	DevelopFinish *time.Time `gorm:"column:developFinish"`
	TestFinish    *time.Time `gorm:"column:testFinish"`
	VerifyFinish  *time.Time `gorm:"column:verifyFinish"`
	DeliverDate   *time.Time `gorm:"column:deliverDate"`
}

// FindIndependentStoriesByAccounts 严格按照 PRD《价值流阶段数据统计逻辑.xlsx》查询独立研发需求。
// 过滤条件：非需求池(sourceType != 'demandpool')、非父需求(isParent = '0')、未删除且未关闭、
// 指派人或所属产品需求负责人(ReqM)为当前团队成员。
func (r *Repo) FindIndependentStoriesByAccounts(ctx context.Context, accounts []string) ([]IndependentStoryRow, error) {
	if r == nil || r.db == nil || len(accounts) == 0 {
		return []IndependentStoryRow{}, nil
	}
	cleaned := make([]string, 0, len(accounts))
	for _, a := range accounts {
		if s := strings.TrimSpace(a); s != "" {
			cleaned = append(cleaned, s)
		}
	}
	if len(cleaned) == 0 {
		return []IndependentStoryRow{}, nil
	}

	var rows []IndependentStoryRow
	err := r.db.WithContext(ctx).Table("zt_story").
		Where("deleted = ?", "0").
		Where("status != ?", "closed").
		Where("type = ?", "story").
		Where("isParent = ?", "0").
		Where("(fromDemand IS NULL OR fromDemand = 0)").
		Where("IFNULL(sourceType, '') != ?", "demandpool").
		Where(`(
			zt_story.assignedTo IN ?
			OR EXISTS (
				SELECT 1 FROM zt_product p
				WHERE p.id = zt_story.product AND p.deleted = '0' AND p.ReqM IN ?
			)
		)`, cleaned, cleaned).
		Select("id, title, status, stage, pri, assignedTo, developFinish, testFinish, verifyFinish, deliverDate").
		Order("id DESC").
		Limit(independentStoryLimit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []IndependentStoryRow{}, nil
	}
	return rows, nil
}
