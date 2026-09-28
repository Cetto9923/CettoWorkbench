// =============================================================================
// 文件: internal/module/po/repodone_scope.go
// 模块: PO 工作台
// 职责: 我的已办对象域与审批评审范围 SQL。
// =============================================================================

package po

// buildApprovalDoneScopeSQL 返回真正构成审批或评审决策的已办动作范围。
// 同时覆盖项目章程、计划变更、建设指引、项目评审、需求/研发需求和用例评审。
func buildApprovalDoneScopeSQL() string {
	return "((a.objectType = 'demand' AND a.action IN ('reviewed', 'reviewchange', 'reviewbymanager')) OR " +
		"(a.objectType = 'story' AND a.action IN ('submitreview', 'reviewed')) OR " +
		"(a.objectType = 'case' AND a.action = 'reviewed') OR " +
		"(a.objectType = 'charter' AND a.action = 'approvalreview') OR " +
		"(a.objectType = 'planchange' AND a.action = 'approvalreview') OR " +
		"(a.objectType = 'buildguideline' AND a.action = 'approvalreview') OR " +
		"(a.objectType = 'review' AND a.action = 'reviewed'))"
}

func buildDoneObjectScopeSQL(tab DoneTab, objectType string) (string, []interface{}) {
	if objectType != "" && objectType != "all" {
		return "a.objectType = ?", []interface{}{objectType}
	}
	switch tab {
	case DoneTabApproval:
		return buildApprovalDoneScopeSQL(), nil
	case DoneTabDemand:
		return "a.objectType IN ?", []interface{}{[]string{"demand", "story"}}
	case DoneTabExecution:
		return "a.objectType IN ?", []interface{}{[]string{"task", "build", "release"}}
	case DoneTabQuality:
		return "a.objectType IN ?", []interface{}{[]string{"bug", "testtask"}}
	case DoneTabRisks:
		return "a.objectType IN ?", []interface{}{[]string{"risk", "issue"}}
	}
	return "", nil
}
