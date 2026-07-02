// =============================================================================
// 文件: internal/module/schedule/repo_zentao_filter.go
// 模块: 排期工作台
// 类型: action
// 职责: 业需与独立研发需求列表快捷筛选 SQL 及数量统计。
// 依赖: internal/module/schedule/form.go
// =============================================================================

package schedule

import (
	"context"
	"strings"
)

type filterClause struct {
	sql  string
	args []interface{}
}

type bizDemandSimpleCountRow struct {
	AllOpen          int64 `gorm:"column:all_open"`
	PendingReview    int64 `gorm:"column:pending_review"`
	ManagerReviewing int64 `gorm:"column:manager_reviewing"`
	Closed           int64 `gorm:"column:closed"`
}

const bizDemandHangSuspendedSQL = ` AND d.hang = '1'`

const bizDemandAllOpenSQL = `AND d.status != 'closed' AND d.status != 'released'`

const bizDemandExcludeReleasedSQL = `AND d.status != 'released'`

const bizDemandUnscheduledExcludeHangSQL = `AND d.hang = '0'`

const indepStoryAllOpenSQL = `AND s.status != 'closed' AND s.status != 'released'`

const indepStoryExcludeReleasedSQL = `AND s.status != 'released'`

type indepStorySimpleCountRow struct {
	AllOpen       int64 `gorm:"column:all_open"`
	PendingReview int64 `gorm:"column:pending_review"`
	Closed        int64 `gorm:"column:closed"`
}

const bizDemandHasChildSQL = `
EXISTS (
  SELECT 1 FROM zt_demand c
  WHERE c.parent = d.id AND c.deleted = '0'
)`

const bizDemandUnscheduledSelfSQL = `
EXISTS (
  SELECT 1 FROM zt_demandclarify dc
  WHERE dc.demand = d.id
    AND TRIM(IFNULL(dc.product, '')) != ''
)
AND (
  d.assignedTo = ?
  OR d.BRA = ?
  OR EXISTS (
    SELECT 1 FROM zt_demandclarify dc
    WHERE dc.demand = d.id
      AND dc.PM = ?
      AND TRIM(IFNULL(dc.product, '')) != ''
  )
)
AND (
  NOT EXISTS (
    SELECT 1 FROM zt_story s
    JOIN zt_planstory ps ON ps.story = s.id
    JOIN zt_versionwindowproduct vwp ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
    WHERE s.fromDemand = d.id
      AND s.deleted = '0'
      AND s.sourceType = 'demandpool'
      AND s.type = 'story'
  )
  OR NOT EXISTS (
    SELECT 1 FROM zt_story s
    WHERE s.fromDemand = d.id
      AND s.deleted = '0'
      AND s.sourceType = 'demandpool'
      AND s.type = 'story'
  )
  OR EXISTS (
    SELECT 1 FROM zt_story s
    WHERE s.fromDemand = d.id
      AND s.deleted = '0'
      AND s.sourceType = 'demandpool'
      AND s.type = 'story'
      AND (
        NOT EXISTS (SELECT 1 FROM zt_task t WHERE t.story = s.id AND t.deleted = '0' AND t.status != 'closed')
        OR EXISTS (
          SELECT 1 FROM zt_task t
          WHERE t.story = s.id AND t.deleted = '0' AND t.status != 'closed'
            AND (t.assignedTo = '' OR t.assignedTo IS NULL)
        )
      )
  )
)`

const bizDemandUnscheduledChildSQL = `
EXISTS (
  SELECT 1 FROM zt_demand c
  WHERE c.parent = d.id
    AND c.deleted = '0'
    AND EXISTS (
      SELECT 1 FROM zt_demandclarify dc
      WHERE dc.demand = c.id
        AND TRIM(IFNULL(dc.product, '')) != ''
    )
    AND (
      c.assignedTo = ?
      OR c.BRA = ?
      OR EXISTS (
        SELECT 1 FROM zt_demandclarify dc
        WHERE dc.demand = c.id
          AND dc.PM = ?
          AND TRIM(IFNULL(dc.product, '')) != ''
      )
    )
    AND (
      NOT EXISTS (
        SELECT 1 FROM zt_story s
        JOIN zt_planstory ps ON ps.story = s.id
        JOIN zt_versionwindowproduct vwp ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
        WHERE s.fromDemand = c.id
          AND s.deleted = '0'
          AND s.sourceType = 'demandpool'
          AND s.type = 'story'
      )
      OR NOT EXISTS (
        SELECT 1 FROM zt_story s
        WHERE s.fromDemand = c.id
          AND s.deleted = '0'
          AND s.sourceType = 'demandpool'
          AND s.type = 'story'
      )
      OR EXISTS (
        SELECT 1 FROM zt_story s
        WHERE s.fromDemand = c.id
          AND s.deleted = '0'
          AND s.sourceType = 'demandpool'
          AND s.type = 'story'
          AND (
            NOT EXISTS (SELECT 1 FROM zt_task t WHERE t.story = s.id AND t.deleted = '0' AND t.status != 'closed')
            OR EXISTS (
              SELECT 1 FROM zt_task t
              WHERE t.story = s.id AND t.deleted = '0' AND t.status != 'closed'
                AND (t.assignedTo = '' OR t.assignedTo IS NULL)
            )
          )
      )
    )
)`

