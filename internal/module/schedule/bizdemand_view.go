// =============================================================================
// 文件: internal/module/schedule/bizdemand_view.go
// 模块: 排期工作台
// 类型: action
// 职责: 将 Service 层业需列表 DTO 转换为页面展示用 BizRequirement 视图模型。
// 依赖: internal/module/schedule/form.go（Stage 常量）
//       internal/pkg/zentao
// =============================================================================

package schedule

import (
	"html/template"
	"strconv"
	"strings"

	"workbench/internal/pkg/zentao"
)

// toBizRequirementsView 将顶层业需列表转为页面树形一级行。
func toBizRequirementsView(items []BizDemandItem, zentaoBase string) []BizRequirement {
	out := make([]BizRequirement, 0, len(items))
	for _, item := range items {
		priority, priClass := formatPriority(item.Pri)
		agileGroup := item.TeamgroupName
		if agileGroup == "" {
			agileGroup = "—"
		}
		windowName := item.WindowName
		if windowName == "" {
			windowName = "—"
		}
		subReqs := toSubBizRequirementsView(item.Children, zentaoBase)
		devReqs := toDevRequirementsView(item.Stories, zentaoBase)
		actionLabel := "详情"
		if len(item.Children) == 0 {
			actionLabel = "去排期"
		}
		out = append(out, BizRequirement{
			DemandID:           item.ID,
			ID:                 formatDemandDisplayID(item.ID),
			Title:              item.Name,
			Priority:           priority,
			PriClass:           priClass,
			AgileGroup:         agileGroup,
			Stage:      item.Stage,
			StageClass: deriveBizStageClass(item.Stage),
			// 窗口阶段列已下线
			// WindowPhase:      item.WindowPhase,
			// WindowPhaseClass: deriveWindowPhaseClass(item.WindowPhase),
			WindowName: windowName,
			Owner:              formatOwner(item.OwnerName),
			ActionLabel:        actionLabel,
			ActionClass:        "primary",
			DetailURL:          template.URL(zentao.DemandViewURLWithBase(zentaoBase, item.ID)),
			HasChildren:        len(item.Children) > 0 || len(item.Stories) > 0,
			SubBizRequirements: subReqs,
			DevRequirements:    devReqs,
			// 未映射的 BizDemandItem 字段（本期零值/忽略）：Status、MainSystemName、ExtraSystemCount
		})
	}
	return out
}

// toSubBizRequirementsView 将子业需列表转为页面树形二级行。
func toSubBizRequirementsView(items []SubDemandItem, zentaoBase string) []SubBizRequirement {
	out := make([]SubBizRequirement, 0, len(items))
	for _, item := range items {
		priority, priClass := formatPriority(item.Pri)
		agileGroup := item.TeamgroupName
		if agileGroup == "" {
			agileGroup = "—"
		}
		windowName := item.WindowName
		if windowName == "" {
			windowName = "—"
		}
		out = append(out, SubBizRequirement{
			DemandID:         item.ID,
			ID:               formatDemandDisplayID(item.ID),
			Title:            item.Name,
			Priority:         priority,
			PriClass:         priClass,
			AgileGroup:       agileGroup,
			Stage:      item.Stage,
			StageClass: deriveBizStageClass(item.Stage),
			// 窗口阶段列已下线
			// WindowPhase:      item.WindowPhase,
			// WindowPhaseClass: deriveWindowPhaseClass(item.WindowPhase),
			WindowName: windowName,
			Owner:            formatOwner(item.OwnerName),
			ActionLabel:      "排期",
			ActionClass:      "primary",
			DetailURL:        template.URL(zentao.DemandViewURLWithBase(zentaoBase, item.ID)),
			DevRequirements:  toDevRequirementsView(item.Stories, zentaoBase),
			// 未映射的 SubDemandItem 字段（本期零值/忽略）：Status、MainSystemName、ExtraSystemCount、
			// TeamgroupName、Stage、WindowName
		})
	}
	return out
}

// toDevRequirementsView 将研发需求列表转为页面树形三级行。
func toDevRequirementsView(stories []StoryItem, zentaoBase string) []DevRequirement {
	out := make([]DevRequirement, 0, len(stories))
	for _, story := range stories {
		priority, priClass := formatPriority(story.Pri)
		owner := story.AssignedToName
		if owner == "" {
			owner = "待分配"
		}
		windowName := story.WindowName
		if windowName == "" {
			windowName = "—"
		}
		agileGroup := story.TeamgroupName
		if agileGroup == "" {
			agileGroup = "—"
		}
		out = append(out, DevRequirement{
			StoryID:     story.ID,
			ID:          formatOtherDisplayID(story.ID),
			Title:       story.Title,
			Priority:    priority,
			PriClass:    priClass,
			IsMain:      story.IsMainSystemAssociation == 1,
			Stage:       story.Stage,
			StageClass:  deriveBizStageClass(story.Stage),
			WindowName:  windowName,
			AgileGroup:  agileGroup,
			Owner:       owner,
			TaskCount:   story.TaskCount,
			ActionLabel: "维护任务",
			ActionClass: "primary",
			DetailURL:   template.URL(zentao.StoryViewURLWithBase(zentaoBase, story.ID)),
			// 未映射的 StoryItem 字段（本期零值/忽略）：ProductName、Stage、WindowName、
			// TeamgroupName、AssignedTo
		})
	}
	return out
}

