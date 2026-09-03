// =============================================================================
// 文件: internal/module/schedule/repo_zentao_story.go
// 模块: 排期工作台
// 类型: action
// 职责: 禅道研发需求(zt_story)只读查询 + 关联窗口/任务统计。
// 依赖: (无项目内部包)
// =============================================================================

package schedule

import (
	"context"
	"strings"
)

// FindStoriesByDemands 按业需 ID 批量查研发需求。
func (r *Repo) FindStoriesByDemands(ctx context.Context, demandIDs []uint) ([]ZtStory, error) {
	if len(demandIDs) == 0 {
		return []ZtStory{}, nil
	}

	const query = `
SELECT
  id,
  title,
  pri,
  product,
  plan,
  stage,
  status,
  fromDemand,
  sourceType,
  parent,
  CAST(IFNULL(NULLIF(isMainSystemAssociation, ''), '0') AS SIGNED) AS isMainSystemAssociation,
  assignedTo
FROM zt_story
WHERE fromDemand IN ?
  AND sourceType = 'demandpool'
  AND type = 'story'
  AND deleted = '0'
ORDER BY fromDemand ASC, isMainSystemAssociation DESC, id ASC`

	var rows []ZtStory
	if err := r.db.WithContext(ctx).Raw(query, demandIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CountClarifyProductsByDemands 统计业需关联的多系统数。
func (r *Repo) CountClarifyProductsByDemands(ctx context.Context, demandIDs []uint) (map[uint]int, error) {
	if len(demandIDs) == 0 {
		return map[uint]int{}, nil
	}

	const query = `
SELECT demand, COUNT(DISTINCT product) AS productCount
FROM zt_demandclarify
WHERE demand IN ?
  AND TRIM(IFNULL(product, '')) != ''
GROUP BY demand`

	var rows []clarifyProductCountRow
	if err := r.db.WithContext(ctx).Raw(query, demandIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]int, len(rows))
	for _, row := range rows {
		out[row.Demand] = row.ProductCount
	}
	return out, nil
}

// FindProductsByIDs 批量查产品/系统名称。
func (r *Repo) FindProductsByIDs(ctx context.Context, productIDs []uint) (map[uint]string, error) {
	if len(productIDs) == 0 {
		return map[uint]string{}, nil
	}

	const query = `
SELECT id, name
FROM zt_product
WHERE id IN ?
  AND deleted = '0'`

	var rows []ZtProduct
	if err := r.db.WithContext(ctx).Raw(query, productIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]string, len(rows))
	for _, row := range rows {
		out[row.ID] = strings.TrimSpace(row.Name)
	}
	return out, nil
}

// FindStoryWindowMappings 查研发需求关联的版本窗口（每 story 取 vw.id 最小的一条）。
func (r *Repo) FindStoryWindowMappings(ctx context.Context, storyIDs []uint) (map[uint]StoryWindowRef, error) {
	if len(storyIDs) == 0 {
		return map[uint]StoryWindowRef{}, nil
	}

	const query = `
SELECT ps.story, vw.id AS windowID, vw.name AS windowName, vw.teamgroup AS teamgroupID
FROM zt_planstory ps
INNER JOIN zt_versionwindowproduct vwp
  ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
INNER JOIN zt_versionwindow vw
  ON vw.id = vwp.versionWindow AND vw.deletedAt IS NULL
WHERE ps.story IN ?
ORDER BY ps.story ASC, vw.id ASC`

	var rows []storyWindowRow
	if err := r.db.WithContext(ctx).Raw(query, storyIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]StoryWindowRef, len(rows))
	for _, row := range rows {
		if _, exists := out[row.StoryID]; exists {
			continue
		}
		out[row.StoryID] = StoryWindowRef{
			StoryID:     row.StoryID,
			WindowID:    row.WindowID,
			WindowName:  strings.TrimSpace(row.WindowName),
			TeamgroupID: row.TeamgroupID,
		}
	}
	return out, nil
}

// CountStoryTasks 统计研发需求下任务总数与未指派数。
func (r *Repo) CountStoryTasks(ctx context.Context, storyIDs []uint) (map[uint]StoryTaskStat, error) {
	if len(storyIDs) == 0 {
		return map[uint]StoryTaskStat{}, nil
	}

	const query = `
SELECT
  story,
  COUNT(*) AS total,
  SUM(CASE WHEN assignedTo IS NULL OR assignedTo = '' THEN 1 ELSE 0 END) AS unassigned
FROM zt_task
WHERE story IN ?
  AND deleted = '0'
  AND status != 'closed'
GROUP BY story`

	var rows []storyTaskStatRow
	if err := r.db.WithContext(ctx).Raw(query, storyIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]StoryTaskStat, len(rows))
	for _, row := range rows {
		out[row.StoryID] = StoryTaskStat{
			StoryID:    row.StoryID,
			Total:      row.Total,
			Unassigned: row.Unassigned,
		}
	}
	return out, nil
}

// ListIndependentStories 分页查询独立研发需求（顶层 parent=0）。
func (r *Repo) ListIndependentStories(ctx context.Context, req ListIndependentReq, productIDs []uint, account string) ([]ZtStory, int64, error) {
	if len(productIDs) == 0 {
		return []ZtStory{}, 0, nil
	}

	clause := mergeFilterClauses(
		buildIndepStoryFilterClause(req.Filter, account),
		buildIndepStoryAdvancedClause(advancedFilterParamsFromIndepReq(req)),
	)
	countArgs := append([]interface{}{productIDs}, clause.args...)
	const countQuery = `
SELECT COUNT(*) AS total
FROM zt_story s
WHERE IFNULL(s.sourceType, '') != 'demandpool'
  AND s.parent = 0
  AND s.type = 'story'
  AND s.deleted = '0'
  AND s.product IN ?`

	var total int64
	if err := r.db.WithContext(ctx).Raw(countQuery+clause.sql, countArgs...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	page := req.Page
	pageSize := req.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	listArgs := append([]interface{}{productIDs}, clause.args...)
	listArgs = append(listArgs, pageSize, offset)
	const listQuery = `
SELECT
  s.id,
  s.title,
  s.pri,
  s.product,
  s.plan,
  s.stage,
  s.status,
  s.fromDemand,
  s.sourceType,
  s.parent,
  CAST(IFNULL(NULLIF(s.isMainSystemAssociation, ''), '0') AS SIGNED) AS isMainSystemAssociation,
  s.assignedTo
FROM zt_story s
WHERE IFNULL(s.sourceType, '') != 'demandpool'
  AND s.parent = 0
  AND s.type = 'story'
  AND s.deleted = '0'
  AND s.product IN ?`

	var rows []ZtStory
	if err := r.db.WithContext(ctx).Raw(listQuery+clause.sql+`
ORDER BY s.id DESC
LIMIT ? OFFSET ?`, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// FindChildStories 批量查子研发需求。
func (r *Repo) FindChildStories(ctx context.Context, parentIDs []uint) ([]ZtStory, error) {
	if len(parentIDs) == 0 {
		return []ZtStory{}, nil
	}

	const query = `
SELECT
  id,
  title,
  pri,
  product,
  plan,
  stage,
  status,
  fromDemand,
  sourceType,
  parent,
  CAST(IFNULL(NULLIF(isMainSystemAssociation, ''), '0') AS SIGNED) AS isMainSystemAssociation,
  assignedTo
FROM zt_story
WHERE parent IN ?
  AND deleted = '0'
ORDER BY parent ASC, id ASC`

	var rows []ZtStory
	if err := r.db.WithContext(ctx).Raw(query, parentIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
