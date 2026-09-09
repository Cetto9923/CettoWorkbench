package po

// currentHandlerDemandWhere mirrors DeriveCurrentHandler for rows that have a real current person.
// “待确认”和“待分配”不是账号，因而不会命中“我处理”。
func currentHandlerDemandWhere(account string) (string, []interface{}) {
	return `(
		(status = 'wait' AND (assignedTo = ? OR (status = 'wait' AND EXISTS (
			SELECT 1 FROM zt_demandreview dr
			WHERE dr.demand = zt_demand.id AND dr.reviewer = ? AND (dr.result IS NULL OR dr.result = '')
		))))
		OR (status IN ('draft', 'refuse') AND (assignedTo = ? OR createdBy = ?))
		OR (status = 'active' AND (
			EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id AND FIND_IN_SET(?, REPLACE(dc.PM, ' ', '')) > 0)
			OR (NOT EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id AND dc.PM IS NOT NULL AND dc.PM <> '') AND assignedTo = ?)
			OR (NOT EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id AND dc.PM IS NOT NULL AND dc.PM <> '') AND (assignedTo IS NULL OR assignedTo = '') AND QD = ?)
		))
		OR (status = 'clarified' AND (assignedTo = ? OR ((assignedTo IS NULL OR assignedTo = '') AND QD = ?)))
		OR (status IN ('developing', 'testing', 'waitacceptance') AND RD = ?)
	)`, []interface{}{account, account, account, account, account, account, account, account, account, account}
}

// currentHandlerDemandWhereWithReviews 优化版本：利用已预查的评审 demand ID 列表替换相关子查询，
// 避免对大表 zt_demandreview 执行逐行全表扫描。当 reviewDemandIDs 为 nil 时退化为原子查询实现。
func currentHandlerDemandWhereWithReviews(account string, reviewDemandIDs []int) (string, []interface{}) {
	if reviewDemandIDs == nil {
		return currentHandlerDemandWhere(account)
	}
	if len(reviewDemandIDs) == 0 {
		return `(
			(status = 'wait' AND assignedTo = ?)
			OR (status IN ('draft', 'refuse') AND (assignedTo = ? OR createdBy = ?))
			OR (status = 'active' AND (
				EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id AND FIND_IN_SET(?, REPLACE(dc.PM, ' ', '')) > 0)
				OR (NOT EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id AND dc.PM IS NOT NULL AND dc.PM <> '') AND assignedTo = ?)
				OR (NOT EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id AND dc.PM IS NOT NULL AND dc.PM <> '') AND (assignedTo IS NULL OR assignedTo = '') AND QD = ?)
			))
			OR (status = 'clarified' AND (assignedTo = ? OR ((assignedTo IS NULL OR assignedTo = '') AND QD = ?)))
			OR (status IN ('developing', 'testing', 'waitacceptance') AND RD = ?)
		)`, []interface{}{account, account, account, account, account, account, account, account, account}
	}
	return `(
		(status = 'wait' AND (assignedTo = ? OR id IN (?)))
		OR (status IN ('draft', 'refuse') AND (assignedTo = ? OR createdBy = ?))
		OR (status = 'active' AND (
			EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id AND FIND_IN_SET(?, REPLACE(dc.PM, ' ', '')) > 0)
			OR (NOT EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id AND dc.PM IS NOT NULL AND dc.PM <> '') AND assignedTo = ?)
			OR (NOT EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id AND dc.PM IS NOT NULL AND dc.PM <> '') AND (assignedTo IS NULL OR assignedTo = '') AND QD = ?)
		))
		OR (status = 'clarified' AND (assignedTo = ? OR ((assignedTo IS NULL OR assignedTo = '') AND QD = ?)))
		OR (status IN ('developing', 'testing', 'waitacceptance') AND RD = ?)
	)`, []interface{}{account, reviewDemandIDs, account, account, account, account, account, account, account, account}
}

// currentHandlerDemandKeywordWhere is the keyword counterpart of currentHandlerDemandWhere.
// It deliberately excludes historical/auxiliary owners such as QD on a developing demand.
func currentHandlerDemandKeywordWhere() string {
	return `
		(status = 'wait' AND (LOWER(IFNULL(assignedTo, '')) LIKE ? OR (status = 'wait' AND EXISTS (
			SELECT 1 FROM zt_demandreview dr
			WHERE dr.demand = zt_demand.id AND LOWER(IFNULL(dr.reviewer, '')) LIKE ? AND (dr.result IS NULL OR dr.result = '')
		))))
		OR (status IN ('draft', 'refuse') AND (LOWER(IFNULL(assignedTo, '')) LIKE ? OR LOWER(IFNULL(createdBy, '')) LIKE ?))
		OR (status = 'active' AND (
			EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id AND LOWER(IFNULL(dc.PM, '')) LIKE ?)
			OR (NOT EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id AND dc.PM IS NOT NULL AND dc.PM <> '') AND LOWER(IFNULL(assignedTo, '')) LIKE ?)
			OR (NOT EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id AND dc.PM IS NOT NULL AND dc.PM <> '') AND (assignedTo IS NULL OR assignedTo = '') AND LOWER(IFNULL(QD, '')) LIKE ?)
		))
		OR (status = 'clarified' AND (LOWER(IFNULL(assignedTo, '')) LIKE ? OR ((assignedTo IS NULL OR assignedTo = '') AND LOWER(IFNULL(QD, '')) LIKE ?)))
		OR (status IN ('developing', 'testing', 'waitacceptance') AND LOWER(IFNULL(RD, '')) LIKE ?)`
}
