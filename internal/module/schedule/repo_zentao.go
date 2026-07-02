// =============================================================================
// 文件: internal/module/schedule/repo_zentao.go
// 模块: 排期工作台
// 类型: action
// 职责: 禅道只读查询（业需、研发需求、需求池等）。
// 依赖: internal/module/schedule/form.go
// =============================================================================

package schedule

import (
	"context"
	"strings"
)

type clarifyProductCountRow struct {
	Demand       uint `gorm:"column:demand"`
	ProductCount int  `gorm:"column:productCount"`
}

type storyWindowRow struct {
	StoryID     uint   `gorm:"column:story"`
	WindowID    uint   `gorm:"column:windowID"`
	WindowName  string `gorm:"column:windowName"`
	TeamgroupID uint   `gorm:"column:teamgroupID"`
}

type storyTaskStatRow struct {
	StoryID    uint `gorm:"column:story"`
	Total      int  `gorm:"column:total"`
	Unassigned int  `gorm:"column:unassigned"`
}

type userRealnameRow struct {
	Account  string `gorm:"column:account"`
	Realname string `gorm:"column:realname"`
}

// GetUserDemandPools 返回用户可见的需求池 ID 列表。
func (r *Repo) GetUserDemandPools(ctx context.Context, account string) ([]uint, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return []uint{}, nil
	}

	isAdmin, err := r.IsAdmin(ctx, account)
	if err != nil {
		return nil, err
	}
	if isAdmin {
		return r.listAllDemandPoolIDs(ctx)
	}
	return r.listDemandPoolIDsForUser(ctx, account)
}

func (r *Repo) listAllDemandPoolIDs(ctx context.Context) ([]uint, error) {
	const query = `
SELECT id
FROM zt_demandpool
WHERE deleted = '0'
ORDER BY id ASC`

	var ids []uint
	if err := r.db.WithContext(ctx).Raw(query).Scan(&ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *Repo) listDemandPoolIDsForUser(ctx context.Context, account string) ([]uint, error) {
	deptIDs, err := r.loadUserDeptIDs(ctx, account)
	if err != nil {
		return nil, err
	}

	const query = `
SELECT id
FROM zt_demandpool
WHERE deleted = '0'
  AND (
    acl = 'open'
    OR FIND_IN_SET(?, participant) > 0
    OR FIND_IN_SET(?, businessReviewer) > 0
    OR dept IN ?
  )
ORDER BY id ASC`

	var ids []uint
	if err := r.db.WithContext(ctx).Raw(query, account, account, deptIDs).Scan(&ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *Repo) loadUserDeptIDs(ctx context.Context, account string) ([]uint, error) {
	const userQuery = `
SELECT dept
FROM zt_user
WHERE account = ? AND deleted = '0'
LIMIT 1`

	var dept uint
	if err := r.db.WithContext(ctx).Raw(userQuery, account).Scan(&dept).Error; err != nil {
		return nil, err
	}
	if dept == 0 {
		return []uint{0}, nil
	}

	const pathQuery = `
SELECT path
FROM zt_dept
WHERE id = ?
LIMIT 1`

	var path string
	if err := r.db.WithContext(ctx).Raw(pathQuery, dept).Scan(&path).Error; err != nil {
		return nil, err
	}

	deptIDs := parseDeptPathIDs(path)
	seen := make(map[uint]struct{}, len(deptIDs)+1)
	unique := make([]uint, 0, len(deptIDs)+1)
	for _, id := range append(deptIDs, dept) {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return []uint{dept}, nil
	}
	return unique, nil
}

func parseDeptPathIDs(path string) []uint {
	path = strings.Trim(path, ",")
	if path == "" {
		return []uint{}
	}
	parts := strings.Split(path, ",")
	ids := make([]uint, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var id uint
		for i := 0; i < len(part); i++ {
			if part[i] < '0' || part[i] > '9' {
				id = 0
				break
			}
			id = id*10 + uint(part[i]-'0')
		}
		if id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

// ListBizDemands 顶层业需分页主查询（parent IN (0,-1)：含被拆分的父需求）。
func (r *Repo) ListBizDemands(ctx context.Context, req ListBizDemandsReq, poolIDs []uint, account string) ([]ZtDemand, int64, error) {
	if len(poolIDs) == 0 {
		return []ZtDemand{}, 0, nil
	}

	clause := mergeFilterClauses(
		applyBizDemandSuspended(buildBizDemandFilterClause(req.Filter, account), req.Suspended),
		buildBizDemandAdvancedClause(advancedFilterParamsFromBizReq(req)),
	)
	countArgs := append([]interface{}{poolIDs}, clause.args...)
	const countQuery = `
SELECT COUNT(*) AS total
FROM zt_demand d
WHERE d.deleted = '0'
  AND d.parent IN (0, -1)
  AND d.pool IN ?`

	var total int64
	if err := r.db.WithContext(ctx).Raw(countQuery+clause.sql, countArgs...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (req.Page - 1) * req.PageSize
	listArgs := append([]interface{}{poolIDs}, clause.args...)
	listArgs = append(listArgs, req.PageSize, offset)
	const listQuery = `
SELECT
  d.id,
  d.name,
  d.pri,
  d.status,
  d.assignedTo,
  d.mainSystem,
  d.teamGroup,
  d.BRA,
  d.QD,
  d.RD,
  d.createdBy,
  d.pool,
  d.parent,
  d.hang,
  d.category,
  d.estimateLaunch
FROM zt_demand d
WHERE d.deleted = '0'
  AND d.parent IN (0, -1)
  AND d.pool IN ?`

	var rows []ZtDemand
	if err := r.db.WithContext(ctx).Raw(listQuery+clause.sql+`
ORDER BY d.id DESC
LIMIT ? OFFSET ?`, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// FindChildDemandsByParents 批量查询子业需。
func (r *Repo) FindChildDemandsByParents(ctx context.Context, parentIDs []uint) ([]ZtDemand, error) {
	if len(parentIDs) == 0 {
		return []ZtDemand{}, nil
	}

	const query = `
SELECT
  id, name, pri, status, assignedTo, mainSystem, teamGroup,
  BRA, QD, RD, createdBy, pool, parent, hang, category, estimateLaunch
FROM zt_demand
WHERE deleted = '0'
  AND parent IN ?
ORDER BY parent ASC, id ASC`

	var rows []ZtDemand
	if err := r.db.WithContext(ctx).Raw(query, parentIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

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
