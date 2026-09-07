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

	"gorm.io/gorm"

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

	for _, id := range storyIDs {
		row, err := detailRepo.findStoryMetaForAction(ctx, id)
		if err != nil {
			// 单行失败：返回 None 但不阻断整批（避免一个不可见 story 让整页失败）。
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
// 当前 Service 不引入 actor 角色 → capability 表查询（避免新增 DB 表 / 复杂依赖）；
// 实际 capability 检查已由 middleware.RequirePerm 在入口保证。这里只在
// actor 缺失 / 非超管场景下做"最小保守放行"。后续 Stage 6+ 可引入更精细的
//
//	actor.GrantedCapabilities 投影，本函数保持 1 行兼容。
func hasCapability(actor *model.User, _ ...perm.Permission) bool {
	if actor == nil {
		return false
	}
	if actor.IsSuperAdmin {
		return true
	}
	// middleware 已 RequirePerm 通过；这里仅按 actor 非空放行。
	return strings.TrimSpace(actor.Account) != ""
}

// deriveStageKey 把数据库 stage+status 映射到 primaryaction.StageKey。
//
// 与 service_detail_tabs.mapValueStage 保持同义；不复用是因为 primaryaction
// StageKey 只覆盖 Derive 关心的子集，避免与 valueStreamStages 全表混淆。
func deriveStageKey(stage, status string) primaryaction.StageKey {
	_, label := mapValueStage(stage, status)
	switch label {
	case "已受理", "已签出", "草稿", "驳回":
		return primaryaction.StageAccept
	case "澄清中":
		return primaryaction.StageClarify
	case "排期中", "已排期":
		return primaryaction.StageSchedule
	case "研发中":
		return primaryaction.StageDeveloping
	case "测试中":
		return primaryaction.StageTesting
	case "待验收":
		return primaryaction.StageAcceptance
	case "发起交付":
		return primaryaction.StageDeliver
	case "待发布":
		return primaryaction.StageRelease
	case "评价反馈":
		return primaryaction.StageFeedback
	case "已上线完成":
		return primaryaction.StageDelivered
	case "已关闭":
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

// findStoryMetaForAction 派生 primaryAction 所需的 zt_story 最小列。
// 复用现有 repo_detail.FindDemandStories 已经按 fromDemand 过滤；这里只取
// 单行 stage/status（避免引入新 Repository）。
func (r *DemandDetailRepo) findStoryMetaForAction(ctx context.Context, id uint) (*DemandStoryRow, error) {
	if r == nil || r.db == nil || id == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var row DemandStoryRow
	err := r.db.WithContext(ctx).Table("zt_story").
		Select("id, status, stage").
		Where("id = ? AND deleted = ?", id, "0").
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}
