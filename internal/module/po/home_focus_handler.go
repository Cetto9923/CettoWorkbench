// =============================================================================
// 文件: internal/module/po/home_focus_handler.go
// 模块: PO 工作台
// 类型: repo
// 职责: 首页待办办理人及关键词查询条件。
// =============================================================================
package po

import "strings"

const pendingReviewSQL = `EXISTS (SELECT 1 FROM zt_demandreview dr WHERE dr.demand = zt_demand.id AND dr.reviewer = ? AND (dr.result IS NULL OR dr.result = ''))`
const acceptanceOwnerSQL = `(RD = ? OR (IFNULL(RD, '') = '' AND assignedTo = ?))`
const latestAcceptanceActorSQL = `(SELECT a.actor FROM zt_action a WHERE a.objectType = 'demand' AND a.objectID = zt_demand.id AND a.action = 'startacceptance' ORDER BY a.id DESC LIMIT 1)`
const demandTestHandlerSQL = `EXISTS (SELECT 1 FROM zt_testtask tt JOIN zt_testrun tr ON tr.task = tt.id JOIN zt_case c ON c.id = tr.case JOIN zt_story s ON s.id = c.story AND s.deleted = '0' WHERE s.fromDemand = zt_demand.id AND tt.deleted = '0' AND (tt.owner = ? OR FIND_IN_SET(?, REPLACE(tt.members, ' ', '')) > 0))`

var developingHandlerSQL = `(status = 'developing' AND (BRA = ? OR (IFNULL(BRA, '') = '' AND assignedTo = ?)))`

func currentHandlerDemandWhere(account string) (string, []interface{}) {
	where := `(status IN ('draft', 'refuse') AND createdBy = ?)
 OR (status = 'wait' AND (createdBy = ? OR ` + pendingReviewSQL + `))
 OR (status = 'active' AND (assignedTo = ? OR BRA = ?))
 OR (status = 'clarified' AND BRA = ?)
 OR ` + developingHandlerSQL + `
 OR (status = 'testing' AND ` + demandTestHandlerSQL + `)
 OR (status = 'waitacceptance' AND (` + acceptanceOwnerSQL + ` OR (` + latestAcceptanceActorSQL + ` = ? AND NOT ` + acceptanceOwnerSQL + `)))
 OR (status = 'acceptanced' AND BRA = ?)
 OR (status = 'released' AND overall = '0' AND (originator = ? OR (IFNULL(originator, '') = '' AND createdBy = ?)))`
	args := make([]interface{}, strings.Count(where, "?"))
	for i := range args {
		args[i] = account
	}
	return "(" + where + ")", args
}

func currentHandlerDemandWhereWithReviews(account string, reviewDemandIDs []int) (string, []interface{}) {
	where, args := currentHandlerDemandWhere(account)
	if reviewDemandIDs == nil {
		return where, args
	}
	if len(reviewDemandIDs) == 0 {
		return strings.Replace(where, pendingReviewSQL, "1 = 0", 1), append(args[:2], args[3:]...)
	}
	args[2] = reviewDemandIDs
	return strings.Replace(where, pendingReviewSQL, "id IN (?)", 1), args
}

func currentHandlerDemandKeywordWhere() string {
	where, _ := currentHandlerDemandWhere("")
	for _, column := range []string{"createdBy", "dr.reviewer", "assignedTo", "BRA", "tt.owner", "RD", "originator"} {
		where = strings.ReplaceAll(where, column+" = ?", "LOWER(IFNULL("+column+", '')) LIKE ?")
	}
	where = strings.ReplaceAll(where, latestAcceptanceActorSQL+" = ?", "LOWER(IFNULL("+latestAcceptanceActorSQL+", '')) LIKE ?")
	return strings.ReplaceAll(where, "FIND_IN_SET(?, REPLACE(tt.members, ' ', '')) > 0", "LOWER(IFNULL(tt.members, '')) LIKE ?")
}

func dateSetBeforeTodaySQL(column string) string {
	return column + " IS NOT NULL AND CAST(" + column + " AS CHAR) NOT LIKE '0000-00-00%' AND " + column + " <= CURDATE()"
}
