// =============================================================================
// 文件: internal/module/schedule/service_scheduling_authz.go
// 模块: 排期工作台
// 类型: service
// 职责: 排期写路径（写库 / 同步禅道）的业务需求对象级写权限闸门。
//       关系判断与 PO 工作台详情读授权共用 internal/module/demandauthz，
//       读写同一套口径，避免两处漂移。
// 依赖: internal/module/demandauthz
//       internal/pkg/errorx
// =============================================================================

package schedule

import (
	"context"

	"workbench/internal/model"
	"workbench/internal/module/demandauthz"
	"workbench/internal/pkg/errorx"
)

// demandWriteAuthz 返回共用的需求对象级授权仓储。
func (s *Service) demandWriteAuthz() *demandauthz.Repo {
	if s == nil || s.repo == nil {
		return nil
	}
	return demandauthz.New(s.repo.db)
}

// CanWriteDemand 判定 actor 是否可对 demandID 执行排期写操作。
// 口径：超级管理员、PMO、个人干系人、团队长管辖；
// 仅持 perm.ScheduleList 的用户视为只读，不授予任何写权限。
func (s *Service) CanWriteDemand(ctx context.Context, actor *model.User, demandID uint) (bool, error) {
	return s.demandWriteAuthz().CanWriteDemand(ctx, actor, demandID)
}

// RequireDemandWriteAccess 是排期写入口的对象级写权限闸门。
// 无写权限时返回 errorx.Forbidden（对外 403），调用方必须在任何写库或
// 同步禅道的动作之前调用：既不落库也不调用禅道接口。
func (s *Service) RequireDemandWriteAccess(ctx context.Context, actor *model.User, demandID uint) error {
	if demandID == 0 {
		return errorx.New(errorx.ErrCodeInvalidParam, "业需 ID 无效")
	}
	ok, err := s.CanWriteDemand(ctx, actor, demandID)
	if err != nil {
		return err
	}
	if !ok {
		return errorx.New(errorx.ErrCodeForbidden, demandauthz.WriteDenialMessage)
	}
	return nil
}
