// =============================================================================
// 文件: internal/module/build/serviceauthz.go
// 模块: 版本管理
// 类型: action
// 职责: 关联或取消关联前校验所有研发需求来源业务需求的写权限。
// 依赖: internal/model, internal/module/testtask, internal/pkg/errorx
// =============================================================================
package build

import (
	"context"
	"fmt"
	"strings"

	"workbench/internal/model"
	"workbench/internal/module/testtask"
	"workbench/internal/pkg/errorx"
)

// 防止借版本接口修改无权操作的业务需求；全部授权完成后才请求禅道。
func (s *Service) requireStoryWriteAccess(ctx context.Context, actor *model.User, stories string) error {
	rows, err := s.repo.findStoryOrigins(ctx, strings.Split(stories, ","))
	if err != nil {
		return err
	}
	authz := testtask.NewService(testtask.NewRepo(s.repo.db), nil, nil, nil)
	for _, row := range rows {
		if row.FromDemand == 0 {
			continue
		}
		if err := authz.RequireDemandWriteAccess(ctx, actor, row.FromDemand); err != nil {
			if biz, ok := errorx.IsBizError(err); ok && biz.Code == errorx.ErrCodeForbidden {
				return errorx.New(errorx.ErrCodeForbidden, fmt.Sprintf("无权操作业务需求 US%d：%s", row.FromDemand, biz.Msg))
			}
			return err
		}
	}
	return nil
}
