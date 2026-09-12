// =============================================================================
// 文件: internal/module/kanban/service.go
// 模块: 工作看板
// 类型: readonly
// 职责: 需求看板业务编排。
// 依赖: internal/model
// =============================================================================

package kanban

import (
	"context"

	"workbench/internal/model"
)

// Service 看板业务服务。
type Service struct {
	repo *Repo
}

// NewService 创建 Service。
func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

// ListMyTeamgroups 返回当前用户所属敏捷小组（按加入时间倒序）。
func (s *Service) ListMyTeamgroups(ctx context.Context, actor *model.User) ([]TeamgroupItem, error) {
	if actor == nil {
		return []TeamgroupItem{}, nil
	}
	return s.repo.ListUserTeamgroups(ctx, actor.Account)
}
