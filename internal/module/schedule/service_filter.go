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
	"strings"

	"workbench/internal/model"
)

// GetBizDemandFilterCounts 查询业务需求各快捷筛选项数量。
func (s *Service) GetBizDemandFilterCounts(ctx context.Context, actor *model.User, activeFilter, reuseFilter string, reuseTotal int64, req ...FilterCountsReq) (FilterCounts, error) {
	account := actorAccount(actor)
	if account == "" {
		return FilterCounts{}, nil
	}

	poolIDs, err := s.repo.GetUserDemandPools(ctx, account)
	if err != nil {
		return FilterCounts{}, err
	}
	var countsReq FilterCountsReq
	if len(req) > 0 {
		countsReq = req[0]
	}
	return s.repo.GetBizDemandFilterCounts(ctx, poolIDs, account, activeFilter, reuseFilter, reuseTotal, countsReq)
}

// GetIndependentFilterCounts 查询独立研发需求各快捷筛选项数量。
func (s *Service) GetIndependentFilterCounts(ctx context.Context, actor *model.User, reuseFilter string, reuseTotal int64, req ...FilterCountsReq) (FilterCounts, error) {
	account := actorAccount(actor)
	if account == "" {
		return FilterCounts{}, nil
	}

	productIDs, err := s.getVisibleProductIDs(ctx, account)
	if err != nil {
		return FilterCounts{}, err
	}
	var countsReq FilterCountsReq
	if len(req) > 0 {
		countsReq = req[0]
	}
	return s.repo.GetIndependentFilterCounts(ctx, productIDs, account, reuseFilter, reuseTotal, countsReq)
}

// GetScheduleFilterCounts 按当前 tab 查询业务需求或研发需求快捷筛选项数量（角标 ajax）。
func (s *Service) GetScheduleFilterCounts(ctx context.Context, actor *model.User, req FilterCountsReq) (FilterCountsResp, error) {
	if strings.TrimSpace(req.Stages) == "" && strings.TrimSpace(req.Stage) != "" {
		req.Stages = strings.TrimSpace(req.Stage)
	}
	activeFilter := NormalizeDemandFilterWithWindows(req.Filter, req.Windows)
	tab := NormalizeFilterCountsTab(req.Tab)

	resp := FilterCountsResp{Success: true}
	switch tab {
	case FilterCountsTabStory:
		story, err := s.GetIndependentFilterCounts(ctx, actor, req.ReuseStoryFilter, req.ReuseStoryTotal, req)
		if err != nil {
			return FilterCountsResp{}, err
		}
		resp.Story = story
	default:
		demand, err := s.GetBizDemandFilterCounts(ctx, actor, activeFilter, req.ReuseDemandFilter, req.ReuseDemandTotal, req)
		if err != nil {
			return FilterCountsResp{}, err
		}
		resp.Demand = demand
	}
	return resp, nil
}
