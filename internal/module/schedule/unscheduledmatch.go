// =============================================================================
// 文件: internal/module/schedule/unscheduledmatch.go
// 模块: 排期工作台
// 类型: action
// 职责: 业需待排期候选判定（与原 SQL 语义对齐，供多阶段查询复用）。
// 依赖: 无
// =============================================================================

package schedule

// matchUnscheduledBizDemandCandidate 判定单条业需是否满足待排期 SQL 语义。
// 窗口判定统一按需求直接绑定（zt_demandwindow），未直接绑定任何窗口即为待排期。
func matchUnscheduledBizDemandCandidate(
	demand ZtDemand,
	productCount int,
	hasChildren bool,
	hasDemandWindow bool,
	stories []ZtStory,
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
	if !hasDemandWindow {
		return true
	}
	// 与排期阶段一致：仅看主系统研需；任一条未建任务或未指派即待排期。
	return anyMainSystemStoryNeedsScheduling(filterMainSystemStories(stories), taskStatByStory)
}

func collectUnscheduledBizDemandTopIDs(
	candidates []ZtDemand,
	productCountByDemand map[uint]int,
	hasChildren map[uint]bool,
	demandHasWindow map[uint]bool,
	storiesByDemand map[uint][]ZtStory,
	taskStatByStory map[uint]StoryTaskStat,
) []uint {
	seen := make(map[uint]bool, len(candidates))
	out := make([]uint, 0, len(candidates))
	for _, demand := range candidates {
		if !matchUnscheduledBizDemandCandidate(
			demand,
			productCountByDemand[demand.ID],
			hasChildren[demand.ID],
			demandHasWindow[demand.ID],
			storiesByDemand[demand.ID],
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
