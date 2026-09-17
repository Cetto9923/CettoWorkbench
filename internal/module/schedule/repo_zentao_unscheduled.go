// =============================================================================
// 文件: internal/module/schedule/repo_zentao_unscheduled.go
// 模块: 排期工作台
// 类型: action
// 职责: 业需待排期顶层 ID 多阶段查询（避免 OceanBase 相关子查询全表扫）。
// 依赖: internal/module/schedule/unscheduledmatch.go
//       internal/module/schedule/repo_zentao.go
// =============================================================================

package schedule

import (
	"context"
	"strings"
)

// FindUnscheduledBizDemandTopIDs 计算当前用户待排期业需的顶层 ID（未按 pool/hang 过滤）。
// 拆成相关集 → 澄清系统 → 父子 → story/窗口/任务批量查询，避免一条巨型相关 EXISTS。
func (r *Repo) FindUnscheduledBizDemandTopIDs(ctx context.Context, account string) ([]uint, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return nil, nil
	}

	candidates, err := r.findUnscheduledRelatedDemands(ctx, account)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	candidateIDs := make([]uint, 0, len(candidates))
	parentIDs := make([]uint, 0)
	for _, d := range candidates {
		candidateIDs = append(candidateIDs, d.ID)
		if d.Parent == 0 || d.Parent == -1 {
			parentIDs = append(parentIDs, d.ID)
		}
	}

	productCountByDemand, err := r.CountClarifyProductsByDemands(ctx, candidateIDs)
	if err != nil {
		return nil, err
	}

	hasChildren, err := r.findDemandHasChildren(ctx, parentIDs)
	if err != nil {
		return nil, err
	}

	stories, err := r.FindStoriesByDemands(ctx, candidateIDs)
	if err != nil {
		return nil, err
	}
	storiesByDemand := groupStoriesByFromDemand(stories)
	storyIDs := pluckStoryIDs(stories)

	windowByStory, err := r.FindStoryWindowMappings(ctx, storyIDs)
	if err != nil {
		return nil, err
	}
	taskStatByStory, err := r.CountStoryTasks(ctx, storyIDs)
	if err != nil {
		return nil, err
	}

	return collectUnscheduledBizDemandTopIDs(
		candidates,
		productCountByDemand,
		hasChildren,
		storiesByDemand,
		windowByStory,
		taskStatByStory,
	), nil
}

func (r *Repo) findUnscheduledRelatedDemands(ctx context.Context, account string) ([]ZtDemand, error) {
	const query = `
SELECT
  x.id,
  x.parent,
  x.status,
  x.assignedTo,
  x.BRA
FROM (
  SELECT id FROM zt_demand WHERE deleted = '0' AND assignedTo = ?
  UNION
  SELECT id FROM zt_demand WHERE deleted = '0' AND BRA = ?
  UNION
  SELECT demand FROM zt_demandclarify WHERE PM = ? AND TRIM(IFNULL(product, '')) != ''
) rel
INNER JOIN zt_demand x ON x.id = rel.id AND x.deleted = '0'
WHERE x.status IN ('clarified', 'developing')`

	var rows []ZtDemand
	if err := r.db.WithContext(ctx).Raw(query, account, account, account).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *Repo) findDemandHasChildren(ctx context.Context, parentIDs []uint) (map[uint]bool, error) {
	if len(parentIDs) == 0 {
		return map[uint]bool{}, nil
	}

	const query = `
SELECT DISTINCT parent
FROM zt_demand
WHERE deleted = '0'
  AND parent IN ?`

	var parents []int
	if err := r.db.WithContext(ctx).Raw(query, parentIDs).Scan(&parents).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]bool, len(parents))
	for _, p := range parents {
		if p > 0 {
			out[uint(p)] = true
		}
	}
	return out, nil
}

func (r *Repo) countUnscheduledBizDemands(ctx context.Context, poolIDs []uint, account string, suspended bool) (int64, error) {
	topIDs, err := r.FindUnscheduledBizDemandTopIDs(ctx, account)
	if err != nil {
		return 0, err
	}
	if len(topIDs) == 0 {
		return 0, nil
	}

	hang := "0"
	if suspended {
		hang = "1"
	}

	const query = `
SELECT COUNT(*) AS total
FROM zt_demand d
WHERE d.deleted = '0'
  AND d.parent IN (0, -1)
  AND d.pool IN ?
  AND d.id IN ?
  AND d.hang = ?`

	var total int64
	if err := r.db.WithContext(ctx).Raw(query, poolIDs, topIDs, hang).Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (r *Repo) listUnscheduledBizDemands(ctx context.Context, req ListBizDemandsReq, poolIDs []uint, account string) ([]ZtDemand, int64, error) {
	topIDs, err := r.FindUnscheduledBizDemandTopIDs(ctx, account)
	if err != nil {
		return nil, 0, err
	}
	if len(topIDs) == 0 {
		return []ZtDemand{}, 0, nil
	}

	hang := "0"
	if req.Suspended {
		hang = "1"
	}

	advanced := buildBizDemandAdvancedClause(advancedFilterParamsFromBizReq(req))
	countArgs := append([]interface{}{poolIDs, topIDs, hang}, advanced.args...)
	const countQuery = `
SELECT COUNT(*) AS total
FROM zt_demand d
WHERE d.deleted = '0'
  AND d.parent IN (0, -1)
  AND d.pool IN ?
  AND d.id IN ?
  AND d.hang = ?`

	var total int64
	if err := r.db.WithContext(ctx).Raw(countQuery+advanced.sql, countArgs...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []ZtDemand{}, 0, nil
	}

	offset := (req.Page - 1) * req.PageSize
	listArgs := append([]interface{}{poolIDs, topIDs, hang}, advanced.args...)
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
  AND d.pool IN ?
  AND d.id IN ?
  AND d.hang = ?`

	var rows []ZtDemand
	if err := r.db.WithContext(ctx).Raw(listQuery+advanced.sql+`
ORDER BY d.id DESC
LIMIT ? OFFSET ?`, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
