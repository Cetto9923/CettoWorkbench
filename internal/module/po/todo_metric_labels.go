// =============================================================================
// 文件: internal/module/po/todo_metric_labels.go
// 模块: PO 工作台
// 类型: service/repo helper
// 职责: 待办对象的领域指标与审批场景展示标签。
// =============================================================================

package po

import "strings"

func issueRiskSeverityForTodo(kind, raw string) string {
	if kind == "risk" {
		return todoRiskImpactLabel(raw)
	}
	if kind != "issue" {
		return ""
	}
	return issueRiskSeverity(raw)
}

func todoRiskImpactLabel(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "low":
		return "低"
	case "2":
		return "较低"
	case "3", "middle", "medium":
		return "中"
	case "4", "high":
		return "较高"
	case "5", "urgent", "immediate":
		return "高"
	default:
		return "—"
	}
}

func todoApprovalSceneLabel(objectType string) string {
	switch objectType {
	case "charter":
		return "项目章程"
	case "buildguideline":
		return "建设指引"
	case "planchange":
		return "计划变更"
	case "review":
		return "需求评审"
	case "reviewchange":
		return "需求变更"
	case "reviewbymanager":
		return "主管审批"
	default:
		return "—"
	}
}
