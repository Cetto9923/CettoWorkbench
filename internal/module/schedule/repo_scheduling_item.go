// =============================================================================
// 文件: internal/module/schedule/repo_scheduling_item.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期一体化弹窗下拉级联 + 研发需求/任务项查询:
//       未过期窗口列表 / 内部用户 / 业需涉及系统 / 业需研发需求 /
//       业需用户故事 / 研发需求任务 / 产品项目 / 项目执行 / 批量项目名。
// 依赖: (无项目内部包)
// =============================================================================

package schedule

import (
	"context"
	"strings"
)

type schedulingWindowRow struct {
	ID          uint   `gorm:"column:id"`
	Name        string `gorm:"column:name"`
	ReleaseDate string `gorm:"column:releaseDate"`
}

type schedulingUserRow struct {
	Account  string `gorm:"column:account"`
	Realname string `gorm:"column:realname"`
}

// ListUpcomingSchedulingWindows 查询未过期的版本窗口列表。
func (r *Repo) ListUpcomingSchedulingWindows(ctx context.Context) ([]SchedulingWindowOption, error) {
	const query = `
SELECT
  id,
  name,
  DATE_FORMAT(releaseDate, '%Y-%m-%d') AS releaseDate
FROM zt_versionwindow
WHERE deletedAt IS NULL
  AND releaseDate >= CURDATE()
ORDER BY releaseDate ASC`

	var rows []schedulingWindowRow
	if err := r.db.WithContext(ctx).Raw(query).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]SchedulingWindowOption, 0, len(rows))
	for _, row := range rows {
		out = append(out, SchedulingWindowOption{
			ID:          row.ID,
			Name:        strings.TrimSpace(row.Name),
			ReleaseDate: formatZenTaoDate(row.ReleaseDate),
		})
	}
	return out, nil
}

// ListInsideUsersForScheduling 查询内部用户列表（负责人下拉）。
func (r *Repo) ListInsideUsersForScheduling(ctx context.Context) ([]SchedulingUserOption, error) {
	const query = `
SELECT account, realname
FROM zt_user
WHERE deleted = '0'
  AND type = 'inside'
ORDER BY account ASC`

	var rows []schedulingUserRow
	if err := r.db.WithContext(ctx).Raw(query).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]SchedulingUserOption, 0, len(rows))
	for _, row := range rows {
		account := strings.TrimSpace(row.Account)
		if account == "" {
			continue
		}
		realname := strings.TrimSpace(row.Realname)
		if realname == "" {
			realname = account
		}
		out = append(out, SchedulingUserOption{
			Account:  account,
			Realname: realname,
		})
	}
	return out, nil
}

// GetDemandInvolvedProducts 查询业需涉及的系统列表（来自 demandclarify）。
func (r *Repo) GetDemandInvolvedProducts(ctx context.Context, demandID uint) ([]ZtProductOption, error) {
	if demandID == 0 {
		return []ZtProductOption{}, nil
	}

	const query = `
SELECT DISTINCT p.id, p.name
FROM zt_demandclarify dc
JOIN zt_product p ON p.id = dc.product AND p.deleted = '0'
WHERE dc.demand = ?
ORDER BY p.id ASC`

	var rows []ZtProductOption
	if err := r.db.WithContext(ctx).Raw(query, demandID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ZtProductOption, 0, len(rows))
	for _, row := range rows {
		if row.ID == 0 {
			continue
		}
		out = append(out, ZtProductOption{
			ID:   row.ID,
			Name: strings.TrimSpace(row.Name),
		})
	}
	return out, nil
}

