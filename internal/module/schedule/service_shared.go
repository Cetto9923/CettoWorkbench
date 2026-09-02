// =============================================================================
// 文件: internal/module/schedule/service_shared.go
// 模块: 排期工作台
// 类型: action
// 职责: 业需与独立研发需求列表共享辅助函数。
// 依赖: internal/module/schedule/repo.go
// =============================================================================

package schedule

import (
	"context"
	"fmt"
	"strings"
)

func (s *Service) loadTeamgroupDisplayNamesByIDs(ctx context.Context, teamgroupIDs []uint) (map[uint]string, error) {
	if len(teamgroupIDs) == 0 {
		return map[uint]string{}, nil
	}

	groups, err := s.repo.FindTeamgroupsByIDs(ctx, teamgroupIDs)
	if err != nil {
		return nil, err
	}
	nameByID := make(map[uint]string, len(groups))
	parentIDs := make([]uint, 0)
	parentSeen := make(map[uint]struct{})
	for _, group := range groups {
		nameByID[group.ID] = strings.TrimSpace(group.Name)
		if group.Parent == 0 {
			continue
		}
		if _, ok := parentSeen[group.Parent]; ok {
			continue
		}
		parentSeen[group.Parent] = struct{}{}
		parentIDs = append(parentIDs, group.Parent)
	}
	if len(parentIDs) > 0 {
		parents, err := s.repo.FindTeamgroupsByIDs(ctx, parentIDs)
		if err != nil {
			return nil, err
		}
		parentNameByID := make(map[uint]string, len(parents))
		for _, parent := range parents {
			parentNameByID[parent.ID] = strings.TrimSpace(parent.Name)
		}
		for _, group := range groups {
			name := strings.TrimSpace(group.Name)
			if group.Parent > 0 {
				if parentName := parentNameByID[group.Parent]; parentName != "" {
					name = fmt.Sprintf("%s / %s", parentName, name)
				}
			}
			nameByID[group.ID] = name
		}
	}
	return nameByID, nil
}
func allStoriesHaveNoWindow(stories []ZtStory, windowByStory map[uint]StoryWindowRef) bool {
	for _, story := range stories {
		if ref, ok := windowByStory[story.ID]; ok && ref.WindowID > 0 {
			return false
		}
	}
	return true
}

func anyStoryHasWindow(stories []ZtStory, windowByStory map[uint]StoryWindowRef) bool {
	return !allStoriesHaveNoWindow(stories, windowByStory)
}

func anyDemandHasWindow(demandIDs []uint, windowByDemand map[uint]DemandWindowRef) bool {
	for _, demandID := range demandIDs {
		if ref, ok := windowByDemand[demandID]; ok && ref.WindowID > 0 {
			return true
		}
	}
	return false
}

func anyDemandOrStoryHasWindow(
	demandIDs []uint,
	stories []ZtStory,
	windowByDemand map[uint]DemandWindowRef,
	windowByStory map[uint]StoryWindowRef,
) bool {
	return anyDemandHasWindow(demandIDs, windowByDemand) || anyStoryHasWindow(stories, windowByStory)
}

func calcDemandWindowPhase(
	demandIDs []uint,
	stories []ZtStory,
	windowByDemand map[uint]DemandWindowRef,
	windowByStory map[uint]StoryWindowRef,
) string {
	if !anyDemandOrStoryHasWindow(demandIDs, stories, windowByDemand, windowByStory) {
		return ""
	}
	return calcSchedulingWindowPhase(pickDemandWindowID(demandIDs, stories, windowByDemand, windowByStory), len(stories))
}

func calcSchedulingWindowPhase(windowID uint, storyCount int) string {
	if windowID == 0 {
		return ""
	}
	if storyCount > 0 {
		return WindowPhaseFinal
	}
	return WindowPhaseInitial
}

func canEditSchedulingWindow(windowID uint, storyCount int) bool {
	return windowID == 0 || storyCount == 0
}

func sumMainSystemTasks(stories []ZtStory, taskStatByStory map[uint]StoryTaskStat) (int, int) {
	taskTotal := 0
	unassignedTotal := 0
	for _, story := range stories {
		stat := taskStatByStory[story.ID]
		taskTotal += stat.Total
		unassignedTotal += stat.Unassigned
	}
	return taskTotal, unassignedTotal
}

func pickBizWindowName(stories []ZtStory, windowByStory map[uint]StoryWindowRef) string {
	for _, story := range stories {
		ref, ok := windowByStory[story.ID]
		if !ok || ref.WindowID == 0 {
			continue
		}
		name := strings.TrimSpace(ref.WindowName)
		if name != "" {
			return name
		}
	}
	return ""
}

func pickDemandWindowID(
	demandIDs []uint,
	stories []ZtStory,
	windowByDemand map[uint]DemandWindowRef,
	windowByStory map[uint]StoryWindowRef,
) uint {
	for _, demandID := range demandIDs {
		ref, ok := windowByDemand[demandID]
		if ok && ref.WindowID > 0 {
			return ref.WindowID
		}
	}
	for _, story := range stories {
		ref, ok := windowByStory[story.ID]
		if ok && ref.WindowID > 0 {
			return ref.WindowID
		}
	}
	return 0
}

func pickDemandWindowName(
	demandIDs []uint,
	stories []ZtStory,
	windowByDemand map[uint]DemandWindowRef,
	windowByStory map[uint]StoryWindowRef,
) string {
	for _, demandID := range demandIDs {
		ref, ok := windowByDemand[demandID]
		if !ok || ref.WindowID == 0 {
			continue
		}
		if name := strings.TrimSpace(ref.WindowName); name != "" {
			return name
		}
	}
	return pickBizWindowName(stories, windowByStory)
}

func resolveDemandOwner(bra string, realnameByAccount map[string]string) string {
	bra = strings.TrimSpace(bra)
	if bra == "" {
		return "待分配"
	}
	return resolveRealname(bra, realnameByAccount)
}

func resolveRealname(account string, realnameByAccount map[string]string) string {
	account = strings.TrimSpace(account)
	if account == "" {
		return ""
	}
	if name := strings.TrimSpace(realnameByAccount[account]); name != "" {
		return name
	}
	return account
}
func pluckStoryIDs(stories []ZtStory) []uint {
	ids := make([]uint, 0, len(stories))
	for _, story := range stories {
		if story.ID == 0 {
			continue
		}
		ids = append(ids, story.ID)
	}
	return ids
}
func appendUniqueUint(ids []uint, id uint) []uint {
	if id == 0 {
		return ids
	}
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	return append(ids, id)
}
