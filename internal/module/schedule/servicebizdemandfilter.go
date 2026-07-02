// =============================================================================
// 文件: internal/module/schedule/servicebizdemandfilter.go
// 模块: 排期工作台
// 类型: action
// 职责: 业务需求待排期筛选树裁剪。
// 依赖: 无
// =============================================================================

package schedule

import "strings"

func (c *bizDemandAssembleContext) filterUnscheduledBizDemandTree(topDemands []ZtDemand) []ZtDemand {
	if len(topDemands) == 0 {
		return []ZtDemand{}
	}

	out := make([]ZtDemand, 0, len(topDemands))
	for _, top := range topDemands {
		children := c.childByParent[int(top.ID)]
		if len(children) == 0 {
			if c.demandMatchesUnscheduled(top) {
				out = append(out, top)
			}
			continue
		}

		matchedChildren := make([]ZtDemand, 0, len(children))
		for _, child := range children {
			if c.demandMatchesUnscheduled(child) {
				matchedChildren = append(matchedChildren, child)
			}
		}
		if len(matchedChildren) == 0 {
			delete(c.childByParent, int(top.ID))
			continue
		}
		c.childByParent[int(top.ID)] = matchedChildren
		out = append(out, top)
	}
	return out
}

func (c bizDemandAssembleContext) demandMatchesUnscheduled(demand ZtDemand) bool {
	if c.productCountByDemand[demand.ID] <= 0 {
		return false
	}
	if !c.demandRelatedToAccount(demand) {
		return false
	}
	return c.demandHasUnscheduledState(demand.ID)
}

func (c bizDemandAssembleContext) demandRelatedToAccount(demand ZtDemand) bool {
	account := strings.TrimSpace(c.account)
	if account == "" {
		return false
	}
	if strings.TrimSpace(demand.AssignedTo) == account {
		return true
	}
	if strings.TrimSpace(demand.BRA) == account {
		return true
	}
	return c.clarifyPMByDemand[demand.ID]
}

func (c bizDemandAssembleContext) demandHasUnscheduledState(demandID uint) bool {
	if !anyDemandHasWindow([]uint{demandID}, c.windowByDemand) {
		return true
	}
	stories := c.storiesByDemand[demandID]
	if len(stories) == 0 {
		return true
	}
	mainStories := filterMainSystemStories(stories)
	taskTotal, unassignedTotal := sumMainSystemTasks(mainStories, c.taskStatByStory)
	if taskTotal == 0 {
		return true
	}
	if unassignedTotal > 0 {
		return true
	}
	return false
}
