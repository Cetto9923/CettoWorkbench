// =============================================================================
// 文件: internal/module/po/repodone_result.go
// 模块: PO 工作台
// 职责: 我的已办动作处理结果动态解析（extra -> approved/rejected/done）与下推 SQL 构造。
// =============================================================================

package po

import (
	"strings"
)

// resolveDoneActionResult 根据 action、objectType 及 zt_action.extra 动态解析真实处理结果。
// 解决评审动作（reviewed/reviewchange）在 extra 中记录 pass/refuse/reject，
// 而不是静态配置的 "done" 的问题，确保已通过与已驳回精准区分。
func resolveDoneActionResult(action, objectType, extra, defaultResult string) (string, string) {
	if (action == "reviewed" || action == "reviewchange") && (objectType == "demand" || objectType == "story") {
		ex := strings.ToLower(strings.TrimSpace(extra))
		if strings.Contains(ex, "pass") {
			return "approved", doneResultText("approved")
		}
		if strings.Contains(ex, "refuse") || strings.Contains(ex, "reject") {
			return "rejected", doneResultText("rejected")
		}
	}
	res := defaultResult
	if res == "" {
		res = "done"
	}
	return res, doneResultText(res)
}

// buildDoneResultFilterSQL 构造结果筛选 SQL：
// 当筛选 approved 时，除了静态 approved 动作外，还将 extra 命中 pass 的评审动作一并拉出；
// 当筛选 rejected 时，除了静态 rejected 动作外，还将 extra 命中 refuse/reject 的评审动作拉出；
// 当筛选 done 时，排除带有明确 pass/refuse/reject 决策结果的评审动作。
func buildDoneResultFilterSQL(result string) (string, []interface{}) {
	switch result {
	case "approved":
		staticCodes := codesWithResult("approved")
		return "(a.action IN ? OR ((a.objectType IN ('demand', 'story') AND a.action = 'reviewed' OR (a.objectType = 'demand' AND a.action = 'reviewchange')) AND LOWER(a.extra) LIKE '%pass%'))",
			[]interface{}{staticCodes}
	case "rejected":
		staticCodes := codesWithResult("rejected")
		return "(a.action IN ? OR ((a.objectType IN ('demand', 'story') AND a.action = 'reviewed' OR (a.objectType = 'demand' AND a.action = 'reviewchange')) AND (LOWER(a.extra) LIKE '%refuse%' OR LOWER(a.extra) LIKE '%reject%')))",
			[]interface{}{staticCodes}
	case "done":
		staticCodes := codesWithResult("done")
		return "(a.action IN ? AND NOT ((a.objectType IN ('demand', 'story') AND a.action = 'reviewed' OR (a.objectType = 'demand' AND a.action = 'reviewchange')) AND (LOWER(a.extra) LIKE '%pass%' OR LOWER(a.extra) LIKE '%refuse%' OR LOWER(a.extra) LIKE '%reject%')))",
			[]interface{}{staticCodes}
	default:
		return "a.action IN ?", []interface{}{codesWithResult(result)}
	}
}
