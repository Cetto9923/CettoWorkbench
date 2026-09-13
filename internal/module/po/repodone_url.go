// =============================================================================
// 文件: internal/module/po/repodone_url.go
// 模块: PO 工作台
// 类型: repo
// 职责: 我的已办对象 → 禅道详情页链接映射。
// 依赖: internal/pkg/zentao
// =============================================================================

package po

import (
	"fmt"

	"workbench/internal/pkg/zentao"
)

// objectViewURL 按 zentao 对象类型拼详情页链接。
// 对 charter / buildguideline 等审批对象，入口由 objectViewURLWithProject
// 按所属项目生成；无项目上下文时不生成可能导致白屏的伪详情链接。
func objectViewURL(objectType string, id uint) string {
	if id == 0 {
		return ""
	}
	switch objectType {
	case "demand":
		return zentao.DemandViewURL(id)
	case "story":
		return zentao.StoryViewURL(id)
	case "task":
		return zentao.TaskViewURL(id)
	case "bug":
		return zentao.BugViewURL(id)
	case "testtask":
		return zentao.TesttaskViewURL(id)
	case "risk", "issue", "feedback", "release", "build", "todo":
		return zentao.URL(objectType, "view", fmt.Sprintf("%sID=%d", objectType, id))
	case "case":
		return zentao.CaseViewURL(id)
	case "charter":
		return zentao.CharterViewURL(id, 0)
	case "buildguideline", "guideline":
		return zentao.BuildguidelineViewURL(id, 0)
	case "planchange":
		return zentao.PlanchangeViewURL(id)
	case "review":
		return zentao.ReviewViewURL(id)
	}
	return ""
}

// objectViewURLWithProject 按 zentao 对象类型拼详情页链接；章程和建设指引必须携带 projectID。
func objectViewURLWithProject(objectType string, objectID, projectID uint) string {
	if objectType == "charter" {
		return zentao.CharterViewURL(objectID, projectID)
	}
	if objectType == "buildguideline" {
		return zentao.BuildguidelineViewURL(objectID, projectID)
	}
	return objectViewURL(objectType, objectID)
}
