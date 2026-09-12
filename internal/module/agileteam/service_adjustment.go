// =============================================================================
// 文件: internal/module/agileteam/service_adjustment.go
// 模块: 敏捷小组治理
// 类型: action
// 职责: 提交 / 确认 / 驳回成员调整。
// 依赖: internal/model
//       internal/pkg/errorx
// =============================================================================

package agileteam

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

func actorAccount(actor *model.User) string {
	if actor == nil {
		return ""
	}
	return strings.TrimSpace(actor.Account)
}

func transitionAdjustmentError(err error) error {
	if errors.Is(err, errAdjustmentStateChanged) {
		return errorx.New("conflict", "调整单已被其他请求处理，请刷新后重试")
	}
	return err
}

// SubmitAdjustment PO/SM 提交成员调整；正式 zt_team 不变。
// allowGlobal 仅用于 PMO（AgileTeamConfirm）全局治理能力；普通工作台角色仍需
// 命中目标 teamgroup 的 PO/manager 对象关系，防止通过 ID 跨小组提交调整。
func (s *Service) SubmitAdjustment(ctx context.Context, actor *model.User, req SubmitAdjustmentReq, allowGlobal bool) (*AdjustmentDetailResp, error) {
	acc, err := requireActorAccount(actor)
	if err != nil {
		return nil, err
	}
	if err := validateAdjustmentWriteBoundary(req); err != nil {
		return nil, err
	}
	teamgroup, err := s.repo.FindTeamgroupByID(ctx, req.TeamgroupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.New("not_found", "敏捷小组不存在")
		}
		return nil, err
	}
	if err := requireTeamgroupObjectEdit(actor, teamgroup, allowGlobal); err != nil {
		return nil, err
	}

	accounts := make([]string, 0, len(req.Items))
	for _, it := range req.Items {
		accounts = append(accounts, it.Account)
	}
	existingUsers, err := s.repo.ExistingUserAccounts(ctx, accounts)
	if err != nil {
		return nil, err
	}
	for _, account := range accounts {
		if !existingUsers[strings.TrimSpace(account)] {
			return nil, errorx.New("invalid", "成员账号不存在或已删除："+strings.TrimSpace(account))
		}
	}

	now := time.Now()
	items := make([]AdjustmentItem, 0, len(req.Items))
	for _, it := range req.Items {
		prevRole, prevHours := "", 0.0
		if m, err := s.repo.FindMember(ctx, req.TeamgroupID, it.Account); err == nil && m != nil {
			prevRole, prevHours = m.Role, m.Hours
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if it.ActionType == ActionAdd && prevRole != "" {
			return nil, errorx.New("invalid", it.Account+" 已是正式成员，请改用角色调整或移除")
		}
		if (it.ActionType == ActionRemove || it.ActionType == ActionRoleChange) && prevRole == "" && prevHours == 0 {
			if _, e2 := s.repo.FindMember(ctx, req.TeamgroupID, it.Account); errors.Is(e2, gorm.ErrRecordNotFound) {
				return nil, errorx.New("invalid", it.Account+" 不是正式成员，无法"+it.ActionType)
			} else if e2 != nil {
				return nil, e2
			}
		}
		role := it.Role
		if role == "" {
			role = prevRole
			if role == "" {
				role = "研发"
			}
		}
		hours := it.AvailableHours
		if it.ActionType == ActionRemove {
			// 移除动作不改变工时；保留历史值用于差异展示。
			hours = prevHours
		}
		items = append(items, AdjustmentItem{
			Account: it.Account, ActionType: it.ActionType,
			Role: role, PrevRole: prevRole,
			AvailableHours: hours, PrevHours: prevHours,
			CreatedBy: acc, UpdatedBy: acc,
			CreatedDate: now, UpdatedDate: now,
		})
	}

	adj := &Adjustment{
		TeamgroupID: req.TeamgroupID, Status: StatusPending,
		Reason: req.Reason, SubmittedBy: acc,
		CreatedBy: acc, UpdatedBy: acc,
		CreatedDate: now, UpdatedDate: now,
	}
	history := &History{
		TeamgroupID: req.TeamgroupID, EventType: EventSubmit, Actor: acc,
		CreatedDate: now,
	}
	id, err := s.repo.SubmitPendingAdjustmentAtomically(ctx, adj, items, history)
	if err != nil {
		if errors.Is(err, errTeamgroupMissing) {
			return nil, errorx.New("not_found", "敏捷小组不存在")
		}
		if errors.Is(err, errAdjustNoConflict) || isAdjustNoDuplicate(err) {
			return nil, errorx.New("conflict", "调整单号冲突，请重试")
		}
		return nil, err
	}
	adj.ID = id
	return s.GetAdjustment(ctx, actor, id, false)
}

func countAction(items []AdjustmentItem, act string) int {
	n := 0
	for _, it := range items {
		if it.ActionType == act {
			n++
		}
	}
	return n
}

