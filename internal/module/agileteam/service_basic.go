// =============================================================================
// 文件: internal/module/agileteam/service_basic.go
// 模块: 敏捷小组治理
// 类型: action
// 职责: 基本信息保存（不走成员确认流）。
// 依赖: internal/model
//       internal/pkg/errorx
// =============================================================================

package agileteam

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

// UpdateBasicInfo 保存口号/信条/Logo/名称。
//
// 2026-08-27 B 决策（AGENTS.md §4.7）- PMO 编辑基本信息绕过组织级教练确认流：
//   - allowGlobal 只允许由 Handler 根据 AgileTeamConfirm（PMO）粗权限传入；
//   - PMO 编辑直接生效，**不**生成任何 adjustment 单，**不**进"待我确认"列表；
//   - 普通 AgileTeamUpdate 用户仍必须命中具体 teamgroup 的 PO/manager 对象关系；
//   - History 区分两个细粒度事件（updatedByPmo / updatedByOwner），便于审计与
//     "待我确认" UI 直接按事件类型排除 PMO 直接编辑。
//
// 适用范围：**仅 UpdateBasicInfo**。SubmitAdjustment / ConfirmAdjustment / RejectAdjustment
// 等涉及成员调整的接口不享受此旁路，仍走原有 confirm/reject 流程。
func (s *Service) UpdateBasicInfo(ctx context.Context, actor *model.User, req UpdateBasicReq, allowGlobal bool) error {
	acc, err := requireActorAccount(actor)
	if err != nil {
		return err
	}
	if err := validateBasicWriteBoundary(req); err != nil {
		return err
	}
	row, err := s.repo.FindTeamgroupByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorx.New("not_found", "敏捷小组不存在")
		}
		return err
	}
	if err := requireTeamgroupObjectEdit(actor, row, allowGlobal); err != nil {
		return err
	}

	// 主体标签：PMO 旁路写"PMO 直接编辑"，PO/Manager 写"PO/教练编辑"。
	// 历史文案显式标注主体，避免与"成员调整已确认生效"等 confirm 事件混淆。
	actorTag := "[PO/教练编辑] "
	eventType := EventUpdateByOwner
	if allowGlobal || actor.IsSuperAdmin {
		actorTag = "[PMO直接编辑] "
		eventType = EventUpdateByPMO
	}

	summary := actorTag + "基本信息已更新"
	if row.Name != req.Name {
		summary = fmt.Sprintf("%s团队名称由「%s」改为「%s」", actorTag, row.Name, req.Name)
	}
	if req.ParentID != nil && *req.ParentID != row.Parent {
		summary = fmt.Sprintf("%s；父级小组已更新", summary)
	}
	history := &History{
		TeamgroupID: req.ID,
		EventType:   eventType,
		Actor:       acc,
		Summary:     summary,
		CreatedDate: time.Now(),
	}
	if err := s.repo.UpdateTeamgroupBasicAtomic(
		ctx,
		req.ID,
		req.Name,
		req.Slogan,
		req.Declaration,
		req.Logo,
		req.ParentID,
		history,
	); err != nil {
		switch {
		case errors.Is(err, errTeamgroupMissing):
			return errorx.New("not_found", "敏捷小组不存在或已被删除")
		case errors.Is(err, errTeamgroupSelfParent):
			return errorx.New("invalid", "不能将自己设为父级小组")
		case errors.Is(err, errTeamgroupParentMissing):
			return errorx.New("invalid", "父级小组不存在")
		case errors.Is(err, errTeamgroupParentCycle):
			return errorx.New("invalid", "不能将子孙小组设为父级")
		default:
			return err
		}
	}
	return nil
}
