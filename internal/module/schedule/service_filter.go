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
func (s *Service) GetBizDemandFilterCounts(ctx context.Context, actor *model.User, activeFilter, reuseFilter string, reuseTotal int64) (FilterCounts, error) {
	account := actorAccount(actor)
	if account == "" {
		return FilterCounts{}, nil
	}

	poolIDs, err := s.repo.GetUserDemandPools(ctx, account)
	if err != nil {
		return FilterCounts{}, err
	}
	return s.repo.GetBizDemandFilterCounts(ctx, poolIDs, account, activeFilter, reuseFilter, reuseTotal)
}

// GetIndependentFilterCounts 查询独立研发需求各快捷筛选项数量。
func (s *Service) GetIndependentFilterCounts(ctx context.Context, actor *model.User, reuseFilter string, reuseTotal int64) (FilterCounts, error) {
	account := actorAccount(actor)
	if account == "" {
		return FilterCounts{}, nil
	}

	productIDs, err := s.getVisibleProductIDs(ctx, account)
	if err != nil {
		return FilterCounts{}, err
	}
	return s.repo.GetIndependentFilterCounts(ctx, productIDs, account, reuseFilter, reuseTotal)
}

// GetScheduleFilterCounts 按当前 tab 查询业务需求或研发需求快捷筛选项数量（角标 ajax）。
func (s *Service) GetScheduleFilterCounts(ctx context.Context, actor *model.User, req FilterCountsReq) (FilterCountsResp, error) {
	activeFilter := NormalizeDemandFilter(req.Filter)
	tab := NormalizeFilterCountsTab(req.Tab)

	resp := FilterCountsResp{Success: true}
	switch tab {
	case FilterCountsTabStory:
		story, err := s.GetIndependentFilterCounts(ctx, actor, req.ReuseStoryFilter, req.ReuseStoryTotal)
		if err != nil {
			return FilterCountsResp{}, err
		}
		resp.Story = story
	default:
		demand, err := s.GetBizDemandFilterCounts(ctx, actor, activeFilter, req.ReuseDemandFilter, req.ReuseDemandTotal)
		if err != nil {
			return FilterCountsResp{}, err
		}
		resp.Demand = demand
	}
	return resp, nil
}