// ConfirmAdjustment 组织级教练确认 → 原子写入 zt_team + 调整状态 + 历史。
func (s *Service) ConfirmAdjustment(ctx context.Context, actor *model.User, req ConfirmReq, allowConfirm bool) error {
	acc, err := requireActorAccount(actor)
	if err != nil {
		return err
	}
	if req.AdjustmentID <= 0 {
		return errorx.New("invalid", "调整单 ID 无效")
	}
	if !actor.IsSuperAdmin && !allowConfirm {
		return errorx.New("forbidden", "仅 PMO 或管理员可确认成员调整")
	}
	adj, err := s.repo.FindAdjustmentByID(ctx, req.AdjustmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorx.New("not_found", "调整单不存在")
		}
		return err
	}
	if adj.Status != StatusPending {
		return errorx.New("invalid", "仅待确认调整单可确认")
	}
	items, err := s.repo.ListItems(ctx, adj.ID)
	if err != nil {
		return err
	}

	accounts := make([]string, 0, len(items))
	for _, item := range items {
		if item.ActionType == ActionAdd || item.ActionType == ActionRoleChange {
			accounts = append(accounts, item.Account)
			if item.AvailableHours <= 0 || item.AvailableHours > 24 {
				return errorx.New("invalid", item.Account+" 的可用工时不合法，无法确认")
			}
		}
	}
	if len(accounts) > 0 {
		existingUsers, err := s.repo.ExistingUserAccounts(ctx, accounts)
		if err != nil {
			return err
		}
		for _, account := range accounts {
			if !existingUsers[strings.TrimSpace(account)] {
				return errorx.New("invalid", "成员账号不存在或已删除，无法确认："+strings.TrimSpace(account))
			}
		}
	}

	return transitionAdjustmentError(s.repo.ApplyConfirmedAdjustment(ctx, adj, items, acc, time.Now()))
}

// RejectAdjustment 驳回。
func (s *Service) RejectAdjustment(ctx context.Context, actor *model.User, req RejectReq, allowConfirm bool) error {
	acc, err := requireActorAccount(actor)
	if err != nil {
		return err
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.AdjustmentID <= 0 {
		return errorx.New("invalid", "调整单 ID 无效")
	}
	if req.Reason == "" || len([]rune(req.Reason)) > 500 {
		return errorx.New("invalid", "驳回原因不能为空且不能超过 500 个字符")
	}
	if !actor.IsSuperAdmin && !allowConfirm {
		return errorx.New("forbidden", "仅 PMO 或管理员可驳回成员调整")
	}
	adj, err := s.repo.FindAdjustmentByID(ctx, req.AdjustmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorx.New("not_found", "调整单不存在")
		}
		return err
	}
	if adj.Status != StatusPending {
		return errorx.New("invalid", "仅待确认调整单可驳回")
	}
	now := time.Now()
	err = s.repo.TransitionPendingAdjustment(ctx, adj, map[string]any{
		"status": StatusRejected, "rejectedBy": acc, "rejectedDate": now,
		"rejectReason": req.Reason, "updatedBy": acc, "updatedDate": now,
	}, &History{
		TeamgroupID: adj.TeamgroupID, EventType: EventReject,
		AdjustmentID: &adj.ID, Actor: acc,
		Summary: "成员调整 " + adj.AdjustNo + " 已驳回：" + req.Reason, CreatedDate: now,
	})
	return transitionAdjustmentError(err)
}

// GetAdjustment 调整单详情（Drawer）。
// 对象级可读：PMO/超管、该小组 PO/敏捷教练、或调整单发起人；禁止仅凭粗粒度 List 跨组读明细。
func (s *Service) GetAdjustment(ctx context.Context, actor *model.User, id int64, canConfirm bool) (*AdjustmentDetailResp, error) {
	acc, err := requireActorAccount(actor)
	if err != nil {
		return nil, err
	}
	adj, err := s.repo.FindAdjustmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.New("not_found", "调整单不存在")
		}
		return nil, err
	}
	tg, err := s.repo.FindTeamgroupByID(ctx, adj.TeamgroupID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	allowed := canConfirm || actor.IsSuperAdmin || adj.SubmittedBy == acc
	if !allowed && tg != nil {
		allowed = canEditTeamgroupObject(actor, tg, false)
	}
	if !allowed {
		return nil, errorx.New("forbidden", "无权查看该调整单")
	}
	teamName := ""
	if tg != nil {
		teamName = tg.Name
	}
	items, err := s.repo.ListItems(ctx, adj.ID)
	if err != nil {
		return nil, err
	}
	accs := []string{adj.SubmittedBy}
	for _, it := range items {
		accs = append(accs, it.Account)
	}
	names, _ := s.repo.ResolveRealnames(ctx, accs)
	lines := make([]AdjustmentLine, 0, len(items))
	for _, it := range items {
		lines = append(lines, AdjustmentLine{
			Account: it.Account, Name: displayName(it.Account, names),
			ActionType: it.ActionType, Role: it.Role, PrevRole: it.PrevRole,
			AvailableHours: it.AvailableHours, PrevHours: it.PrevHours,
		})
	}
	return &AdjustmentDetailResp{
		ID: adj.ID, TeamgroupID: adj.TeamgroupID, TeamName: teamName,
		AdjustNo: adj.AdjustNo, Status: adj.Status, Reason: adj.Reason,
		SubmittedBy: displayName(adj.SubmittedBy, names),
		SubmittedAt: formatTime(adj.CreatedDate),
		Items:       lines, CanConfirm: canConfirm && adj.Status == StatusPending,
	}, nil
}
