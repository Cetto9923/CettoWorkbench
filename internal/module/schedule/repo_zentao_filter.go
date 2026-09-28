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
	AllOpen int64 `gorm:"column:all_open"`
	Closed  int64 `gorm:"column:closed"`
}

const bizDemandHangSuspendedSQL = ` AND d.hang = '1'`

const bizDemandAllOpenSQL = `AND d.status != 'closed' AND d.status != 'released'`

const bizDemandUnscheduledExcludeHangSQL = `AND d.hang = '0'`

const indepStoryAllOpenSQL = `AND s.status != 'closed' AND s.status != 'released'`

type indepStorySimpleCountRow struct {
	AllOpen       int64 `gorm:"column:all_open"`
	PendingReview int64 `gorm:"column:pending_review"`
	Closed        int64 `gorm:"column:closed"`
}

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

func buildBizDemandFilterClause(filter, account string) filterClause {
	filter = NormalizeDemandFilter(filter)
	account = strings.TrimSpace(account)
	switch filter {
	case FilterUnscheduled:
		// 待排期走 FindUnscheduledBizDemandTopIDs 多阶段路径，避免巨型相关子查询。
		if account == "" {
			return filterClause{sql: "AND 1 = 0"}
		}
		return filterClause{sql: bizDemandUnscheduledExcludeHangSQL}
	case FilterPendingReview:
		// 与首页价值流「受理」一致：与我相关 + draft/wait/refuse
		if account == "" {
			return filterClause{sql: "AND 1 = 0"}
		}
		return filterClause{
			sql: `AND d.status IN ('draft', 'wait', 'refuse')
AND (
  d.id IN (SELECT demand FROM zt_demandclarify WHERE PM = ?)
  OR d.QD = ?
  OR d.RD = ?
  OR d.BRA = ?
)`,
			args: []interface{}{account, account, account, account},
		}
	case FilterManagerReviewing:
		// 主管审批中 + 与我相关：指派人 / BRA / 澄清 PM（product 非空，与待排期一致）
		if account == "" {
			return filterClause{sql: "AND 1 = 0"}
		}
		return filterClause{
			sql: `AND d.isManagerReview = 'reviewing'
AND (
  d.assignedTo = ?
  OR d.BRA = ?
  OR d.id IN (
    SELECT demand FROM zt_demandclarify
    WHERE PM = ? AND TRIM(IFNULL(product, '')) != ''
  )
)`,
			args: []interface{}{account, account, account},
		}
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
		// 待排期不含已关闭/已发布；与「全部未关闭」一致排除 closed/released。
		return filterClause{
			sql:  indepStoryAllOpenSQL + "\n" + indepStoryUnscheduledSQL,
			args: []interface{}{account, account, account, account},
		}
	case FilterPendingReview:
		// 待受理仅业需（价值流受理不含独立研需）
		return filterClause{sql: "AND 1 = 0"}
	case FilterManagerReviewing:
		return filterClause{sql: "AND 1 = 0"}
	case FilterClosed:
		return filterClause{sql: "AND s.status = 'closed'"}
	default:
		return filterClause{sql: indepStoryAllOpenSQL}
	}
}

// GetBizDemandFilterCounts 统计业务需求各快捷筛选项数量。
// reuseFilter/reuseTotal：列表 total 可复用时跳过对应重 COUNT。
func (r *Repo) GetBizDemandFilterCounts(ctx context.Context, poolIDs []uint, account, activeFilter, reuseFilter string, reuseTotal int64, extraReq ...FilterCountsReq) (FilterCounts, error) {
	if len(poolIDs) == 0 {
		return FilterCounts{}, nil
	}
	var extra filterClause
	if len(extraReq) > 0 {
		params := advancedFilterParamsFromCountsReq(extraReq[0])
		extra = buildBizDemandAdvancedClause(params)
	}
	if extra.sql != "" {
		reuseFilter = ""
	}

	simpleQuery := `
SELECT
  SUM(CASE WHEN status != 'closed' AND status != 'released' THEN 1 ELSE 0 END) AS all_open,
  SUM(CASE WHEN status = 'closed' THEN 1 ELSE 0 END) AS closed
FROM zt_demand d
WHERE deleted = '0' AND parent IN (0, -1) AND pool IN ?`
	simpleArgs := []interface{}{poolIDs}
	if extra.sql != "" {
		simpleQuery += "\n" + extra.sql
		simpleArgs = append(simpleArgs, extra.args...)
	}

	var row bizDemandSimpleCountRow
	if err := r.db.WithContext(ctx).Raw(simpleQuery, simpleArgs...).Scan(&row).Error; err != nil {
		return FilterCounts{}, err
	}

	needUnscheduled := !filterCountReuseMatches(reuseFilter, FilterUnscheduled)
	needPendingReview := !filterCountReuseMatches(reuseFilter, FilterPendingReview)
	needManagerReviewing := !filterCountReuseMatches(reuseFilter, FilterManagerReviewing)
	unscheduled, pendingReview, managerReviewing, err := r.countBizDemandHeavyFilters(ctx, poolIDs, account, needUnscheduled, needPendingReview, needManagerReviewing, extra)
	if err != nil {
		return FilterCounts{}, err
	}

	var suspended int64
	if !filterCountReuseMatches(reuseFilter, FilterCountReuseSuspended) {
		suspended, err = r.countBizDemandsWithFilter(ctx, poolIDs, account, activeFilter, true, extra)
		if err != nil {
			return FilterCounts{}, err
		}
	}

	counts := FilterCounts{
		AllOpen:          row.AllOpen,
		Unscheduled:      unscheduled,
		PendingReview:    pendingReview,
		ManagerReviewing: managerReviewing,
		Closed:           row.Closed,
		Suspended:        suspended,
	}
	applyFilterCountReuse(&counts, reuseFilter, reuseTotal)
	return counts, nil
}

