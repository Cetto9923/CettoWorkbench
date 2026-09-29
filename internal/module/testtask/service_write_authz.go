// =============================================================================
// 文件: internal/module/testtask/service_write_authz.go
// 模块: 提测办理
// 类型: service
// 职责: 提测写入口（创建版本 / 创建测试单）的业务需求对象级写权限闸门。
//       以 demandID 为对象，直接复用第 3 轮抽出的 internal/module/demandauthz，
//       与排期工作台业需排期保存完全同一套关系口径，不另立一套判断。
//       无权时既不写库也不请求禅道。
// 依赖: internal/model
//       internal/module/demandauthz
//       internal/pkg/errorx
// =============================================================================

package testtask

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

// RequireDemandWriteAccess 是提测写入口的对象级写权限闸门。
// 调用方必须在任何参数解析、禅道调用之前执行：无权时返回 errorx.Forbidden
// （对外 403），既不落库也不发起禅道请求。
func (s *Service) RequireDemandWriteAccess(ctx context.Context, actor *model.User, demandID uint) error {
	if demandID == 0 {
		return errorx.New(errorx.ErrCodeInvalidParam, "需求 ID 无效")
	}
	if actor == nil {
		return errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	ok, err := s.demandWriteAuthz().CanWriteDemand(ctx, actor, demandID)
	if err != nil {
		return err
	}
	if !ok {
		return errorx.New(errorx.ErrCodeForbidden, demandauthz.WriteDenialMessage)
	}
	return nil
}
