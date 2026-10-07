// =============================================================================
// 文件: internal/module/schedule/service_window_authz.go
// 模块: 排期工作台
// 类型: service
// 职责: 版本窗口更新 / 删除的对象级写权限闸门。
//       与既有 Delete 的「只有创建人可以删除」保持同一口径，并额外放行
//       超级管理员与 PMO。Update 与 Delete 共用 canModifyWindow，
//       避免两处各写一套导致口径漂移。
//       Create 保持现状（任何有排期入口的人可建），本轮不加权限点。
// 依赖: internal/model
//       internal/module/demandauthz
//       internal/pkg/errorx
// =============================================================================

package schedule

import (
	"context"
	"strings"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

// WindowWriteDenialMessage 无权改动该版本窗口时统一对外的错误提示。
const WindowWriteDenialMessage = "只有创建人可以修改该版本窗口"

// canModifyWindow 判定 actor 是否可更新或删除指定版本窗口。
// 判定顺序：账号有效 → 创建人 → 超管 → PMO。
func (s *Service) canModifyWindow(ctx context.Context, actor *model.User, window *model.VersionWindow) error {
	if window == nil {
		return errorx.New(errorx.ErrCodeInvalidParam, "窗口不存在")
	}
	account := actor.TrimmedAccount()
	if account == "" {
		return errorx.New(errorx.ErrCodeForbidden, WindowWriteDenialMessage)
	}
	if strings.TrimSpace(window.CreatedBy) == account {
		return nil
	}
	if actor.IsSuperAdmin {
		return nil
	}
	authz := s.demandWriteAuthz()
	if authz != nil && actor.ID > 0 {
		isPMO, err := authz.IsPMORole(ctx, actor.ID)
		if err != nil {
			return err
		}
		if isPMO {
			return nil
		}
	}
	return errorx.New(errorx.ErrCodeForbidden, WindowWriteDenialMessage)
}

// RequireWindowWriteAccess 供 handler 在参数解析之前做前置鉴权，
// 使 403 优先于 400/422。窗口不存在时放行，由 Update / Delete 沿用
// 既有的「窗口不存在」错误，不在此处改变既有语义。
func (s *Service) RequireWindowWriteAccess(ctx context.Context, actor *model.User, windowID uint64) error {
	if windowID == 0 {
		return errorx.New(errorx.ErrCodeInvalidParam, "窗口 ID 无效")
	}
	window, err := s.repo.FindByID(ctx, windowID)
	if err != nil {
		return err
	}
	if window == nil {
		return nil
	}
	return s.canModifyWindow(ctx, actor, window)
}
