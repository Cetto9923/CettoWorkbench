// =============================================================================
// 文件: internal/module/po/repodone_url.go
// 模块: PO 工作台
// 类型: repo
// 职责: 我的已办对象 → 禅道详情页链接映射。从 repodone.go 拆出，使其回到
//       500 行硬上限之内，并把"对象类型 → 链接参数"这一项职责集中到一处。
// 依赖: internal/pkg/zentao
// =============================================================================

package po

import (
	"fmt"

	"workbench/internal/pkg/zentao"
)

// objectViewURL 按 zentao 对象类型拼详情页链接。
// 对 charter / buildguideline 等审批对象，使用对象自身 ID 作为首选参数，
// 与禅道 max5 实际链接 m=<m>&f=view&id=N 一致；缺失时回退到 projectID。
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
	case "buildguideline":
		return zentao.BuildguidelineViewURL(id, 0)
	case "planchange":
		return zentao.PlanchangeViewURL(id)
	case "review":
		return zentao.ReviewViewURL(id)
	}
	return ""
}

// objectViewURLWithProject 按 zentao 对象类型拼详情页链接；审批对象可携带 projectID 回退。
// zt_action.objectID 实测非 0，因此正常路径走 id={objectID}；projectID 仅在对象自身
// ID 缺失时兜底，两者都为 0 时返回空串，由前端渲染不可点标题而不是跳禅道首页。
func objectViewURLWithProject(objectType string, objectID, projectID uint) string {
	if objectType == "charter" {
		return zentao.CharterViewURL(objectID, projectID)
	}
	if objectType == "buildguideline" {
		return zentao.BuildguidelineViewURL(objectID, projectID)
	}
	return objectViewURL(objectType, objectID)
}
