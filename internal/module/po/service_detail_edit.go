// =============================================================================
// 文件: internal/module/po/service_detail_edit.go
// 模块: PO 工作台
// 类型: service
// 职责: 业务需求编辑修改权限与锁定规则判定（对齐禅道原生需求生命周期）。
// =============================================================================

package po

import (
	"context"
	"strings"

	"workbench/internal/model"
)

// deriveDemandEditability 判定当前用户对需求的编辑权限及锁定原因。
//
// 禅道原生规则：
// 1. 状态为 draft (草稿) 或 refuse (已驳回)：创建人、指派人或管理员随时允许编辑；
// 2. 状态为 wait (待评审)：
//   - 若尚未有任何评审人出具评审结论 (reviewedCount == 0)：创建人、指派人或管理员允许直接编辑修改；
//   - 若已有评审人给出评审意见 (reviewedCount > 0)：需求锁定禁止直接修改，提示需先撤回评审；
//
// 3. 其它已进入流转的状态：不支持在此直接修改。
func deriveDemandEditability(actor *model.User, row *DemandDetailRow, reviewedCount int) (canEdit bool, reason string) {
	if actor == nil || row == nil {
		return false, ""
	}
	account := strings.TrimSpace(actor.Account)
	if account == "" {
		return false, ""
	}
	st := strings.ToLower(strings.TrimSpace(row.Status))
	isCreatorOrAdmin := actor.IsSuperAdmin || strings.TrimSpace(row.CreatedBy) == account
	isAssignee := strings.TrimSpace(row.AssignedTo) == account

	if st == "draft" || st == "refuse" {
		if isCreatorOrAdmin || isAssignee {
			return true, ""
		}
		return false, "非创建人或指派人无编辑权限"
	}
	if st == "wait" {
		if !isCreatorOrAdmin && !isAssignee {
			return false, "非创建人或当前指派人无编辑权限"
		}
		if reviewedCount == 0 {
			return true, ""
		}
		return false, "已有评审人出具评审意见，需求已锁定修改；如需修改请先撤回评审申请"
	}
	return false, "当前阶段状态不支持直接编辑修改"
}

// populateDemandEditability 在加载详情时计算并填入编辑权限相关字段。
func (s *DetailService) populateDemandEditability(ctx context.Context, actor *model.User, row *DemandDetailRow, summary *DemandSummary) {
	if s == nil || actor == nil || row == nil || summary == nil {
		return
	}
	var reviewedCount int64
	if s.repo != nil {
		c, err := s.repo.CountReviewedReviewers(ctx, row.ID)
		if err == nil {
			reviewedCount = c
		}
	}
	summary.ReviewedCount = int(reviewedCount)
	summary.HasReviewed = reviewedCount > 0

	canEdit, reason := deriveDemandEditability(actor, row, summary.ReviewedCount)
	summary.CanEdit = canEdit
	summary.EditDisabledReason = reason
}
