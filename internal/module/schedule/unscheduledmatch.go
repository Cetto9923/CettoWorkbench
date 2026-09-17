// =============================================================================
// 文件: internal/module/schedule/unscheduledmatch.go
// 模块: 排期工作台
// 类型: action
// 职责: 业需待排期候选判定（与原 SQL 语义对齐，供多阶段查询复用）。
// 依赖: 无
// =============================================================================

package schedule

// matchUnscheduledBizDemandCandidate 判定单条业需是否满足待排期 SQL 语义。
// 窗口判定走 story→planstory→versionwindowproduct（与旧 EXISTS 一致），不是 zt_demandwindow。
func matchUnscheduledBizDemandCandidate(
	demand ZtDemand,
	productCount int,
	hasChildren bool,
	stories []ZtStory,
	windowByStory map[uint]StoryWindowRef,
	taskStatByStory map[uint]StoryTaskStat,
) bool {
	if !demandStatusUnscheduledEligible(demand.Status) {
		return false
	}
	if productCount <= 0 {
		return false
	}
	if demand.Parent == 0 || demand.Parent == -1 {
		if hasChildren {
			return false
		}
	} else if demand.Parent <= 0 {
		return false
	}
	return demandHasUnscheduledStoryState(stories, windowByStory, taskStatByStory)
}

func demandHasUnscheduledStoryState(
	stories []ZtStory,
	windowByStory map[uint]StoryWindowRef,
	taskStatByStory map[uint]StoryTaskStat,
) bool {
	hasWindow := false
	for _, story := range stories {
		if ref, ok := windowByStory[story.ID]; ok && ref.WindowID > 0 {
			hasWindow = true
			break
		}
	}
	if !hasWindow {
		return true
	}
	if len(stories) == 0 {
		return true
	}
	for _, story := range stories {
		stat := taskStatByStory[story.ID]
		if stat.Total == 0 || stat.Unassigned > 0 {
			return true
		}
	}
	return false
}

func collectUnscheduledBizDemandTopIDs(
	candidates []ZtDemand,
	productCountByDemand map[uint]int,
	hasChildren map[uint]bool,
	storiesByDemand map[uint][]ZtStory,
	windowByStory map[uint]StoryWindowRef,
	taskStatByStory map[uint]StoryTaskStat,
) []uint {
	seen := make(map[uint]bool, len(candidates))
	out := make([]uint, 0, len(candidates))
	for _, demand := range candidates {
		if !matchUnscheduledBizDemandCandidate(
			demand,
			productCountByDemand[demand.ID],
			hasChildren[demand.ID],
			storiesByDemand[demand.ID],
			windowByStory,
			taskStatByStory,
		) {
			continue
		}
		topID := demand.ID
		if demand.Parent > 0 {
			topID = uint(demand.Parent)
		}
		if topID == 0 || seen[topID] {
			continue
		}
		seen[topID] = true
		out = append(out, topID)
	}
	return out
}