func formatPriority(pri int) (label, class string) {
	if pri <= 0 {
		return "", ""
	}
	if pri > 4 {
		pri = 4
	}
	label = "P" + strconv.Itoa(pri)
	class = strings.ToLower(label)
	return label, class
}

func deriveBizStageClass(stage string) string {
	switch stage {
	case StageNoWindow, StageNoStory, StageNoTask:
		return "stage-tag--draft"
	case StageTaskUnassigned, StageTaskAssigned, IndependentStageTaskAssigned:
		return "stage-tag--final"
	default:
		return ""
	}
}

// 窗口阶段列已下线，暂保留以备恢复
// func deriveWindowPhaseClass(phase string) string {
// 	switch phase {
// 	case WindowPhaseInitial:
// 		return "schedule-window-phase-tag--initial"
// 	case WindowPhaseFinal:
// 		return "schedule-window-phase-tag--final"
// 	default:
// 		return ""
// 	}
// }

func formatOwner(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "待分配"
	}
	return name
}

// formatDemandDisplayID 业务需求展示编号：#US{id}。
// 顶层业需、以及从业需拆出的子需求都走这条；研需/子研不要用。
func formatDemandDisplayID(id uint) string {
	return "US" + strconv.FormatUint(uint64(id), 10)
}

// formatOtherDisplayID 非业务需求展示编号：#{id}（研需、子研等）。
func formatOtherDisplayID(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}

// toIndependentRequirementsView 将独立研发需求列表转为页面树形行。
func toIndependentRequirementsView(items []IndependentStoryItem, zentaoBase string) []IndependentRequirement {
	out := make([]IndependentRequirement, 0, len(items))
	for _, item := range items {
		priority, priClass := formatPriority(item.Pri)
		productName := item.ProductName
		if productName == "" {
			productName = "—"
		}
		windowName := item.WindowName
		if windowName == "" {
			windowName = "—"
		}
		teamgroupName := item.TeamgroupName
		if teamgroupName == "" {
			teamgroupName = "—"
		}
		children := toIndependentChildrenView(item.Children, zentaoBase)
		out = append(out, IndependentRequirement{
			StoryID:       item.ID,
			ID:            formatOtherDisplayID(item.ID),
			Title:         item.Title,
			Priority:      priority,
			PriClass:      priClass,
			ProductName:   productName,
			Stage:         item.Stage,
			StageClass:    deriveIndependentStageClass(item.Stage),
			WindowName:    windowName,
			TeamgroupName: teamgroupName,
			Owner:         formatOwner(item.AssignedToName),
			TaskCount:     item.TaskCount,
			HasChildren:   len(children) > 0,
			DetailURL:     template.URL(zentao.StoryViewURLWithBase(zentaoBase, item.ID)),
			Children:      children,
		})
	}
	return out
}

func toIndependentChildrenView(items []IndependentStoryItem, zentaoBase string) []IndependentChildRequirement {
	out := make([]IndependentChildRequirement, 0, len(items))
	for _, item := range items {
		priority, priClass := formatPriority(item.Pri)
		productName := item.ProductName
		if productName == "" {
			productName = "—"
		}
		windowName := item.WindowName
		if windowName == "" {
			windowName = "—"
		}
		teamgroupName := item.TeamgroupName
		if teamgroupName == "" {
			teamgroupName = "—"
		}
		detailURL := template.URL(zentao.StoryViewURLWithBase(zentaoBase, item.ID))
		out = append(out, IndependentChildRequirement{
			StoryID:       item.ID,
			ID:            formatOtherDisplayID(item.ID),
			Title:         item.Title,
			Priority:      priority,
			PriClass:      priClass,
			ProductName:   productName,
			Stage:         item.Stage,
			StageClass:    deriveIndependentStageClass(item.Stage),
			WindowName:    windowName,
			TeamgroupName: teamgroupName,
			Owner:         formatOwner(item.AssignedToName),
			TaskCount:     item.TaskCount,
			DetailURL:     detailURL,
		})
	}
	return out
}

func deriveIndependentStageClass(stage string) string {
	switch stage {
	case StageNoWindow, StageNoStory, StageNoTask:
		return "stage-tag--draft"
	case StageTaskUnassigned, StageTaskAssigned, IndependentStageTaskAssigned:
		return "stage-tag--final"
	default:
		return ""
	}
}
