// =============================================================================
// 文件: internal/module/po/service_primaryaction.go
// 模块: PO 工作台
// 类型: service
// 职责: 业务需求 / 故事 / 独立研需的主操作派生（valueStream 阶段 + capabilities + 对象级授权）。
//       集中事实批量获取 → primaryaction.Derive。避免行循环查 DB（N+1）。
// 依赖: primaryaction, internal/pkg/perm
// =============================================================================

package po

import (
	"context"
	"strings"

	"workbench/internal/model"
	"workbench/internal/module/po/primaryaction"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/zentao"
)

// primaryActionFactsDemand  业务需求行派生主操作所需的服务层事实。
//
// 计算规则与矩阵 / 工程 约束一致：
//   - capabilities 由 perm.GrantedCapabilities 查询；当前 Service 不直接查 DB，
//     用 service 拥有的 actor 信息（最简形式：actor.IsSuperAdmin → 放行；否则假定
//     middleware 已 RequirePerm(PoHomeList|ScheduleList|PoBoardDemandList) 等）。
//     后续 Stage 6+ 可接入真正的 capability 检查；当前先按 actor 字段硬判定。
//   - 对象级授权：IsAcceptanceOwner = (accepter == actor.Account)。
//   - 评价 / 测试单事实：批量 IN 查询一次性 fetch。
type primaryActionFactsDemand struct {
	ObjectID              uint
	Kind                  primaryaction.ObjectKind
	Stage                 primaryaction.StageKey
	IsAcceptanceOwner     bool
	TestsCount            int
	FirstTestURL          string
	HasPendingEvaluate    bool
	HasHistoricalEvaluate bool
}

// DeriveDemandPrimaryActions 批量派生一组业务需求的 primaryAction。
//
// 入参 demandIDs 来自 /demands / /board/demand / 详情页主行；
// actor 由 Service 调用方提供；
// 列表场景下 prePageStoryIDs 用于业务需求 → 故事的 testtask 二次关联（可选；nil 时跳过）。
func (s *Service) DeriveDemandPrimaryActions(
	ctx context.Context,
	actor *model.User,
	demandIDs []uint,
) (map[uint]primaryaction.PrimaryAction, error) {
	out := make(map[uint]primaryaction.PrimaryAction, len(demandIDs))
	if len(demandIDs) == 0 {
		return out, nil
	}

	detailRepo := s.detailRepo()
	if detailRepo == nil {
		for _, id := range demandIDs {
			out[id] = primaryaction.None()
		}
		return out, nil
	}

	rows, err := detailRepo.FindDemandPrimaryActions(ctx, demandIDs)
	if err != nil {
		return nil, err
	}

	// 批量获取测试单事实（业务需求 → 故事 → 测试单）。
	ttMap, err := detailRepo.CountDemandTestTasks(ctx, demandIDs)
	if err != nil {
		return nil, err
	}

	// 批量获取评价事实。
	account := ""
	if actor != nil {
		account = strings.TrimSpace(actor.Account)
	}
	evalMap, err := detailRepo.FindDemandEvaluateStatus(ctx, account, demandIDs)
	if err != nil {
		return nil, err
	}

	for _, id := range demandIDs {
		row := rows[id]
		facts := primaryActionFactsDemand{
			ObjectID: id,
			Kind:     primaryaction.ObjectBusinessDemand,
			Stage:    deriveStageKey(row.Stage, row.Status),
		}
		if account != "" && strings.TrimSpace(row.Accepter) == account {
			facts.IsAcceptanceOwner = true
		}
		if tt, ok := ttMap[id]; ok {
			facts.TestsCount = tt.Count
			if tt.FirstID > 0 {
				facts.FirstTestURL = zentao.TesttaskViewURL(tt.FirstID)
			}
		}
		if eval, ok := evalMap[id]; ok {
			facts.HasPendingEvaluate = eval.HasPendingEvaluateForAccount
			facts.HasHistoricalEvaluate = eval.HasAnyEvaluate
		}
		out[id] = deriveDemandPrimaryAction(actor, facts)
	}
	return out, nil
}