const bizDemandUnscheduledSQL = `
AND (
  (
    NOT ` + bizDemandHasChildSQL + `
    AND ` + bizDemandUnscheduledSelfSQL + `
  )
  OR ` + bizDemandUnscheduledChildSQL + `
)`

const bizDemandUnassignedSQL = `
AND EXISTS (
  SELECT 1 FROM zt_story s WHERE s.fromDemand = d.id AND s.deleted = '0'
  AND (
    NOT EXISTS (SELECT 1 FROM zt_task t WHERE t.story = s.id AND t.deleted = '0' AND t.status != 'closed')
    OR EXISTS (
      SELECT 1 FROM zt_task t
      WHERE t.story = s.id AND t.deleted = '0' AND t.status != 'closed'
        AND (t.assignedTo = '' OR t.assignedTo IS NULL)
    )
  )
)`

const indepStoryUnscheduledSQL = `
AND (
  s.assignedTo = ?
  OR EXISTS (
    SELECT 1 FROM zt_product p
    WHERE p.id = s.product AND p.deleted = '0'
      AND (p.PO = ? OR p.QD = ? OR p.RD = ?)
  )
)
AND (
  NOT EXISTS (
    SELECT 1 FROM zt_planstory ps
    JOIN zt_versionwindowproduct vwp ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
    WHERE ps.story = s.id
  )
  OR NOT EXISTS (SELECT 1 FROM zt_task t WHERE t.story = s.id AND t.deleted = '0' AND t.status != 'closed')
  OR EXISTS (
    SELECT 1 FROM zt_task t
    WHERE t.story = s.id AND t.deleted = '0' AND t.status != 'closed'
      AND (t.assignedTo = '' OR t.assignedTo IS NULL)
  )
)`

const indepStoryUnassignedSQL = `
AND (
  NOT EXISTS (SELECT 1 FROM zt_task t WHERE t.story = s.id AND t.deleted = '0' AND t.status != 'closed')
  OR EXISTS (
    SELECT 1 FROM zt_task t
    WHERE t.story = s.id AND t.deleted = '0' AND t.status != 'closed'
      AND (t.assignedTo = '' OR t.assignedTo IS NULL)
  )
)`

func buildBizDemandFilterClause(filter, account string) filterClause {
	filter = NormalizeDemandFilter(filter)
	account = strings.TrimSpace(account)
	switch filter {
	case FilterUnscheduled:
		return filterClause{
			sql:  bizDemandExcludeReleasedSQL + "\n" + bizDemandUnscheduledExcludeHangSQL + bizDemandUnscheduledSQL,
			args: []interface{}{account, account, account, account, account, account},
		}
	case FilterPendingReview:
		return filterClause{sql: "AND d.status = 'wait'"}
	case FilterUnassigned:
		return filterClause{
			sql: bizDemandExcludeReleasedSQL + "\n" + bizDemandUnassignedSQL,
		}
	case FilterManagerReviewing:
		return filterClause{sql: "AND d.isManagerReview = 'reviewing'"}
	case FilterClosed:
		return filterClause{sql: "AND d.status = 'closed'"}
	default:
		return filterClause{sql: bizDemandAllOpenSQL}
	}
}

func applyBizDemandSuspended(clause filterClause, suspended bool) filterClause {
	if !suspended {
		return clause
	}
	clause.sql += bizDemandHangSuspendedSQL
	return clause
}

func buildIndepStoryFilterClause(filter, account string) filterClause {
	filter = NormalizeDemandFilter(filter)
	account = strings.TrimSpace(account)
	switch filter {
	case FilterUnscheduled:
		return filterClause{
			sql:  indepStoryExcludeReleasedSQL + "\n" + indepStoryUnscheduledSQL,
			args: []interface{}{account, account, account, account},
		}
	case FilterPendingReview:
		return filterClause{sql: "AND s.status = 'reviewing'"}
	case FilterUnassigned:
		return filterClause{
			sql: indepStoryExcludeReleasedSQL + "\n" + indepStoryUnassignedSQL,
		}
	case FilterManagerReviewing:
		return filterClause{sql: "AND 1 = 0"}
	case FilterClosed:
		return filterClause{sql: "AND s.status = 'closed'"}
	default:
		return filterClause{sql: indepStoryAllOpenSQL}
	}
}

