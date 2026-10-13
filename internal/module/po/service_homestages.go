// =============================================================================
// 文件: internal/module/po/service_homestages.go
// 模块: PO 工作台
// 类型: action
// 职责: 首页价值流阶段统计与挂起（价值流并集）计数。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
)

// buildValueStreamStages 按阶段统计业需/研需；suspended 时仅业需且 hang='1'，研需恒为 0。
func (s *Service) buildValueStreamStages(ctx context.Context, account string, suspended bool) ([]ValueStreamStage, error) {
	stages := make([]ValueStreamStage, 0, len(valueStreamStages))
	allIdx := -1
	for _, def := range valueStreamStages {
		if def.status == "all" {
			allIdx = len(stages)
			stages = append(stages, ValueStreamStage{Label: def.label, Status: def.status})
			continue
		}

		filter, ok := mysqlStageFilters[def.status]
		if !ok {
			stages = append(stages, ValueStreamStage{Label: def.label, Status: def.status})
			continue
		}
		filter.suspended = suspended

		var demand, story int64
		n, countErr := s.repo.CountRoleDemands(ctx, account, filter)
		if countErr != nil {
			return nil, countErr
		}
		demand = n
		if !suspended {
			if filter.scheduleIncomplete {
				sn, storyErr := s.repo.CountScheduleStories(ctx, account)
				if storyErr != nil {
					return nil, storyErr
				}
				story = sn
			}
			if filter.deliverStories {
				sn, storyErr := s.repo.CountDeliverStories(ctx, account)
				if storyErr != nil {
					return nil, storyErr
				}
				story = sn
			}
		}
		stages = append(stages, ValueStreamStage{
			Label:       def.label,
			Status:      def.status,
			Count:       demand + story,
			DemandCount: demand,
			StoryCount:  story,
		})
	}

	if allIdx >= 0 {
		demandSum, storySum, allErr := s.countAllStageUniq(ctx, account, suspended)
		if allErr != nil {
			return nil, allErr
		}
		stages[allIdx].DemandCount = demandSum
		stages[allIdx].StoryCount = storySum
		stages[allIdx].Count = demandSum + storySum
	}
	return stages, nil
}

// countAllStageUniq 各阶段只查 ID，按 kind+id 去重后返回业需/研需数量。
// suspended 时业需叠加 hang='1'，不计入研需。
func (s *Service) countAllStageUniq(ctx context.Context, account string, suspended bool) (demandSum, storySum int64, err error) {
	seenDemand := make(map[int]struct{})
	seenStory := make(map[int]struct{})
	for _, def := range valueStreamStages {
		if def.status == "all" {
			continue
		}
		filter, ok := mysqlStageFilters[def.status]
		if !ok {
			continue
		}
		filter.suspended = suspended
		ids, idErr := s.repo.FindRoleDemandIDs(ctx, account, filter)
		if idErr != nil {
			return 0, 0, idErr
		}
		for _, id := range ids {
			seenDemand[id] = struct{}{}
		}
		if suspended {
			continue
		}
		if filter.scheduleIncomplete {
			storyIDs, storyErr := s.repo.FindScheduleStoryIDs(ctx, account)
			if storyErr != nil {
				return 0, 0, storyErr
			}
			for _, id := range storyIDs {
				seenStory[id] = struct{}{}
			}
		}
		if filter.deliverStories {
			storyIDs, storyErr := s.repo.FindDeliverStoryIDs(ctx, account)
			if storyErr != nil {
				return 0, 0, storyErr
			}
			for _, id := range storyIDs {
				seenStory[id] = struct{}{}
			}
		}
	}
	return int64(len(seenDemand)), int64(len(seenStory)), nil
}

// countSuspendedInValueStream 价值流并集内挂起业需数（芯片角标）。
func (s *Service) countSuspendedInValueStream(ctx context.Context, account string) (int64, error) {
	demandSum, _, err := s.countAllStageUniq(ctx, account, true)
	return demandSum, err
}
