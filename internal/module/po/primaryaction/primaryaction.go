// =============================================================================
// 文件: internal/module/po/primaryaction/primaryaction.go
// 模块: PO 工作台
// 类型: contract
// 职责: 服务层 primaryAction 派生（价值流阶段办理的单一主操作）。
//       根据 valueStream 阶段 + 对象种类 + 当前用户 capabilities + 对象级授权 +
//       前置条件（验收人/评价资格），派生该行的单一主操作。
//
//       设计原则：
//         - 每行只能有一个 primaryAction；不再兜底为"查看详情"。
//         - Service 派生；Repo 仅投影候选事实；Handler 仅做协议适配。
//         - 当现有 service/handler 未配置（§4-1 审批 / §4-2 评价 / §4-4 测试单关联）
//           时，key 仍按合同返回，enabled=false + reason 写明未配置，避免前端假按钮。
//
// 依赖: model.User（actor）
// =============================================================================

package primaryaction

import "strings"

// ActionKind 行为类型，与前端按钮形态对应。
type ActionKind string

const (
	KindInternal ActionKind = "internal" // 内部页面（相对 URL，当前页打开）
	KindExternal ActionKind = "external" // 外部链接（禅道 / 外部系统，新窗打开）
	KindDrawer   ActionKind = "drawer"   // 当前页弹层 / 抽屉打开
	KindSchedule ActionKind = "schedule" // 排期页跳转（同 KindInternal，但语义区分）
)

// ActionKey 是按价值流阶段派生的主操作标识。前端按 key 渲染对应按钮形态。
//
// 与 Stage 5 任务输入锁定的合同表一一对应（注意：受理与验收本人的 key 不同）。
type ActionKey string

const (
	KeyApprove        ActionKey = "approve"         // 评审（待我评审）
	KeyWithdrawReview ActionKey = "withdraw_review" // 撤回评审（待评审+本人创建）
	KeySubmitReview   ActionKey = "submit_review"   // 提交评审（草稿/驳回+本人创建）
	KeyEdit           ActionKey = "edit"            // 编辑需求
	KeyClarify        ActionKey = "clarify"         // 澄清
	KeySchedule       ActionKey = "schedule"        // 排期
	KeySubmitTest     ActionKey = "submit_test"     // 提测
	KeyViewTestOrder  ActionKey = "view_test_order" // 联调测试 → 测试单
	KeyAcceptDone     ActionKey = "accept"          // 验收（本人验收人）
	KeyRemindAccept   ActionKey = "remind_accept"   // 催办验收（非验收人）
	KeyDeliver        ActionKey = "deliver"         // 发起交付
	KeyPublish        ActionKey = "publish"         // 发布（暂留空）
	KeyEvaluate       ActionKey = "evaluate"        // 评价反馈（本人有未评任务）
	KeyViewEvaluate   ActionKey = "view_evaluate"   // 评价反馈（已可读历史评价）
)

// PrimaryAction 单一主操作的服务端合同字段；前端只读，不派生。
//
// 注意：字段命名与现有 JSON 合同保持小驼峰；前端在 Stage 6 直接读取。
type PrimaryAction struct {
	Key     string `json:"key"`     // 见 ActionKey 常量；空串 = 无可派生操作
	Label   string `json:"label"`   // 按钮文字；空操作时为 "—"
	Kind    string `json:"kind"`    // 见 ActionKind；空操作时为 ""
	URL     string `json:"url"`     // 跳转 URL；internal 相对、external 禅道完整 URL
	Enabled bool   `json:"enabled"` // false = 显示但禁用
	Reason  string `json:"reason"`  // enabled=false 时的中文原因
}

// None 是无任何派生主操作时的统一返回（前端显示 "—"）。
func None() PrimaryAction {
	return PrimaryAction{
		Key:     "",
		Label:   "—",
		Kind:    "",
		URL:     "",
		Enabled: false,
		Reason:  "当前阶段无可办理事项",
	}
}

// DisabledWithReason 派生主操作存在但当前行无法启用（例如非验收人催办但无 urge 权限）。
// 用于保留按钮可见但不可点，并显示明确原因。
func DisabledWithReason(key, label, kind, url, reason string) PrimaryAction {
	return PrimaryAction{
		Key:     string(key),
		Label:   label,
		Kind:    string(kind),
		URL:     url,
		Enabled: false,
		Reason:  reason,
	}
}

// Enabled 派生主操作且当前行可启用。
func Enabled(key, label, kind, url string) PrimaryAction {
	return PrimaryAction{
		Key:     string(key),
		Label:   label,
		Kind:    string(kind),
		URL:     url,
		Enabled: true,
		Reason:  "",
	}
}

// StageKey 是来自 service_detail_tabs.mapValueStage 的内部 key。
// 与 plan §4 / PLAN §4 表一一对应。
type StageKey string

