// =============================================================================
// 文件: internal/module/po/service_homestages.go
// 模块: PO 工作台
// 类型: action
// 职责: 首页价值流阶段统计与焦点角标（挂起 / 关注 / 超期 / 我负责 / 待我处理）计数。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"strings"
)

// focusIncludesStories 空焦点、超期、我负责、待我处理需合并独立研需；挂起/关注仅业需。
func focusIncludesStories(focus string) bool {
	return focus == FocusNone || focus == FocusOverdue || focus == FocusMyManaged || focus == FocusMyAction
}

// applyFocusFilterFlags 将焦点芯片写入阶段 filter（挂起 / 超期 / 我负责 / 待我处理叠加）。
func applyFocusFilterFlags(filter *mysqlStageFilter, focus string) {
	filter.suspended = focus == FocusSuspended
	filter.overdue = focus == FocusOverdue
	filter.myManaged = focus == FocusMyManaged
	filter.myAction = focus == FocusMyAction
}

// buildValueStreamStages 按阶段统计业需/研需；focus 决定候选口径（空=角色；suspended/my_related 仅业需；overdue/my_managed/my_action 含研需）。
func (s *Service) buildValueStreamStages(ctx context.Context, account, focus string) ([]ValueStreamStage, error) {
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
		applyFocusFilterFlags(&filter, focus)

		var demand, story int64
		var countErr error
		switch focus {
		case FocusMyRelated:
			demand, countErr = s.repo.CountStarredDemands(ctx, account, filter)
		default:
			demand, countErr = s.repo.CountRoleDemands(ctx, account, filter)
		}
		if countErr != nil {
			return nil, countErr
		}
		if focusIncludesStories(focus) {
			if filter.scheduleIncomplete {
				sn, storyErr := s.repo.CountScheduleStories(ctx, account, filter)
				if storyErr != nil {
					return nil, storyErr
				}
				story = sn
			}
			if filter.deliverStories {
				sn, storyErr := s.repo.CountDeliverStories(ctx, account, filter)
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
		demandSum, storySum, allErr := s.countAllStageUniq(ctx, account, focus)
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
// focus=suspended/my_related 时仅业需；overdue/my_managed/my_action 含独立研需。
func (s *Service) countAllStageUniq(ctx context.Context, account, focus string) (demandSum, storySum int64, err error) {
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
		applyFocusFilterFlags(&filter, focus)
		var ids []int
		var idErr error
		switch focus {
		case FocusMyRelated:
			ids, idErr = s.repo.FindStarredDemandIDs(ctx, account, filter)
		default:
			ids, idErr = s.repo.FindRoleDemandIDs(ctx, account, filter)
		}
		if idErr != nil {
			return 0, 0, idErr
		}
		for _, id := range ids {
			seenDemand[id] = struct{}{}
		}
		if !focusIncludesStories(focus) {
			continue
		}
		if filter.scheduleIncomplete {
			storyIDs, storyErr := s.repo.FindScheduleStoryIDs(ctx, account, filter)
			if storyErr != nil {
				return 0, 0, storyErr
			}
			for _, id := range storyIDs {
				seenStory[id] = struct{}{}
			}
		}
		if filter.deliverStories {
			storyIDs, storyErr := s.repo.FindDeliverStoryIDs(ctx, account, filter)
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
	demandSum, _, err := s.countAllStageUniq(ctx, account, FocusSuspended)
	return demandSum, err
}

// countOverdueInValueStream 价值流并集内超期业需+独立研需数（芯片角标）。
func (s *Service) countOverdueInValueStream(ctx context.Context, account string) (int64, error) {
	demandSum, storySum, err := s.countAllStageUniq(ctx, account, FocusOverdue)
	if err != nil {
		return 0, err
	}
	return demandSum + storySum, nil
}

// countMyManagedInValueStream 价值流并集内「我负责」业需+独立研需数（芯片角标）。
func (s *Service) countMyManagedInValueStream(ctx context.Context, account string) (int64, error) {
	demandSum, storySum, err := s.countAllStageUniq(ctx, account, FocusMyManaged)
	if err != nil {
		return 0, err
	}
	return demandSum + storySum, nil
}

// countMyActionInValueStream 价值流并集内「待我处理」业需+独立研需数（芯片角标）。
func (s *Service) countMyActionInValueStream(ctx context.Context, account string) (int64, error) {
	demandSum, storySum, err := s.countAllStageUniq(ctx, account, FocusMyAction)
	if err != nil {
		return 0, err
	}
	return demandSum + storySum, nil
}

// countMyRelatedInValueStream 价值流并集内「我关注的」业需数（价值流 ∩ zt_starinfo）。
func (s *Service) countMyRelatedInValueStream(ctx context.Context, account string) (int64, error) {
	if strings.TrimSpace(account) == "" {
		return 0, nil
	}
	demandSum, _, err := s.countAllStageUniq(ctx, account, FocusMyRelated)
	return demandSum, err
}
