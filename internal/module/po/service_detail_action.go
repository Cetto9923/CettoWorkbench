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
	"fmt"
	"strings"

	"workbench/internal/model"
	"workbench/internal/module/po/primaryaction"
	"workbench/internal/pkg/perm"
)

// deriveDetailFlags 从需求行解析 hang/changing/returning 三标志及提示文案。
// 口径：
//  1. hang == "1" 为已挂起；
//  2. isChange == "changing" 为变更中；
//  3. isReturned == "1" 为退回中；
//  4. 单标志文案如「需求已挂起，主操作已暂停，请在禅道处理」；
//     多个标志并列用「、」（如「需求已挂起、变更中，主操作已暂停，请在禅道处理」）。
func deriveDetailFlags(row *DemandDetailRow) (DemandFlags, string) {
	if row == nil {
		return DemandFlags{}, ""
	}
	flags := DemandFlags{
		Hang:      strings.TrimSpace(row.Hang) == "1",
		Changing:  strings.TrimSpace(row.IsChange) == "changing",
		Returning: strings.TrimSpace(row.IsReturned) == "1",
	}
	var parts []string
	if flags.Hang {
		parts = append(parts, "已挂起")
	}
	if flags.Changing {
		parts = append(parts, "变更中")
	}
	if flags.Returning {
		parts = append(parts, "退回中")
	}
	if len(parts) == 0 {
		return flags, ""
	}
	notice := fmt.Sprintf("需求%s，主操作已暂停，请在禅道处理", strings.Join(parts, "、"))
	return flags, notice
}

// deriveAcceptancePrimaryAction 为待验收状态（waitacceptance）派生详情主操作。
// 口径：
// 1. assignedTo == 当前用户 且具有验收权限（PoHomeList 或 PoBoardDemandList）时，派生启用的「验收」；
// 2. assignedTo == 当前用户 但无验收权限时，派生禁用的「验收」（文案「验收办理页尚未接入真实禅道写链」）；
// 3. 非 assignedTo 用户且具有催办权限（PoHomeList）时，派生启用的「催办验收」；
// 4. 非 assignedTo 用户且无催办权限时，派生禁用的「催办验收」（文案「当前用户没有催办验收权限」）。
func deriveAcceptancePrimaryAction(ctx context.Context, actor *model.User, row *DemandDetailRow) primaryaction.PrimaryAction {
	account := ""
	if actor != nil {
		account = strings.TrimSpace(actor.Account)
	}
	isAssignee := account != "" && account == strings.TrimSpace(row.AssignedTo)
	if isAssignee {
		if hasCapability(ctx, actor, perm.PoHomeList, perm.PoBoardDemandList) {
			return primaryaction.Enabled(
				string(primaryaction.KeyAcceptDone),
				"验收",
				string(primaryaction.KindDrawer),
				primaryaction.AcceptDoneURL(row.ID, primaryaction.ObjectBusinessDemand),
			)
		}
		return primaryaction.DisabledWithReason(
			string(primaryaction.KeyAcceptDone),
			"验收",
			string(primaryaction.KindDrawer),
			primaryaction.AcceptDoneURL(row.ID, primaryaction.ObjectBusinessDemand),
			"验收办理页尚未接入真实禅道写链",
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
// waitacceptance 状态下按 assignedTo==本人 改判；其他状态复用批量派生。
// 注意：标志判断由外层通过 Flags.HasAny() 统一处理，内部不再重复判断。
func (s *DetailService) buildPrimaryActionForDetail(
	ctx context.Context,
	actor *model.User,
	row *DemandDetailRow,
) primaryaction.PrimaryAction {
	if row == nil || row.ID == 0 {
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

// applyPrimaryActionForDetail 根据详情标志与状态派生并设置 PrimaryAction。
// 若存在挂起/变更中/退回中标志，主操作置空（nil）；否则根据状态与写权限派生。
func (s *DetailService) applyPrimaryActionForDetail(
	ctx context.Context,
	actor *model.User,
	row *DemandDetailRow,
	resp *DemandDetailResp,
	canWrite bool,
) {
	if resp == nil {
		return
	}
	if resp.Summary.Flags.HasAny() {
		resp.PrimaryAction = nil
		return
	}
	pa := s.buildPrimaryActionForDetail(ctx, actor, row)
	// 提测改走四步弹窗：详情 JSON 不再下发旧整页 URL，避免任何入口误跳 submit_test.html。
	if pa.Key == string(primaryaction.KeySubmitTest) {
		pa.URL = ""
	}
	// 写权限闸门：与排期保存接口同一口径（超管 / PMO / 干系人 / 团队长管辖）。
	// 仅持 perm.ScheduleList 的只读用户不出现可点的写按钮。
	if !canWrite {
		gatePrimaryActionForReadOnly(&pa)
	}
	resp.PrimaryAction = &pa
	bindPrimaryActionSpotlight(resp.Spotlight, pa)
	if pa.Key == string(primaryaction.KeyApprove) && pa.Enabled {
		resp.Summary.CanReview = true
	}
}
