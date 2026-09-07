// =============================================================================
// 文件: internal/module/po/primaryaction/primaryaction_url.go
// 模块: PO 工作台
// 类型: contract
// 职责: primaryAction 各阶段的 URL 拼接。
//       内部操作走相对路径；测试单为禅道外部链接，使用 TesttaskViewURL。
// 依赖: internal/pkg/zentao
// =============================================================================

package primaryaction

import (
	"fmt"

	"workbench/internal/pkg/zentao"
)

// input 是 Derive 用的事实快照的最小子集；URL helper 只需 ObjectID + Kind。
type inputShape interface {
	getObjectID() uint
	getKind() ObjectKind
}

// URL helpers 接收一个轻量 input struct（避免循环依赖）。
//
// 用 struct 直接传参而不是引用 Input，避免 URL 派生逻辑被未来的能力字段污染。

// AcceptURL 受理 / 审批 POST 端点（前端按 modal 形态提交）。
// §4-1 当前未配置独立 endpoint，URL 仅作 contract 占位。
func AcceptURL(objectID uint, kind ObjectKind) string {
	return fmt.Sprintf("/demands/%d/accept", objectID)
}

// ClarifyURL 澄清办理页 — 内部页（直接走需求详情页 + requirement tab）。
func ClarifyURL(objectID uint, kind ObjectKind) string {
	return fmt.Sprintf("/demands/%d/detail?tab=requirement#clarificationSection", objectID)
}

// ScheduleURL 排期入口；业务需求 vs 独立研发需求走不同路由。
func ScheduleURL(objectID uint, kind ObjectKind) string {
	if kind == ObjectStory || kind == ObjectIndependentStory {
		return fmt.Sprintf("/schedule/stories/%d/scheduling", objectID)
	}
	return fmt.Sprintf("/schedule/demands/%d/scheduling", objectID)
}

// SubmitTestURL 提测办理端点。
// §4-2 阻塞：当前无 endpoint，URL 仅作 contract 占位。
func SubmitTestURL(objectID uint, kind ObjectKind) string {
	if kind == ObjectStory || kind == ObjectIndependentStory {
		return fmt.Sprintf("/demands/%d/submit-test?kind=story", objectID)
	}
	return fmt.Sprintf("/demands/%d/submit-test", objectID)
}

// AcceptDoneURL 验收端点（本人验收人）。
// §4-2 阻塞：当前无 endpoint。
func AcceptDoneURL(objectID uint, kind ObjectKind) string {
	return fmt.Sprintf("/demands/%d/accept-done", objectID)
}

// UrgeAcceptURL 催办验收端点（非验收人）。
// §4-2 阻塞：当前无 endpoint。
func UrgeAcceptURL(objectID uint, kind ObjectKind) string {
	return fmt.Sprintf("/demands/%d/urge-accept", objectID)
}

// DeliverURL 发起交付端点。
// §4-2 阻塞：当前无 endpoint。
func DeliverURL(objectID uint, kind ObjectKind) string {
	return fmt.Sprintf("/demands/%d/deliver", objectID)
}

// EvaluateURL 评价端点。
// §4-2 阻塞：当前无 endpoint。
func EvaluateURL(objectID uint, kind ObjectKind) string {
	return fmt.Sprintf("/demands/%d/evaluate", objectID)
}

// ViewEvaluateURL 查看历史评价 — 内部页（需求详情 history tab）。
func ViewEvaluateURL(objectID uint, kind ObjectKind) string {
	return fmt.Sprintf("/demands/%d/detail?tab=history#historySection", objectID)
}

// TesttaskURL 单个测试单禅道 URL（由 Repo 提供的真实 FirstTestURL 优先）。
// 这里提供 fallback 给上层 Derive 调用。
func TesttaskURL(testtaskID uint) string {
	if testtaskID == 0 {
		return ""
	}
	return zentao.TesttaskViewURL(testtaskID)
}

// acceptURL 包装 AcceptURL — 与 Derive Input 内部使用一致。
func acceptURL(in Input) string {
	return AcceptURL(in.ObjectID, in.Kind)
}

// clarifyURL 包装 ClarifyURL。
func clarifyURL(in Input) string {
	return ClarifyURL(in.ObjectID, in.Kind)
}

// scheduleURL 包装 ScheduleURL。
func scheduleURL(in Input) string {
	return ScheduleURL(in.ObjectID, in.Kind)
}

// submitTestURL 包装 SubmitTestURL。
func submitTestURL(in Input) string {
	return SubmitTestURL(in.ObjectID, in.Kind)
}

// acceptDoneURL 包装 AcceptDoneURL。
func acceptDoneURL(in Input) string {
	return AcceptDoneURL(in.ObjectID, in.Kind)
}

// urgeAcceptURL 包装 UrgeAcceptURL。
func urgeAcceptURL(in Input) string {
	return UrgeAcceptURL(in.ObjectID, in.Kind)
}

// deliverURL 包装 DeliverURL。
func deliverURL(in Input) string {
	return DeliverURL(in.ObjectID, in.Kind)
}

// evaluateURL 包装 EvaluateURL。
func evaluateURL(in Input) string {
	return EvaluateURL(in.ObjectID, in.Kind)
}

// viewEvaluateURL 包装 ViewEvaluateURL。
func viewEvaluateURL(in Input) string {
	return ViewEvaluateURL(in.ObjectID, in.Kind)
}