const (
	StageAccept     StageKey = "accept"     // 受理
	StageClarify    StageKey = "clarify"    // 澄清
	StageSchedule   StageKey = "schedule"   // 排期
	StageDeveloping StageKey = "developing" // 研发（含 submittest）
	StageTesting    StageKey = "testing"    // 测试
	StageAcceptance StageKey = "acceptance" // 验收（waitacceptance）
	StageDeliver    StageKey = "deliver"    // 发起交付（acceptanced）
	StageRelease    StageKey = "release"    // 待发布（waitdeliver）— 留空
	StageFeedback   StageKey = "feedback"   // 评价反馈（released）
	StageClosed     StageKey = "closed"     // 已关闭
	StageDelivered  StageKey = "delivered"  // 已上线完成
	StageOther      StageKey = "other"      // 兜底
)

// ObjectKind 表示当前行对象类型。
type ObjectKind string

const (
	ObjectBusinessDemand   ObjectKind = "business_demand"
	ObjectSubDemand        ObjectKind = "sub_demand"
	ObjectStory            ObjectKind = "story"
	ObjectIndependentStory ObjectKind = "independent_story"
)

// Input 是派生主操作的输入。
//
// 与 AGENTS.md 一致：业务规则 / 对象级授权在 Service 层汇总为结构化事实后传入；
// Repo 不做授权决策，仅提供事实投影（HasTests / AcceptanceOwner 等）。
type Input struct {
	Stage    StageKey
	Status   string
	Kind     ObjectKind
	ObjectID uint

	// 需求创建人与当前账号关系
	CreatedBy  string
	IsCreator  bool
	IsAssignee bool
	CanReview  bool

	// 当前用户对当前行的能力事实（由 Service 检查后传入）：
	HasAcceptCapability     bool
	HasClarifyCapability    bool
	HasScheduleCapability   bool
	HasSubmitTestCapability bool
	HasUrgeCapability       bool
	HasDeliverCapability    bool
	HasEvaluateCapability   bool
	HasReadCapability       bool

	// 当前用户是否就是当前对象的验收人（Service 通过 CheckDemandVisibility
	// 或 loadDemandIfVisible 计算后传入）。
	IsAcceptanceOwner bool

	// 联调测试：当前 story / demand 关联的测试单候选。
	// - TestsCount == 0：无测试单
	// - TestsCount == 1：直接展示该单 URL
	// - TestsCount >  1：保留为空 URL + reason 提示选择列表（前端负责选单）
	TestsCount   int
	FirstTestURL string

	// 评价反馈：当前用户是否对该需求有未完成的评价任务。
	HasPendingEvaluateTask bool
	// 评价反馈：当前需求是否已有可读历史评价。
	HasHistoricalEvaluate bool
}