func (r *Repo) countBizDemandHeavyFilters(ctx context.Context, poolIDs []uint, account string, needUnscheduled, needPendingReview, needManagerReviewing bool, extra filterClause) (unscheduled, pendingReview, managerReviewing int64, err error) {
	if needUnscheduled {
		unscheduled, err = r.countBizDemandsWithFilter(ctx, poolIDs, account, FilterUnscheduled, false, extra)
		if err != nil {
			return 0, 0, 0, err
		}
	}
	if needPendingReview {
		pendingReview, err = r.countBizDemandsWithFilter(ctx, poolIDs, account, FilterPendingReview, false, extra)
		if err != nil {
			return 0, 0, 0, err
		}
	}
	if needManagerReviewing {
		managerReviewing, err = r.countBizDemandsWithFilter(ctx, poolIDs, account, FilterManagerReviewing, false, extra)
		if err != nil {
			return 0, 0, 0, err
		}
	}
	return unscheduled, pendingReview, managerReviewing, nil
}

func (r *Repo) countBizDemandsWithFilter(ctx context.Context, poolIDs []uint, account, filter string, suspended bool, extra ...filterClause) (int64, error) {
	var extraClause filterClause
	if len(extra) > 0 {
		extraClause = extra[0]
	}
	if NormalizeDemandFilter(filter) == FilterUnscheduled {
		return r.countUnscheduledBizDemands(ctx, poolIDs, account, suspended, extraClause)
	}

	clause := applyBizDemandSuspended(buildBizDemandFilterClause(filter, account), suspended)
	countQuery := `
SELECT COUNT(*) AS total
FROM zt_demand d
WHERE d.deleted = '0'
  AND d.parent IN (0, -1)
  AND d.pool IN ?`

	args := append([]interface{}{poolIDs}, clause.args...)
	query := countQuery + clause.sql
	if extraClause.sql != "" {
		query += "\n" + extraClause.sql
		args = append(args, extraClause.args...)
	}

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
func (r *Repo) GetIndependentFilterCounts(ctx context.Context, productIDs []uint, account, reuseFilter string, reuseTotal int64, extraReq ...FilterCountsReq) (FilterCounts, error) {
	if len(productIDs) == 0 {
		return FilterCounts{}, nil
	}
	var indepExtra filterClause
	if len(extraReq) > 0 {
		params := advancedFilterParamsFromCountsReq(extraReq[0])
		indepExtra = buildIndepStoryAdvancedClause(params)
	}
	if indepExtra.sql != "" {
		reuseFilter = ""
	}

	simpleQuery := `
SELECT
  SUM(CASE WHEN s.status != 'closed' AND s.status != 'released' THEN 1 ELSE 0 END) AS all_open,
  0 AS pending_review,
  SUM(CASE WHEN s.status = 'closed' THEN 1 ELSE 0 END) AS closed
FROM zt_story s
WHERE IFNULL(s.sourceType, '') != 'demandpool'
  AND s.parent = 0
  AND s.type = 'story'
  AND s.deleted = '0'
  AND s.product IN ?`
	simpleArgs := []interface{}{productIDs}
	if indepExtra.sql != "" {
		simpleQuery += "\n" + indepExtra.sql
		simpleArgs = append(simpleArgs, indepExtra.args...)
	}

	var row indepStorySimpleCountRow
	if err := r.db.WithContext(ctx).Raw(simpleQuery, simpleArgs...).Scan(&row).Error; err != nil {
		return FilterCounts{}, err
	}

	needUnscheduled := !filterCountReuseMatches(reuseFilter, FilterUnscheduled)
	unscheduled, err := r.countIndepStoryHeavyFilters(ctx, productIDs, account, needUnscheduled, indepExtra)
	if err != nil {
		return FilterCounts{}, err
	}

	counts := FilterCounts{
		AllOpen:          row.AllOpen,
		Unscheduled:      unscheduled,
		PendingReview:    row.PendingReview,
		ManagerReviewing: 0,
		Closed:           row.Closed,
	}
	applyFilterCountReuse(&counts, reuseFilter, reuseTotal)
	return counts, nil
}

func (r *Repo) countIndepStoryHeavyFilters(ctx context.Context, productIDs []uint, account string, needUnscheduled bool, extra ...filterClause) (int64, error) {
	if !needUnscheduled {
		return 0, nil
	}
	return r.countIndepStoriesWithFilter(ctx, productIDs, account, FilterUnscheduled, extra...)
}

func (r *Repo) countIndepStoriesWithFilter(ctx context.Context, productIDs []uint, account, filter string, extra ...filterClause) (int64, error) {
	clause := buildIndepStoryFilterClause(filter, account)
	countQuery := `
SELECT COUNT(*) AS total
FROM zt_story s
WHERE IFNULL(s.sourceType, '') != 'demandpool'
  AND s.parent = 0
  AND s.type = 'story'
  AND s.deleted = '0'
  AND s.product IN ?`

	args := append([]interface{}{productIDs}, clause.args...)
	query := countQuery + clause.sql
	if len(extra) > 0 && extra[0].sql != "" {
		query += "\n" + extra[0].sql
		args = append(args, extra[0].args...)
	}

	var total int64
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}