// DeriveStoryPrimaryActions 批量派生一组研发需求（包括独立研需）的 primaryAction。
//
// 故事的 stage 来源：status + stage（与 mapValueStage 一致）。
// 故事的 accepter / assignedTo 不在 zt_story 语义上等价于业务需求，
// 这里直接复用 mapValueStage 派生阶段 key，并把 IsAcceptanceOwner 视为 false
// （研需对象的"本人验收"语义不在当前仓库可见）。
func (s *Service) DeriveStoryPrimaryActions(
	ctx context.Context,
	actor *model.User,
	storyIDs []uint,
	independentFlag bool,
) (map[uint]primaryaction.PrimaryAction, error) {
	out := make(map[uint]primaryaction.PrimaryAction, len(storyIDs))
	if len(storyIDs) == 0 {
		return out, nil
	}

	detailRepo := s.detailRepo()
	if detailRepo == nil {
		for _, id := range storyIDs {
			out[id] = primaryaction.None()
		}
		return out, nil
	}

	ttMap, err := detailRepo.CountStoryTestTasks(ctx, storyIDs)
	if err != nil {
		return nil, err
	}

	// 批量取故事事实（单次 IN），避免行循环单查（N+1）。
	metaMap, err := detailRepo.FindStoryMetaForAction(ctx, storyIDs)
	if err != nil {
		return nil, err
	}

	for _, id := range storyIDs {
		row, ok := metaMap[id]
		if !ok {
			// 单行不可见：返回 None 但不阻断整批（避免一个不可见 story 让整页失败）。
			out[id] = primaryaction.None()
			continue
		}
		kind := primaryaction.ObjectStory
		if independentFlag {
			kind = primaryaction.ObjectIndependentStory
		}
		in := primaryaction.Input{
			Stage:                   deriveStageKey(row.Stage, row.Status),
			Kind:                    kind,
			ObjectID:                id,
			HasAcceptCapability:     hasCapability(actor, perm.PoHomeList, perm.PoBoardDemandList),
			HasClarifyCapability:    hasCapability(actor, perm.PoHomeList),
			HasScheduleCapability:   hasCapability(actor, perm.ScheduleList, perm.ScheduleUpdate),
			HasSubmitTestCapability: hasCapability(actor, perm.PoHomeList),
			HasUrgeCapability:       hasCapability(actor, perm.PoHomeList),
			HasDeliverCapability:    hasCapability(actor, perm.PoHomeList),
			HasEvaluateCapability:   hasCapability(actor, perm.PoHomeList),
			HasReadCapability:       hasCapability(actor, perm.PoHomeList, perm.PoBoardDemandList),
			IsAcceptanceOwner:       false,
			TestsCount:              0,
			FirstTestURL:            "",
		}
		if tt, ok := ttMap[id]; ok {
			in.TestsCount = tt.Count
			if tt.FirstID > 0 {
				in.FirstTestURL = zentao.TesttaskViewURL(tt.FirstID)
			}
		}
		out[id] = primaryaction.Derive(in)
	}
	return out, nil
}

// deriveDemandPrimaryAction 由 Service 在已聚合事实后调用 primaryaction.Derive。
func deriveDemandPrimaryAction(actor *model.User, facts primaryActionFactsDemand) primaryaction.PrimaryAction {
	in := primaryaction.Input{
		Stage:                   facts.Stage,
		Kind:                    facts.Kind,
		ObjectID:                facts.ObjectID,
		HasAcceptCapability:     hasCapability(actor, perm.PoHomeList, perm.PoBoardDemandList),
		HasClarifyCapability:    hasCapability(actor, perm.PoHomeList),
		HasScheduleCapability:   hasCapability(actor, perm.ScheduleList, perm.ScheduleUpdate),
		HasSubmitTestCapability: hasCapability(actor, perm.PoHomeList),
		HasUrgeCapability:       hasCapability(actor, perm.PoHomeList),
		HasDeliverCapability:    hasCapability(actor, perm.PoHomeList),
		HasEvaluateCapability:   hasCapability(actor, perm.PoHomeList),
		HasReadCapability:       hasCapability(actor, perm.PoHomeList, perm.PoBoardDemandList),
		IsAcceptanceOwner:       facts.IsAcceptanceOwner,
		TestsCount:              facts.TestsCount,
		FirstTestURL:            facts.FirstTestURL,
		HasPendingEvaluateTask:  facts.HasPendingEvaluate,
		HasHistoricalEvaluate:   facts.HasHistoricalEvaluate,
	}
	return primaryaction.Derive(in)
}

