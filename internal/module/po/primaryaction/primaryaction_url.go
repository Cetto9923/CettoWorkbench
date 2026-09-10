// =============================================================================
// 文件: internal/module/po/primaryaction/primaryaction_url.go
// 模块: PO 工作台
// 类型: contract
// 职责: primaryAction 各阶段的 URL 拼接。
//       可办理操作必须指向禅道原生页面/API；测试单使用 TesttaskViewURL。
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

// AcceptURL 评审站内提交端点（前端按 drawer 形态提交）。
func AcceptURL(objectID uint, kind ObjectKind) string {
	return fmt.Sprintf("/demands/%d/review", objectID)
}

// WithdrawReviewURL 撤回评审站内提交端点。
func WithdrawReviewURL(objectID uint, kind ObjectKind) string {
	return fmt.Sprintf("/demands/%d/withdraw-review", objectID)
}

// SubmitReviewURL 提交评审站内端点。
func SubmitReviewURL(objectID uint, kind ObjectKind) string {
	return fmt.Sprintf("/demands/%d/submit-review", objectID)
}

// ClarifyURL 澄清站内提交端点。
func ClarifyURL(objectID uint, kind ObjectKind) string {
	return fmt.Sprintf("/demands/%d/clarify", objectID)
}

// ScheduleURL 排期入口；业务需求 vs 独立研发需求走不同路由。
func ScheduleURL(objectID uint, kind ObjectKind) string {
	if kind == ObjectStory || kind == ObjectIndependentStory {
		return fmt.Sprintf("/schedule/stories/%d/scheduling", objectID)
	}
	return fmt.Sprintf("/schedule/demands/%d/scheduling", objectID)
}

// SubmitTestURL 提测改走四步弹窗，不再返回旧整页 /submit-test URL。
// 保留函数签名供 Derive/单测调用；空串表示前端按 key=submit_test 开 modal。
func SubmitTestURL(objectID uint, kind ObjectKind) string {
	_ = objectID
	_ = kind
	return ""
}

// AcceptDoneURL 验收站内提交端点（本人验收人）。
func AcceptDoneURL(objectID uint, kind ObjectKind) string {
	return fmt.Sprintf("/demands/%d/acceptance", objectID)
}

// UrgeAcceptURL 催办验收站内提交端点（非验收人）。
func UrgeAcceptURL(objectID uint, kind ObjectKind) string {
	return fmt.Sprintf("/demands/%d/urge", objectID)
}

// DeliverURL 发起交付站内提交端点。
func DeliverURL(objectID uint, kind ObjectKind) string {
	return fmt.Sprintf("/demands/%d/deliver", objectID)
}

// EvaluateURL 评价端点。
// §4-2 阻塞：当前无 endpoint。
func EvaluateURL(objectID uint, kind ObjectKind) string {
	return zentao.URL("demand", "appraise", fmt.Sprintf("demandID=%d", objectID))
}

// acceptURL 包装 AcceptURL — 与 Derive Input 内部使用一致。
func acceptURL(in Input) string {
	return AcceptURL(in.ObjectID, in.Kind)
}

func withdrawReviewURL(in Input) string {
	return WithdrawReviewURL(in.ObjectID, in.Kind)
}

func submitReviewURL(in Input) string {
	return SubmitReviewURL(in.ObjectID, in.Kind)
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