// GetDemandStories 按业需 ID 查询关联研发需求（排期弹窗用户故事条目）。
func (r *Repo) GetDemandStories(ctx context.Context, demandID uint) ([]ZtStory, error) {
	if demandID == 0 {
		return []ZtStory{}, nil
	}

	const query = `
SELECT
  id,
  title,
  product,
  estimate,
  assignedTo,
  CAST(IFNULL(NULLIF(isMainSystemAssociation, ''), '0') AS SIGNED) AS isMainSystemAssociation
FROM zt_story
WHERE fromDemand = ?
  AND deleted = '0'
ORDER BY isMainSystemAssociation DESC, id ASC`

	var rows []ZtStory
	if err := r.db.WithContext(ctx).Raw(query, demandID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetDemandUserStories 按业需 ID 查询用户故事条目（来自 zt_demanduserstory）。
func (r *Repo) GetDemandUserStories(ctx context.Context, demandID uint) ([]ZtDemandUserStory, error) {
	if demandID == 0 {
		return []ZtDemandUserStory{}, nil
	}

	const query = `
SELECT
  id,
  demand,
  role,
  gv,
  product,
  point,
  revpoint,
  source_type AS sourceType
FROM zt_demanduserstory
WHERE demand = ?
ORDER BY id ASC`

	var rows []ZtDemandUserStory
	if err := r.db.WithContext(ctx).Raw(query, demandID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetStoryTasks 查询某个研发需求下的未关闭任务列表。
func (r *Repo) GetStoryTasks(ctx context.Context, storyID uint) ([]ZtTaskItem, error) {
	if storyID == 0 {
		return []ZtTaskItem{}, nil
	}

	const query = `
SELECT
  id,
  name,
  type,
  pri,
  assignedTo,
  estimate,
  consumed,
  ` + "`left`" + `,
  DATE_FORMAT(estStarted, '%Y-%m-%d') AS estStarted,
  DATE_FORMAT(deadline, '%Y-%m-%d') AS deadline,
  status,
  finishedBy,
  DATE_FORMAT(finishedDate, '%Y-%m-%d') AS finishedDate,
  project,
  execution
FROM zt_task
WHERE story = ?
  AND deleted = '0'
  AND status != 'closed'
ORDER BY id ASC`

	var rows []ZtTaskItem
	if err := r.db.WithContext(ctx).Raw(query, storyID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetProductProjects 查询产品关联的进行中和未开始项目。
func (r *Repo) GetProductProjects(ctx context.Context, productID uint) ([]ZtProjectOption, error) {
	if productID == 0 {
		return []ZtProjectOption{}, nil
	}

	const query = `
SELECT p.id, p.name, p.status, p.model
FROM zt_projectproduct pp
JOIN zt_project p ON p.id = pp.project AND p.deleted = '0' AND p.type = 'project'
WHERE pp.product = ?
  AND p.status IN ('doing', 'wait')
ORDER BY p.id DESC`

	var rows []ZtProjectOption
	if err := r.db.WithContext(ctx).Raw(query, productID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetProjectExecutions 查询项目下进行中和未开始的执行。
func (r *Repo) GetProjectExecutions(ctx context.Context, projectID uint) ([]ZtExecutionOption, error) {
	if projectID == 0 {
		return []ZtExecutionOption{}, nil
	}

	const query = `
SELECT id, name, type, status
FROM zt_project
WHERE parent = ?
  AND type IN ('sprint', 'stage', 'kanban')
  AND deleted = '0'
  AND status IN ('doing', 'wait')
ORDER BY id DESC`

	var rows []ZtExecutionOption
	if err := r.db.WithContext(ctx).Raw(query, projectID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// FindProjectsByIDs 批量查项目名称。
func (r *Repo) FindProjectsByIDs(ctx context.Context, ids []uint) (map[uint]string, error) {
	if len(ids) == 0 {
		return map[uint]string{}, nil
	}

	const query = `
SELECT id, name
FROM zt_project
WHERE id IN ?
  AND deleted = '0'`

	type projectNameRow struct {
		ID   uint   `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	var rows []projectNameRow
	if err := r.db.WithContext(ctx).Raw(query, ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]string, len(rows))
	for _, row := range rows {
		if row.ID == 0 {
			continue
		}
		out[row.ID] = strings.TrimSpace(row.Name)
	}
	return out, nil
}
