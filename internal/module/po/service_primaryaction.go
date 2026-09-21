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
//   - capabilities 由 hasCapability 读取认证中间件放入请求上下文的权限快照；
//     超级管理员放行，普通用户通过 perm.HasAnyGranted 检查实际能力。
//   - 对象级授权：IsAcceptanceOwner = (accepter == actor.Account)。
//   - 评价 / 测试单事实：批量 IN 查询一次性 fetch。
type primaryActionFactsDemand struct {
	ObjectID              uint
	Kind                  primaryaction.ObjectKind
	Stage                 primaryaction.StageKey
	Status                string
	CreatedBy             string
	IsCreator             bool
	IsAssignee            bool
	CanReview             bool
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

	// 批量查询待评审需求中当前用户是否有待评审任务。
	var waitIDs []int
	for _, id := range demandIDs {
		row := rows[id]
		if strings.TrimSpace(row.Status) == "wait" {
			waitIDs = append(waitIDs, int(id))
		}
	}
	var pendingReviewMap map[int]struct{}
	if len(waitIDs) > 0 && account != "" && s.repo != nil {
		pendingReviewMap, _ = s.repo.FindPendingReviewDemandIDs(ctx, account, waitIDs)
	}

	for _, id := range demandIDs {
		row := rows[id]
		facts := primaryActionFactsDemand{
			ObjectID:  id,
			Kind:      primaryaction.ObjectBusinessDemand,
			Stage:     deriveStageKey(row.Stage, row.Status),
			Status:    row.Status,
			CreatedBy: row.CreatedBy,
		}
		if account != "" {
			if strings.TrimSpace(row.Accepter) == account {
				facts.IsAcceptanceOwner = true
			}
			if strings.TrimSpace(row.CreatedBy) == account {
				facts.IsCreator = true
			}
			if strings.TrimSpace(row.AssignedTo) == account {
				facts.IsAssignee = true
			}
		}
		if pendingReviewMap != nil {
			if _, ok := pendingReviewMap[int(id)]; ok {
				facts.CanReview = true
			}
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
		out[id] = deriveDemandPrimaryAction(ctx, actor, facts)
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
			Stage:                   deriveStoryStageKey(row.Stage, row.Status),
			Kind:                    kind,
			ObjectID:                id,
			HasAcceptCapability:     hasCapability(ctx, actor, perm.PoHomeList, perm.PoBoardDemandList),
			HasClarifyCapability:    hasCapability(ctx, actor, perm.PoHomeList),
			HasScheduleCapability:   hasCapability(ctx, actor, perm.ScheduleList, perm.ScheduleUpdate),
			HasSubmitTestCapability: hasCapability(ctx, actor, perm.PoHomeList),
			HasUrgeCapability:       hasCapability(ctx, actor, perm.PoHomeList),
			HasDeliverCapability:    hasCapability(ctx, actor, perm.PoHomeList),
			HasEvaluateCapability:   hasCapability(ctx, actor, perm.PoHomeList),
			HasReadCapability:       hasCapability(ctx, actor, perm.PoHomeList, perm.PoBoardDemandList),
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
func deriveDemandPrimaryAction(ctx context.Context, actor *model.User, facts primaryActionFactsDemand) primaryaction.PrimaryAction {
	in := primaryaction.Input{
		Stage:                   facts.Stage,
		Status:                  facts.Status,
		Kind:                    facts.Kind,
		ObjectID:                facts.ObjectID,
		CreatedBy:               facts.CreatedBy,
		IsCreator:               facts.IsCreator,
		IsAssignee:              facts.IsAssignee,
		CanReview:               facts.CanReview,
		HasAcceptCapability:     hasCapability(ctx, actor, perm.PoHomeList, perm.PoBoardDemandList),
		HasClarifyCapability:    hasCapability(ctx, actor, perm.PoHomeList),
		HasScheduleCapability:   hasCapability(ctx, actor, perm.ScheduleList, perm.ScheduleUpdate),
		HasSubmitTestCapability: hasCapability(ctx, actor, perm.PoHomeList),
		HasUrgeCapability:       hasCapability(ctx, actor, perm.PoHomeList),
		HasDeliverCapability:    hasCapability(ctx, actor, perm.PoHomeList),
		HasEvaluateCapability:   hasCapability(ctx, actor, perm.PoHomeList),
		HasReadCapability:       hasCapability(ctx, actor, perm.PoHomeList, perm.PoBoardDemandList),
		IsAcceptanceOwner:       facts.IsAcceptanceOwner,
		TestsCount:              facts.TestsCount,
		FirstTestURL:            facts.FirstTestURL,
		HasPendingEvaluateTask:  facts.HasPendingEvaluate,
		HasHistoricalEvaluate:   facts.HasHistoricalEvaluate,
	}
	return primaryaction.Derive(in)
}

// hasCapability reads the request capability snapshot installed by RequireLogin.
// Authentication alone is never a capability grant.
func hasCapability(ctx context.Context, actor *model.User, perms ...perm.Permission) bool {
	if actor == nil {
		return false
	}
	if actor.IsSuperAdmin {
		return true
	}
	return perm.HasAnyGranted(ctx, perms...)
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

// deriveStoryStageKey 把研发需求（zt_story）的 stage 与 status 映射到 primaryaction.StageKey。
// 研发需求没有需求澄清阶段；active 状态下由 stage 决定其生命周期（如 wait -> StageSchedule）。
func deriveStoryStageKey(stage, status string) primaryaction.StageKey {
	st := strings.ToLower(strings.TrimSpace(status))
	if st == "closed" {
		return primaryaction.StageClosed
	}
	sg := strings.ToLower(strings.TrimSpace(stage))
	switch sg {
	case "wait", "planned", "projected", "schedule":
		return primaryaction.StageSchedule
	case "developing", "developed":
		return primaryaction.StageDeveloping
	case "tested", "testing", "delivering":
		return primaryaction.StageTesting
	case "verified":
		return primaryaction.StageAcceptance
	case "released":
		return primaryaction.StageDelivered
	case "closed":
		return primaryaction.StageClosed
	}
	switch st {
	case "developing":
		return primaryaction.StageDeveloping
	case "testing", "tested":
		return primaryaction.StageTesting
	case "active":
		return primaryaction.StageSchedule
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
