package po

// currentHandlerDemandWhere returns the stage-specific homepage "my action"
// predicate. The same account can be related to a demand in several ways, but
// only the role that is actionable in the current value-stream stage matches.
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
		OR (status = 'developing' AND BRA = ?)
		OR (status = 'testing' AND (QD = ? OR (accepter = ? AND ` + dateSetBeforeTodaySQL("testFinish") + `)))
		OR (status = 'waitacceptance' AND accepter = ?)
		OR (status IN ('acceptanced', 'waitdeliver') AND BRA = ?)
		OR (status = 'released' AND (originator = ? OR BRA = ?))
	)`, []interface{}{account, account, account, account, account, account, account, account, account, account, account, account, account, account, account}
}

// currentHandlerDemandWhereWithReviews 优化版本：利用已预查的评审 demand ID 列表替换相关子查询，
// 避免对大表 zt_demandreview 执行逐行全表扫描。当 reviewDemandIDs 为 nil 时退化为原子查询实现。
func currentHandlerDemandWhereWithReviews(account string, reviewDemandIDs []int) (string, []interface{}) {
	if reviewDemandIDs == nil {
		return currentHandlerDemandWhere(account)
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
		OR (status = 'developing' AND BRA = ?)
		OR (status = 'testing' AND (QD = ? OR (accepter = ? AND ` + dateSetBeforeTodaySQL("testFinish") + `)))
		OR (status = 'waitacceptance' AND accepter = ?)
		OR (status IN ('acceptanced', 'waitdeliver') AND BRA = ?)
		OR (status = 'released' AND (originator = ? OR BRA = ?))
	)`, []interface{}{account, reviewDemandIDs, account, account, account, account, account, account, account, account, account, account, account, account, account}
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
		OR (status = 'developing' AND LOWER(IFNULL(BRA, '')) LIKE ?)
		OR (status = 'testing' AND (LOWER(IFNULL(QD, '')) LIKE ? OR LOWER(IFNULL(accepter, '')) LIKE ?))
		OR (status = 'waitacceptance' AND LOWER(IFNULL(accepter, '')) LIKE ?)
		OR (status IN ('acceptanced', 'waitdeliver') AND LOWER(IFNULL(BRA, '')) LIKE ?)
		OR (status = 'released' AND (LOWER(IFNULL(originator, '')) LIKE ? OR LOWER(IFNULL(BRA, '')) LIKE ?))`
}

// dateSetBeforeTodaySQL avoids a zero-date literal, which can fail under
// MySQL's NO_ZERO_DATE mode while still accepting ZenTao's historical values.
func dateSetBeforeTodaySQL(column string) string {
	return column + " IS NOT NULL AND CAST(" + column + " AS CHAR) NOT LIKE '0000-00-00%' AND " + column + " <= CURDATE()"
}
