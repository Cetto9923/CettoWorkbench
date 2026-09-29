// =============================================================================
// 文件: internal/module/po/service_detail_writeauthz.go
// 模块: PO 工作台
// 类型: service
// 职责: 业务需求对象级「写权限」闸门。读授权（service_detail_authz.go）与写
//       授权共用 internal/module/demandauthz 的同一套关系判断，避免口径漂移。
// 依赖: internal/module/demandauthz
//       internal/module/po/primaryaction
// =============================================================================

package po

import (
	"context"

	"workbench/internal/model"
	"workbench/internal/module/demandauthz"
	"workbench/internal/module/po/primaryaction"
	"workbench/internal/pkg/errorx"
)

// ReadOnlyDemandReason 是「可见但无写权限」时统一下发给前端的说明。
const ReadOnlyDemandReason = "仅可查看，无该业务需求写权限"

// canWriteDemand 判定 actor 是否可对 demandID 执行写操作（写库 / 同步禅道）。
// 口径：超级管理员、PMO、个人干系人、团队长管辖；仅持 perm.ScheduleList 视为只读。
func (s *DetailService) canWriteDemand(ctx context.Context, actor *model.User, demandID uint) (bool, error) {
	if s == nil || s.repo == nil {
		return false, nil
	}
	return s.repo.authzRepo().CanWriteDemand(ctx, actor, demandID)
}

// RequireDemandWrite 是写权限的强制入口：无写权限返回 errorx.Forbidden，
// 调用方据此返回 403，且必须在任何写库 / 同步禅道动作之前调用。
func (s *DetailService) RequireDemandWrite(ctx context.Context, actor *model.User, demandID uint) error {
	if demandID == 0 {
		return errorx.New(errorx.ErrCodeInvalidParam, "需求 ID 无效")
	}
	ok, err := s.canWriteDemand(ctx, actor, demandID)
	if err != nil {
		return err
	}
	if !ok {
		return errorx.New(errorx.ErrCodeForbidden, demandauthz.WriteDenialMessage)
	}
	return nil
}

// gatePrimaryActionForReadOnly 把已派生的主操作降级为不可点击。
// 无写权限时不保留跳转 URL，避免前端出现可点的写入口。
func gatePrimaryActionForReadOnly(pa *primaryaction.PrimaryAction) {
	if pa == nil {
		return
	}
	pa.Enabled = false
	pa.URL = ""
	pa.Reason = ReadOnlyDemandReason
}
