// =============================================================================
// 文件: internal/module/schedule/service_story_authz.go
// 模块: 排期工作台
// 类型: service
// 职责: 独立研发需求（zt_story）写路径的对象级写权限闸门。
//       口径：超级管理员、PMO，或对该需求所属产品可见（复用 getVisibleProductIDs）的人。
//       刻意不套用业务需求的「干系人」关系判断：研发需求的对象是产品而非需求，
//       两套关系各自成立，不合并口径也不新增 perm.* 权限点。
//       仅有 perm.ScheduleList 而产品不可见者只读。
// 依赖: internal/model
//       internal/module/demandauthz
//       internal/pkg/errorx
// =============================================================================

package schedule

import (
	"context"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

// StoryWriteDenialMessage 无权写该研发需求时统一对外的错误提示。
const StoryWriteDenialMessage = "无权修改该研发需求"

// RequireStoryWriteAccess 是研发需求写入口（排期保存 / 维护任务保存）的对象级写权限闸门。
// 无写权限时返回 errorx.Forbidden（对外 403），调用方必须在任何写库或同步禅道
// 的动作之前调用：既不落库也不调用禅道接口。
func (s *Service) RequireStoryWriteAccess(ctx context.Context, actor *model.User, storyID uint) error {
	if storyID == 0 {
		return errorx.New(errorx.ErrCodeInvalidParam, "研发需求 ID 无效")
	}
	account := actor.TrimmedAccount()
	if actor == nil || account == "" {
		return errorx.New(errorx.ErrCodeForbidden, StoryWriteDenialMessage)
	}
	if actor.IsSuperAdmin {
		return nil
	}
	// 组织角色 PMO：全量放行，与业务需求写权限同一套关系查询底座。
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
	// 产品可见性：与独立研发需求列表 ListIndependentStories 完全同一口径，
	// 读得到列表才谈得上写，避免出现「列表可见但保存被拒」或反之的割裂。
	productID, err := s.repo.GetStoryProductID(ctx, storyID)
	if err != nil {
		return err
	}
	visibleIDs, err := s.getVisibleProductIDs(ctx, account)
	if err != nil {
		return err
	}
	for _, id := range visibleIDs {
		if id == productID {
			return nil
		}
	}
	return errorx.New(errorx.ErrCodeForbidden, StoryWriteDenialMessage)
}
