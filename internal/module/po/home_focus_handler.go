package po

// currentHandlerDemandWhere returns the stage-specific homepage "my action" predicate: only the role actionable in the current value-stream stage matches.
func currentHandlerDemandWhere(account string) (string, []interface{}) {
	return `(
		(status IN ('draft', 'refuse') AND createdBy = ?)
		OR (status = 'wait' AND EXISTS (
			SELECT 1 FROM zt_demandreview dr
			WHERE dr.demand = zt_demand.id AND dr.reviewer = ? AND (dr.result IS NULL OR dr.result = '')
		))
		OR (status = 'active' AND (
			assignedTo = ? OR BRA = ? OR EXISTS (
				SELECT 1 FROM zt_demandclarify dc
				WHERE dc.demand = zt_demand.id AND FIND_IN_SET(?, REPLACE(dc.PM, ' ', '')) > 0
			)
		))
		OR (status = 'clarified' AND (
			assignedTo = ? OR BRA = ? OR EXISTS (
				SELECT 1 FROM zt_demandclarify dc
				WHERE dc.demand = zt_demand.id AND FIND_IN_SET(?, REPLACE(dc.PM, ' ', '')) > 0
			)
		))
		OR ` + developingHandlerSQL + `
		OR (status = 'testing' AND (QD = ? OR (accepter = ? AND ` + dateSetBeforeTodaySQL("testFinish") + `)))
		OR (status = 'waitacceptance' AND accepter = ?)
		OR (status IN ('acceptanced', 'waitdeliver') AND BRA = ?)
		OR (status = 'released' AND (originator = ? OR BRA = ?))
	)`, []interface{}{account, account, account, account, account, account, account, account, account, account, account, account, account, account, account, account}
}

// currentHandlerDemandWhereWithReviews 用已预查的评审 demand ID 列表替换相关子查询，避免对大表
// zt_demandreview 逐行全表扫描；reviewDemandIDs 为 nil 时退化为原子查询实现，为空切片时 wait 置 (1 = 0)。
func currentHandlerDemandWhereWithReviews(account string, reviewDemandIDs []int) (string, []interface{}) {
	if reviewDemandIDs == nil {
		return currentHandlerDemandWhere(account)
	}
	if len(reviewDemandIDs) == 0 {
		return `(
		(status IN ('draft', 'refuse') AND createdBy = ?)
		OR (status = 'wait' AND 1 = 0)
		OR (status = 'active' AND (
			assignedTo = ? OR BRA = ? OR EXISTS (
				SELECT 1 FROM zt_demandclarify dc
				WHERE dc.demand = zt_demand.id AND FIND_IN_SET(?, REPLACE(dc.PM, ' ', '')) > 0
			)
		))
		OR (status = 'clarified' AND (
			assignedTo = ? OR BRA = ? OR EXISTS (
				SELECT 1 FROM zt_demandclarify dc
				WHERE dc.demand = zt_demand.id AND FIND_IN_SET(?, REPLACE(dc.PM, ' ', '')) > 0
			)
		))
		OR ` + developingHandlerSQL + `
		OR (status = 'testing' AND (QD = ? OR (accepter = ? AND ` + dateSetBeforeTodaySQL("testFinish") + `)))
		OR (status = 'waitacceptance' AND accepter = ?)
		OR (status IN ('acceptanced', 'waitdeliver') AND BRA = ?)
		OR (status = 'released' AND (originator = ? OR BRA = ?))
	)`, []interface{}{account, account, account, account, account, account, account, account, account, account, account, account, account, account, account}
	}
	return `(
		(status IN ('draft', 'refuse') AND createdBy = ?)
		OR (status = 'wait' AND id IN (?))
		OR (status = 'active' AND (
			assignedTo = ? OR BRA = ? OR EXISTS (
				SELECT 1 FROM zt_demandclarify dc
				WHERE dc.demand = zt_demand.id AND FIND_IN_SET(?, REPLACE(dc.PM, ' ', '')) > 0
			)
		))
		OR (status = 'clarified' AND (
			assignedTo = ? OR BRA = ? OR EXISTS (
				SELECT 1 FROM zt_demandclarify dc
				WHERE dc.demand = zt_demand.id AND FIND_IN_SET(?, REPLACE(dc.PM, ' ', '')) > 0
			)
		))
		OR ` + developingHandlerSQL + `
		OR (status = 'testing' AND (QD = ? OR (accepter = ? AND ` + dateSetBeforeTodaySQL("testFinish") + `)))
		OR (status = 'waitacceptance' AND accepter = ?)
		OR (status IN ('acceptanced', 'waitdeliver') AND BRA = ?)
		OR (status = 'released' AND (originator = ? OR BRA = ?))
	)`, []interface{}{account, reviewDemandIDs, account, account, account, account, account, account, account, account, account, account, account, account, account, account}
}

// currentHandlerDemandKeywordWhere is the keyword counterpart of the same
// stage-specific predicate. It keeps the homepage "当前负责人" search in sync
// with the actual "待我处理" role mapping.
func currentHandlerDemandKeywordWhere() string {
	return `
		(status IN ('draft', 'refuse') AND LOWER(IFNULL(createdBy, '')) LIKE ?)
		OR (status = 'wait' AND EXISTS (
			SELECT 1 FROM zt_demandreview dr
			WHERE dr.demand = zt_demand.id AND LOWER(IFNULL(dr.reviewer, '')) LIKE ? AND (dr.result IS NULL OR dr.result = '')
		))
		OR (status = 'active' AND (
			LOWER(IFNULL(assignedTo, '')) LIKE ? OR LOWER(IFNULL(BRA, '')) LIKE ? OR EXISTS (
				SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id AND LOWER(IFNULL(dc.PM, '')) LIKE ?
			)
		))
		OR (status = 'clarified' AND (
			LOWER(IFNULL(assignedTo, '')) LIKE ? OR LOWER(IFNULL(BRA, '')) LIKE ? OR EXISTS (
				SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id AND LOWER(IFNULL(dc.PM, '')) LIKE ?
			)
		))
		OR (status = 'developing' AND (LOWER(IFNULL(BRA, '')) LIKE ? OR ((BRA IS NULL OR BRA = '') AND LOWER(IFNULL(assignedTo, '')) LIKE ? AND ` + dateSetBeforeTodaySQL("developFinish") + `)))
		OR (status = 'testing' AND (LOWER(IFNULL(QD, '')) LIKE ? OR LOWER(IFNULL(accepter, '')) LIKE ?))
		OR (status = 'waitacceptance' AND LOWER(IFNULL(accepter, '')) LIKE ?)
		OR (status IN ('acceptanced', 'waitdeliver') AND LOWER(IFNULL(BRA, '')) LIKE ?)
		OR (status = 'released' AND (LOWER(IFNULL(originator, '')) LIKE ? OR LOWER(IFNULL(BRA, '')) LIKE ?))`
}

// dateSetBeforeTodaySQL avoids a zero-date literal (MySQL NO_ZERO_DATE) while accepting ZenTao's historical values.
func dateSetBeforeTodaySQL(column string) string {
	return column + " IS NOT NULL AND CAST(" + column + " AS CHAR) NOT LIKE '0000-00-00%' AND " + column + " <= CURDATE()"
}

// developingHandlerSQL 提测办理人，与 DeriveCurrentHandler 同优先级：BRA 优先；BRA 空时由负责人承担，且须 developFinish 已到。
var developingHandlerSQL = `(status = 'developing' AND (BRA = ? OR ((BRA IS NULL OR BRA = '') AND assignedTo = ? AND ` + dateSetBeforeTodaySQL("developFinish") + `)))`
