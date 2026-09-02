// =============================================================================
// 文件: internal/module/schedule/service_filter.go
// 模块: 排期工作台
// 类型: action
// 职责: 列表快捷筛选数量统计。
// 依赖: internal/model
//       internal/module/schedule/repo.go
// =============================================================================

package schedule

import (
	"context"

	"workbench/internal/model"
)

// GetBizDemandFilterCounts 查询业务需求各快捷筛选项数量。
func (s *Service) GetBizDemandFilterCounts(ctx context.Context, actor *model.User, activeFilter string) (FilterCounts, error) {
	account := actorAccount(actor)
	if account == "" {
		return FilterCounts{}, nil
	}

	poolIDs, err := s.repo.GetUserDemandPools(ctx, account)
	if err != nil {
		return FilterCounts{}, err
	}
	return s.repo.GetBizDemandFilterCounts(ctx, poolIDs, account, activeFilter)
}

// GetIndependentFilterCounts 查询独立研发需求各快捷筛选项数量。
func (s *Service) GetIndependentFilterCounts(ctx context.Context, actor *model.User) (FilterCounts, error) {
	account := actorAccount(actor)
	if account == "" {
		return FilterCounts{}, nil
	}

	productIDs, err := s.getVisibleProductIDs(ctx, account)
	if err != nil {
		return FilterCounts{}, err
	}
	return s.repo.GetIndependentFilterCounts(ctx, productIDs, account)
}
