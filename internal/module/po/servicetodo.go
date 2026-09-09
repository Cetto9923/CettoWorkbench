// =============================================================================
// 文件: internal/module/po/servicetodo.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的待办服务（V10.1 02 节 7 维 AND）。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"sort"
	"strings"
	"time"

	"workbench/internal/model"
)

// TodoList 我的待办列表服务。
// V10.1 02 节：7 维 AND 公式。本期完整实现：Tab + 办理场景 + 阶段 + 对象 + 我的关系 + 办理责任 + 关键词。
// 仅已接入统一查询的数据域可通过页面进入，需求治理 / 全部 Tab 走 actor scope 聚合。
func (s *Service) TodoList(ctx context.Context, actor *model.User, req TodoListReq) (*TodoListResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return &TodoListResp{Items: []TodoItem{}, Page: req.Page, PageSize: req.PageSize, Facets: buildTodoFacets(nil)}, nil
	}
	pagedResult, err := s.repo.QueryTodoUnified(ctx, actor.Account, req)
	if err != nil {
		return nil, err
	}
	return &TodoListResp{
		Items:    pagedResult.Items,
		Total:    pagedResult.Total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Summary:  pagedResult.Summary,
		Groups:   pagedResult.Groups,
		Facets:   pagedResult.Facets,
	}, nil
}

func filterTodoKeyword(items []TodoItem, keyword string) []TodoItem {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return items
	}
	out := make([]TodoItem, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.DisplayID), keyword) ||
			strings.Contains(strings.ToLower(item.Title), keyword) ||
			strings.Contains(strings.ToLower(item.Owner), keyword) {
			out = append(out, item)
		}
	}
	return out
}

func filterTodoDimensions(items []TodoItem, req TodoListReq) []TodoItem {
	out := make([]TodoItem, 0, len(items))
	for _, item := range items {
		if req.Relation != RelationAll && req.Relation != "" &&
			!((req.Relation == RelationInCharge && item.Relation == "我负责") ||
				(req.Relation == RelationCooperate && item.Relation == "我配合") ||
				(req.Relation == RelationFollow && item.Relation == "我关注")) {
			continue
		}
		if req.Responsibility != ResponsibilityAll && req.Responsibility != "" &&
			!((req.Responsibility == ResponsibilityMyAction && item.Responsibility == "待我处理") ||
				(req.Responsibility == ResponsibilityMyFollowUp && item.Responsibility == "待我跟进")) {
			continue
		}
		// 阶段与办理场景当前仅有业务需求的正式口径；选择后不混入无法判定的任务和 Bug。
		if req.Stage != "" && req.Stage != "all" && item.Kind != "demand" {
			continue
		}
		if req.Action != "" && req.Action != TodoActionAll && item.Kind != "demand" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func sortTodoItems(items []TodoItem) {
	priorityRank := func(priority string) int {
		switch priority {
		case "P1":
			return 1
		case "P2":
			return 2
		case "P3":
			return 3
		default:
			return 4
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		leftRank, rightRank := priorityRank(items[i].Priority), priorityRank(items[j].Priority)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		leftDeadline, rightDeadline := items[i].Deadline, items[j].Deadline
		if leftDeadline == "" {
			leftDeadline = "9999-12-31"
		}
		if rightDeadline == "" {
			rightDeadline = "9999-12-31"
		}
		if leftDeadline != rightDeadline {
			return leftDeadline < rightDeadline
		}
		return items[i].ID > items[j].ID
	})
}

func summarizeTodoItems(items []TodoItem) TodoSummary {
	today := time.Now().Format("2006-01-02")
	result := TodoSummary{Pending: len(items)}
	for _, item := range items {
		if item.Deadline == today {
			result.Today++
		}
		if item.Deadline != "" && item.Deadline < today {
			result.Overdue++
		}
		if item.Blocked {
			result.Blocked++
		}
		if item.Priority == "P1" {
			result.P1++
		}
	}
	return result
}

func countTodoGroups(items []TodoItem) TodoGroupCounts {
	counts := TodoGroupCounts{All: len(items)}
	for _, item := range items {
		switch item.Kind {
		case "demand", "story":
			counts.Demand++
		case "task":
			counts.Execution++
		case "bug", "testtask":
			counts.Testing++
		case "issue", "risk":
			counts.Risk++
		case "todo":
			counts.Personal++
		case "approval":
			counts.Approval++
		}
	}
	return counts
}

func filterTodoGroup(items []TodoItem, tab TodoTab) []TodoItem {
	if tab == TodoTabAll || tab == "" {
		return items
	}
	out := make([]TodoItem, 0, len(items))
	for _, item := range items {
		matched := (tab == TodoTabDemand && (item.Kind == "demand" || item.Kind == "story")) ||
			(tab == TodoTabExecution && item.Kind == "task") ||
			(tab == TodoTabTesting && (item.Kind == "bug" || item.Kind == "testtask")) ||
			(tab == TodoTabRisk && (item.Kind == "issue" || item.Kind == "risk")) ||
			(tab == TodoTabPersonal && item.Kind == "todo") ||
			(tab == TodoTabApproval && item.Kind == "approval")
		if matched {
			out = append(out, item)
		}
	}
	return out
}

func filterTodoFocus(items []TodoItem, focus string, now time.Time) []TodoItem {
	if focus == "" || focus == "pending" {
		return items
	}
	today := now.Format("2006-01-02")
	out := make([]TodoItem, 0, len(items))
	for _, item := range items {
		matched := (focus == "today" && item.Deadline == today) ||
			(focus == "overdue" && item.Deadline != "" && item.Deadline < today) ||
			(focus == "blocked" && item.Blocked) ||
			(focus == "p1" && item.Priority == "P1")
		if matched {
			out = append(out, item)
		}
	}
	return out
}