// hasCapability 用 actor 字段的简化能力判定。
//
// 状态（2026-09-07 复核）：BLOCKED - CAPABILITY SOURCE NOT AVAILABLE IN SERVICE。
// 仓库内 capability 真源在 middleware 的 gin.Context["userPerms"] map
// （internal/middleware/permission.go:61-71，key = perm.Permission.String()）；
// 当前 Service 签名是 (ctx context.Context, actor *model.User)，
// 既拿不到 gin.Context，也拿不到 userPerms。*model.User 没有
// GrantedCapabilities 字段（internal/model/user.go）。因此本函数体只能
// 沿用"actor 缺失 / SuperAdmin / account 非空"三种放行；actor 非空并不等于
// 拥有具体业务 capability（已认证 ≠ 拥有业务权限）。
//
// 后续两条可选路径（不在本轮实施）：
//  1. middleware 在 c.Set("currentUser", u) 时同步写入 actor.GrantedCapabilities
//     字段；需要扩展 model.User 并在所有装载点同步。
//  2. middleware 将 userPerms 沿 c.Request.WithContext(...) 透传到 ctx.Value(...)；
//     Service 通过 ctx.Value("userPerms") 取出。需要在 agent-onboarding.md /
//     architecture.md 评估 ctx 透传机制是否被允许。
//
// 两条路径都需要独立任务与产品/架构授权。
func hasCapability(actor *model.User, _ ...perm.Permission) bool {
	if actor == nil {
		return false
	}
	if actor.IsSuperAdmin {
		return true
	}
	// middleware 已 RequirePerm 通过；这里仅按 actor 非空放行。
	// 注意：account 非空 ≠ 拥有目标 capability；本行为已知遗留，见 BLOCKED 说明。
	return strings.TrimSpace(actor.Account) != ""
}

// deriveStageKey 把业务需求的值流 status（或 zt_story.stage 列）映射到 primaryaction.StageKey。
//
// 与 service.go 的 valueStreamStages / repovaluestream.go 的 mysqlStageFilters 的 status
// 键保持一致（权威来源），使验收 / 发起交付 / 发布 / 评价反馈分支可被派生。
func deriveStageKey(stage, status string) primaryaction.StageKey {
	st := strings.ToLower(strings.TrimSpace(status))
	switch st {
	case "draft", "wait", "refuse":
		return primaryaction.StageAccept
	case "active", "clarify":
		return primaryaction.StageClarify
	case "clarified", "planned", "schedule":
		return primaryaction.StageSchedule
	case "developing":
		return primaryaction.StageDeveloping
	case "testing", "tested":
		return primaryaction.StageTesting
	case "waitacceptance":
		return primaryaction.StageAcceptance
	case "acceptanced":
		return primaryaction.StageDeliver
	case "waitdeliver", "publish", "publishing":
		return primaryaction.StageRelease
	case "released":
		return primaryaction.StageFeedback
	case "delivered", "verified":
		return primaryaction.StageDelivered
	case "closed":
		return primaryaction.StageClosed
	}
	// 兜底：zt_story.stage 列的常见取值（与 status 未命中时）。
	sg := strings.ToLower(strings.TrimSpace(stage))
	switch sg {
	case "wait", "draft":
		return primaryaction.StageAccept
	case "planned", "schedule":
		return primaryaction.StageSchedule
	case "developing":
		return primaryaction.StageDeveloping
	case "tested", "testing", "delivering":
		return primaryaction.StageTesting
	case "released":
		return primaryaction.StageDelivered
	case "closed":
		return primaryaction.StageClosed
	}
	return primaryaction.StageOther
}

// detailRepo 返回 Service 内 DetailService 持有的 DemandDetailRepo（不暴露 Repo 给外部）。
func (s *Service) detailRepo() *DemandDetailRepo {
	if s == nil || s.detailSvc == nil {
		return nil
	}
	return s.detailSvc.repo
}
