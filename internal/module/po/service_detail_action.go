// =============================================================================
// 文件: internal/module/po/service_detail_action.go
// 模块: PO 工作台
// 类型: service
// 职责: 业务需求详情页主操作（PrimaryAction）与状态标志（Flags）派生。
// 依赖: internal/model, internal/module/po/primaryaction, internal/pkg/perm
// =============================================================================

package po

import (
	"context"
	"strings"

	"workbench/internal/model"
	"workbench/internal/module/po/primaryaction"
	"workbench/internal/pkg/perm"
)

// FlagNoticePaused 是需求存在挂起/变更中/退回中标志时的统一步伐暂停文案。
const FlagNoticePaused = "需求已挂起/变更中/退回中，主操作已暂停，请在禅道处理"

// deriveDetailFlags 从需求行解析 hang/changing/returning 三标志及提示文案。
func deriveDetailFlags(row *DemandDetailRow) (DemandFlags, string) {
	if row == nil {
		return DemandFlags{}, ""
	}
	flags := DemandFlags{
		Hang:      strings.TrimSpace(row.Hang) == "1",
		Changing:  strings.TrimSpace(row.IsChange) == "changing",
		Returning: strings.TrimSpace(row.IsReturned) == "1",
	}
	var notice string
	if flags.Hang || flags.Changing || flags.Returning {
		notice = FlagNoticePaused
	}
	return flags, notice
}

// deriveAcceptancePrimaryAction 为待验收状态（waitacceptance）派生详情主操作。
// 口径：assignedTo == 当前用户 时派生验收；非 assignedTo 时派生催办验收。
func deriveAcceptancePrimaryAction(ctx context.Context, actor *model.User, row *DemandDetailRow) primaryaction.PrimaryAction {
	account := ""
	if actor != nil {
		account = strings.TrimSpace(actor.Account)
	}
	isAssignee := account != "" && account == strings.TrimSpace(row.AssignedTo)
	if isAssignee {
		return primaryaction.Enabled(
			string(primaryaction.KeyAcceptDone),
			"验收",
			string(primaryaction.KindDrawer),
			primaryaction.AcceptDoneURL(row.ID, primaryaction.ObjectBusinessDemand),
		)
	}
	if hasCapability(ctx, actor, perm.PoHomeList) {
		return primaryaction.Enabled(
			string(primaryaction.KeyRemindAccept),
			"催办验收",
			string(primaryaction.KindDrawer),
			primaryaction.UrgeAcceptURL(row.ID, primaryaction.ObjectBusinessDemand),
		)
	}
	return primaryaction.DisabledWithReason(
		string(primaryaction.KeyRemindAccept),
		"催办验收",
		string(primaryaction.KindDrawer),
		primaryaction.UrgeAcceptURL(row.ID, primaryaction.ObjectBusinessDemand),
		"当前用户没有催办验收权限",
	)
}

// buildPrimaryActionForDetail 详情行主操作。
// 标志存在时返回 None；waitacceptance 状态下按 assignedTo==本人 改判；其他状态复用批量派生。
func (s *DetailService) buildPrimaryActionForDetail(
	ctx context.Context,
	actor *model.User,
	row *DemandDetailRow,
) primaryaction.PrimaryAction {
	if row == nil || row.ID == 0 {
		return primaryaction.None()
	}
	if strings.TrimSpace(row.Hang) == "1" ||
		strings.TrimSpace(row.IsChange) == "changing" ||
		strings.TrimSpace(row.IsReturned) == "1" {
		return primaryaction.None()
	}
	if strings.TrimSpace(row.Status) == "waitacceptance" {
		return deriveAcceptancePrimaryAction(ctx, actor, row)
	}
	svc := s.parentService()
	if svc == nil {
		return primaryaction.None()
	}
	out, err := svc.DeriveDemandPrimaryActions(ctx, actor, []uint{row.ID})
	if err != nil {
		return primaryaction.None()
	}
	if pa, ok := out[row.ID]; ok {
		return pa
	}
	return primaryaction.None()
}
