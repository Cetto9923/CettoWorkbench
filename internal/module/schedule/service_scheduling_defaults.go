package schedule

import (
	"fmt"
	"strings"
)

func buildDemandSchedulingStoryDefaults(
	detail *DemandSchedulingDetail,
	clarifyRows []DemandSchedulingClarifyDefault,
	userStories []UserStoryItem,
	plans []SchedulingWindowProductPlan,
	realnameByAccount map[string]string,
) []DemandSchedulingStoryDefault {
	if detail == nil || len(clarifyRows) == 0 {
		return []DemandSchedulingStoryDefault{}
	}

	planByProduct := make(map[uint]SchedulingWindowProductPlan)
	for _, plan := range plans {
		if plan.WindowID != detail.WindowID || plan.ProductID == 0 {
			continue
		}
		if _, exists := planByProduct[plan.ProductID]; !exists {
			planByProduct[plan.ProductID] = plan
		}
	}

	storiesByProduct := make(map[uint][]string)
	estimateByProduct := make(map[uint]int)
	storyIndexByProduct := make(map[uint]int)
	for _, story := range userStories {
		if story.ProductID == 0 {
			continue
		}
		estimateByProduct[story.ProductID] += story.EffectivePoint
		role := strings.TrimSpace(story.Role)
		gv := strings.TrimSpace(story.GV)
		if role == "" && gv == "" {
			continue
		}
		storyIndexByProduct[story.ProductID]++
		storiesByProduct[story.ProductID] = append(storiesByProduct[story.ProductID],
			fmt.Sprintf("%d、作为%s，我希望%s", storyIndexByProduct[story.ProductID], role, gv))
	}

	out := make([]DemandSchedulingStoryDefault, 0, len(clarifyRows))
	for _, clarify := range clarifyRows {
		if clarify.ProductID == 0 {
			continue
		}
		productName := strings.TrimSpace(clarify.ProductName)
		if productName == "" {
			productName = fmt.Sprintf("系统%d", clarify.ProductID)
		}
		var spec string
		if lines := storiesByProduct[clarify.ProductID]; len(lines) > 0 {
			spec = strings.Join(lines, "\n") + "\n"
		}
		analyst := strings.TrimSpace(clarify.Analyst)
		plan := planByProduct[clarify.ProductID]
		out = append(out, DemandSchedulingStoryDefault{
			ProductID:      clarify.ProductID,
			ProductName:    productName,
			Title:          fmt.Sprintf("US%d-%s-%s", detail.ID, strings.TrimSpace(detail.Name), productName),
			Spec:           spec,
			AssignedTo:     analyst,
			AssignedToName: resolveRealname(analyst, realnameByAccount),
			PlanID:         plan.PlanID,
			PlanName:       strings.TrimSpace(plan.PlanName),
			Type:           "story",
			TypeLabel:      "功能",
			Pri:            normalizeStoryPriority(detail.Pri),
			Estimate:       estimateByProduct[clarify.ProductID],
		})
	}
	return out
}