// Derive 由 Input 派生单一主操作。
//
// 决策矩阵（与 PLAN §4 + Stage 5 任务输入锁定的"阶段 → primaryAction 合同表"一致）：
//
//	Stage / Kind       Action            默认 URL                  Enabled 默认值
//	───────────────────────────────────────────────────────────────────────────
//	accept             modal accept      POST /demands/:id/accept  capability
//	clarify            modal clarify     /demands/:id/detail?tab=req capability
//	schedule + biz     internal schedule /schedule/demands/:id/sched  capability
//	schedule + story   internal schedule /schedule/stories/:id/sched capability
//	developing         modal submit_test （无整页 URL；前端 openPoSubmitTestModal） capability
//	testing            external test_link 禅道测试单 URL / "暂无测试单"  capability (always false: 未配置)
//	acceptance+本人    modal accept_done POST /demands/:id/accept-done IsAcceptanceOwner
//	acceptance+他人    modal urge_accept POST /demands/:id/urge-accept capability
//	deliver            modal deliver    POST /demands/:id/deliver  capability+前置
//	release            "" 留空          —                         永远 None
//	feedback+未评      modal evaluate   POST /demands/:id/evaluate HasPendingEvaluateTask
//	feedback+有评价    internal view_evaluate /demands/:id/detail?tab=history capability
//	closed/delivered/  ""               —                         None
//	other / 兜底
//
// 业务子需求、子需求、研发需求的判断由 Input.Kind 决定（URL target）。
func Derive(in Input) PrimaryAction {
	if in.ObjectID == 0 {
		return None()
	}

	switch in.Stage {
	case StageAccept:
		if in.Kind == ObjectStory || in.Kind == ObjectIndependentStory {
			return None()
		}
		st := strings.ToLower(strings.TrimSpace(in.Status))
		switch st {
		case "wait":
			if in.CanReview {
				if !in.HasAcceptCapability {
					return DisabledWithReason(string(KeyApprove), "评审", string(KindDrawer),
						acceptURL(in),
						"当前用户没有评审权限")
				}
				return Enabled(string(KeyApprove), "评审", string(KindDrawer), acceptURL(in))
			}
			if in.IsCreator {
				return Enabled(string(KeyWithdrawReview), "撤回", string(KindDrawer), withdrawReviewURL(in))
			}
			return None()
		case "draft", "refuse":
			if in.IsCreator || in.IsAssignee {
				return Enabled(string(KeySubmitReview), "提交评审", string(KindDrawer), submitReviewURL(in))
			}
			return None()
		default:
			// 兜底：未显式传 status 但标记了 CanReview（如单测或旧兼容）
			if in.CanReview {
				if !in.HasAcceptCapability {
					return DisabledWithReason(string(KeyApprove), "评审", string(KindDrawer),
						acceptURL(in),
						"当前用户没有评审权限")
				}
				return Enabled(string(KeyApprove), "评审", string(KindDrawer), acceptURL(in))
			}
			if in.IsCreator {
				return Enabled(string(KeyWithdrawReview), "撤回", string(KindDrawer), withdrawReviewURL(in))
			}
			return None()
		}

	case StageClarify:
		if in.Kind == ObjectStory || in.Kind == ObjectIndependentStory {
			return None()
		}
		if !in.HasClarifyCapability {
			return DisabledWithReason(string(KeyClarify), "澄清", string(KindDrawer),
				clarifyURL(in),
				"当前用户没有澄清权限")
		}
		return Enabled(string(KeyClarify), "澄清", string(KindDrawer), clarifyURL(in))

	case StageSchedule:
		if !in.HasScheduleCapability {
			return DisabledWithReason(string(KeySchedule), "排期", string(KindInternal),
				scheduleURL(in),
				"当前用户没有排期权限")
		}
		return Enabled(string(KeySchedule), "排期", string(KindSchedule), scheduleURL(in))

	case StageDeveloping:
		// 提测：四步弹窗（testtask 模块）；不再挂旧整页 /submit-test。
		if !in.HasSubmitTestCapability {
			return DisabledWithReason(string(KeySubmitTest), "提测", string(KindDrawer),
				submitTestURL(in),
				"当前用户没有提测权限")
		}
		return Enabled(string(KeySubmitTest), "提测", string(KindDrawer), submitTestURL(in))

	case StageTesting:
		// 联调测试 → 禅道测试单。
		// §4-4 Stage 5 必须为 test_link 加 zt_testtask join；当前 Repo
		// 已存在 FindStoryTestTasks，本函数消费其结果。
		if !in.HasReadCapability {
			return DisabledWithReason(string(KeyViewTestOrder), "测试单", string(KindExternal),
				"", "当前用户没有读取权限")
		}
		switch {
		case in.TestsCount == 0:
			return DisabledWithReason(string(KeyViewTestOrder), "测试单", string(KindExternal),
				"", "暂无关联测试单")
		case in.TestsCount == 1:
			return Enabled(string(KeyViewTestOrder), "测试单", string(KindExternal), in.FirstTestURL)
		default:
			return DisabledWithReason(string(KeyViewTestOrder), "测试单", string(KindExternal),
				"", fmtNTestTasks(in.TestsCount))
		}

	case StageAcceptance:
		if in.IsAcceptanceOwner {
			if in.HasAcceptCapability {
				return Enabled(string(KeyAcceptDone), "验收", string(KindDrawer), acceptDoneURL(in))
			}
			return DisabledWithReason(string(KeyAcceptDone), "验收", string(KindDrawer),
				acceptDoneURL(in), "验收办理页尚未接入真实禅道写链")
		}
		if !in.HasUrgeCapability {
			return DisabledWithReason(string(KeyRemindAccept), "催办验收", string(KindDrawer), urgeAcceptURL(in), "当前用户没有催办验收权限")
		}
		return Enabled(string(KeyRemindAccept), "催办验收", string(KindDrawer), urgeAcceptURL(in))

	case StageDeliver:
		if !in.HasDeliverCapability {
			return DisabledWithReason(string(KeyDeliver), "发起交付", string(KindDrawer),
				deliverURL(in),
				"当前用户没有发起交付权限")
		}
		// 前置条件：交付需先验收（row.Accepter != "" 或 row.Status 在已验收）。
		// Service 层校验后把 okDeliver 表达为 HasAcceptanceCompleted。
		// 为避免在 Input 加冗余字段，这里用 stage 自身判定：
		// StageDeliver 已表示 status==acceptanced，前置默认通过。
		return Enabled(string(KeyDeliver), "发起交付", string(KindDrawer), deliverURL(in))

	case StageRelease:
		// 发布：plan §4 显式约定留空，不创建假按钮。
		return None()

	case StageFeedback:
		// 评价反馈：未评 + 已评价两种入口并存，PLAN §4-3 / Stage 1 阻塞。
		switch {
		case in.HasPendingEvaluateTask:
			if in.HasEvaluateCapability {
				return Enabled(string(KeyEvaluate), "评价", string(KindExternal), evaluateURL(in))
			}
			return DisabledWithReason(string(KeyEvaluate), "评价", string(KindDrawer),
				evaluateURL(in), "评价办理页尚未接入真实禅道写链")
		case in.HasHistoricalEvaluate:
			// PRD explicitly excludes a historical-evaluation action from PO flow.
			return None()
		default:
			return None()
		}

	case StageClosed, StageDelivered:
		return None()

	default:
		return None()
	}
}

// formatNTestTasks 生成测试单多于 1 张时的中文 reason。
func fmtNTestTasks(n int) string {
	if n <= 1 {
		return "暂无关联测试单"
	}
	if n == 2 {
		return "已关联 2 张测试单，请在抽屉内选择"
	}
	return "已关联多张测试单，请在抽屉内选择"
}