// GetBizDemandFilterCounts 统计业务需求各快捷筛选项数量。
func (r *Repo) GetBizDemandFilterCounts(ctx context.Context, poolIDs []uint, account, activeFilter string) (FilterCounts, error) {
	if len(poolIDs) == 0 {
		return FilterCounts{}, nil
	}

	const simpleQuery = `
SELECT
  SUM(CASE WHEN status != 'closed' AND status != 'released' THEN 1 ELSE 0 END) AS all_open,
  SUM(CASE WHEN status = 'wait' THEN 1 ELSE 0 END) AS pending_review,
  SUM(CASE WHEN isManagerReview = 'reviewing' THEN 1 ELSE 0 END) AS manager_reviewing,
  SUM(CASE WHEN status = 'closed' THEN 1 ELSE 0 END) AS closed
FROM zt_demand
WHERE deleted = '0' AND parent IN (0, -1) AND pool IN ?`

	var row bizDemandSimpleCountRow
	if err := r.db.WithContext(ctx).Raw(simpleQuery, poolIDs).Scan(&row).Error; err != nil {
		return FilterCounts{}, err
	}

	unscheduled, err := r.countBizDemandsWithFilter(ctx, poolIDs, account, FilterUnscheduled, false)
	if err != nil {
		return FilterCounts{}, err
	}
	unassigned, err := r.countBizDemandsWithFilter(ctx, poolIDs, account, FilterUnassigned, false)
	if err != nil {
		return FilterCounts{}, err
	}
	suspended, err := r.countBizDemandsWithFilter(ctx, poolIDs, account, activeFilter, true)
	if err != nil {
		return FilterCounts{}, err
	}

	return FilterCounts{
		AllOpen:          row.AllOpen,
		Unscheduled:      unscheduled,
		PendingReview:    row.PendingReview,
		Unassigned:       unassigned,
		ManagerReviewing: row.ManagerReviewing,
		Closed:           row.Closed,
		Suspended:        suspended,
	}, nil
}

func (r *Repo) countBizDemandsWithFilter(ctx context.Context, poolIDs []uint, account, filter string, suspended bool) (int64, error) {
	clause := applyBizDemandSuspended(buildBizDemandFilterClause(filter, account), suspended)
	const countQuery = `
SELECT COUNT(*) AS total
FROM zt_demand d
WHERE d.deleted = '0'
  AND d.parent IN (0, -1)
  AND d.pool IN ?`

	args := append([]interface{}{poolIDs}, clause.args...)
	query := countQuery + clause.sql

	var total int64
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// FindDemandClarifyPMMatches 查询当前用户作为系统需求分析人员的业务需求 ID。
func (r *Repo) FindDemandClarifyPMMatches(ctx context.Context, demandIDs []uint, account string) (map[uint]bool, error) {
	account = strings.TrimSpace(account)
	if len(demandIDs) == 0 || account == "" {
		return map[uint]bool{}, nil
	}

	const query = `
SELECT DISTINCT demand
FROM zt_demandclarify
WHERE demand IN ?
  AND PM = ?
  AND TRIM(IFNULL(product, '')) != ''`

	var ids []uint
	if err := r.db.WithContext(ctx).Raw(query, demandIDs, account).Scan(&ids).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]bool, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		out[id] = true
	}
	return out, nil
}

// GetIndependentFilterCounts 统计独立研发需求各快捷筛选项数量。
func (r *Repo) GetIndependentFilterCounts(ctx context.Context, productIDs []uint, account string) (FilterCounts, error) {
	if len(productIDs) == 0 {
		return FilterCounts{}, nil
	}

	const simpleQuery = `
SELECT
  SUM(CASE WHEN s.status != 'closed' AND s.status != 'released' THEN 1 ELSE 0 END) AS all_open,
  SUM(CASE WHEN s.status = 'reviewing' THEN 1 ELSE 0 END) AS pending_review,
  SUM(CASE WHEN s.status = 'closed' THEN 1 ELSE 0 END) AS closed
FROM zt_story s
WHERE IFNULL(s.sourceType, '') != 'demandpool'
  AND s.parent = 0
  AND s.type = 'story'
  AND s.deleted = '0'
  AND s.product IN ?`

	var row indepStorySimpleCountRow
	if err := r.db.WithContext(ctx).Raw(simpleQuery, productIDs).Scan(&row).Error; err != nil {
		return FilterCounts{}, err
	}

	unscheduled, err := r.countIndepStoriesWithFilter(ctx, productIDs, account, FilterUnscheduled)
	if err != nil {
		return FilterCounts{}, err
	}
	unassigned, err := r.countIndepStoriesWithFilter(ctx, productIDs, account, FilterUnassigned)
	if err != nil {
		return FilterCounts{}, err
	}

	return FilterCounts{
		AllOpen:          row.AllOpen,
		Unscheduled:      unscheduled,
		PendingReview:    row.PendingReview,
		Unassigned:       unassigned,
		ManagerReviewing: 0,
		Closed:           row.Closed,
	}, nil
}

func (r *Repo) countIndepStoriesWithFilter(ctx context.Context, productIDs []uint, account, filter string) (int64, error) {
	clause := buildIndepStoryFilterClause(filter, account)
	const countQuery = `
SELECT COUNT(*) AS total
FROM zt_story s
WHERE IFNULL(s.sourceType, '') != 'demandpool'
  AND s.parent = 0
  AND s.type = 'story'
  AND s.deleted = '0'
  AND s.product IN ?`

	args := append([]interface{}{productIDs}, clause.args...)
	query := countQuery + clause.sql

	var total int64
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}
