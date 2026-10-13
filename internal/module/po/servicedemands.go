// =============================================================================
// 文件: internal/module/po/servicedemands.go
// 模块: PO 工作台
// 类型: action
// 职责: 价值流阶段需求列表装配，支持 objectType / priority / keyword / focus 筛选。
// 依赖: internal/model
// =============================================================================

package po

import (
	"context"
	"strings"

	"workbench/internal/model"
)

// listAllStageDemands 「全部」列表 = 其余各阶段列表按阶段顺序拼接，按 kind+id 去重（保留首次出现）。
// 返回全量 Items，由 Demands 统一切页。
func (s *Service) listAllStageDemands(ctx context.Context, actor *model.User, displayMap map[string]string, req DemandsReq) (*DemandsResp, error) {
	items := make([]WorkItemDetail, 0)
	seen := make(map[string]struct{})
	for _, def := range valueStreamStages {
		if def.status == "all" {
			continue
		}
		filter, ok := mysqlStageFilters[def.status]
		if !ok {
			continue
		}
		filter.suspended = req.Focus == FocusSuspended
		// 拉全量再并集；分页在 Demands 出口统一切
		resp, err := s.listMySQLDemands(ctx, actor, def.status, filter, displayMap, 0, 0, req)
		if err != nil {
			return nil, err
		}
		for _, item := range resp.Items {
			key := workItemKey(item.Kind, item.ID)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			items = append(items, item)
		}
	}
	return &DemandsResp{Items: items, Total: int64(len(items))}, nil
}

// listMySQLDemands 从 MySQL 加载指定价值流阶段列表。
// objectType=demand 只业需；story 只研需（无研需阶段返回空）；all 业需+研需合并。
// priority / keyword 在装配后过滤；有任一时禁用 SQL 分页，避免 Total 不准。
// pageSize<=0 表示不分页（供「全部」并集）；否则后端分页并填充 Total。
func (s *Service) listMySQLDemands(ctx context.Context, actor *model.User, stageStatus string, filter mysqlStageFilter, displayMap map[string]string, page, pageSize int, req DemandsReq) (*DemandsResp, error) {
	account := ""
	if actor != nil {
		account = actor.Account
	}
	includeDemand := req.ObjectType != "story"
	// 焦点仅业需口径（挂起/关注等）时不合并研需
	includeStory := req.ObjectType != "demand" && !req.focusDemandOnly()
	needMerge := includeStory && (filter.scheduleIncomplete || filter.deliverStories)
	unpaged := pageSize <= 0
	useSQLPage := !needMerge && !unpaged && req.Priority == "all" && req.Keyword == ""

	if !includeDemand {
		items, err := s.listStageStories(ctx, actor, account, stageStatus, filter, displayMap)
		if err != nil {
			return nil, err
		}
		return finalizeDemandsResp(items, page, pageSize, unpaged, req), nil
	}

	if useSQLPage {
		total, rows, err := s.findFocusDemands(ctx, account, filter, req, pageSize, (page-1)*pageSize)
		if err != nil {
			return nil, err
		}
		items, buildErr := s.buildDemandWorkItems(ctx, account, stageStatus, rows, displayMap)
		if buildErr != nil {
			return nil, buildErr
		}
		return &DemandsResp{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
	}

	_, rows, err := s.findFocusDemands(ctx, account, filter, req, 0, 0)
	if err != nil {
		return nil, err
	}
	items, buildErr := s.buildDemandWorkItems(ctx, account, stageStatus, rows, displayMap)
	if buildErr != nil {
		return nil, buildErr
	}
	if needMerge {
		storyItems, storyErr := s.collectStageStories(ctx, actor, account, stageStatus, filter, displayMap)
		if storyErr != nil {
			return nil, storyErr
		}
		items = append(items, storyItems...)
	}
	return finalizeDemandsResp(items, page, pageSize, unpaged, req), nil
}

// findFocusDemands 按焦点口径查业需；limit<=0 时不分页且 total 为 0（由调用方按 items 计）。
func (s *Service) findFocusDemands(ctx context.Context, account string, filter mysqlStageFilter, req DemandsReq, limit, offset int) (total int64, rows []DemandRow, err error) {
	switch req.Focus {
	case FocusMyRelated:
		if limit > 0 {
			total, err = s.repo.CountStarredDemands(ctx, account, filter)
			if err != nil {
				return 0, nil, err
			}
		}
		rows, err = s.repo.FindStarredDemands(ctx, account, filter, limit, offset)
	default:
		if limit > 0 {
			total, err = s.repo.CountRoleDemands(ctx, account, filter)
			if err != nil {
				return 0, nil, err
			}
		}
		rows, err = s.repo.FindRoleDemands(ctx, account, filter, limit, offset)
	}
	return total, rows, err
}

func (s *Service) listStageStories(ctx context.Context, actor *model.User, account, stageStatus string, filter mysqlStageFilter, displayMap map[string]string) ([]WorkItemDetail, error) {
	if !filter.scheduleIncomplete && !filter.deliverStories {
		return []WorkItemDetail{}, nil
	}
	return s.collectStageStories(ctx, actor, account, stageStatus, filter, displayMap)
}

func (s *Service) collectStageStories(ctx context.Context, actor *model.User, account, stageStatus string, filter mysqlStageFilter, displayMap map[string]string) ([]WorkItemDetail, error) {
	label := valueStreamLabelForStatus(stageStatus)
	items := make([]WorkItemDetail, 0)
	if filter.scheduleIncomplete {
		stories, storyErr := s.repo.FindScheduleStories(ctx, account)
		if storyErr != nil {
			return nil, storyErr
		}
		items = append(items, storyWorkItems(stories, label, actor, displayMap)...)
	}
	if filter.deliverStories {
		stories, storyErr := s.repo.FindDeliverStories(ctx, account)
		if storyErr != nil {
			return nil, storyErr
		}
		items = append(items, storyWorkItems(stories, label, actor, displayMap)...)
	}
	return items, nil
}

// finalizeDemandsResp 按优先级/关键词过滤后，按需切页。
func finalizeDemandsResp(items []WorkItemDetail, page, pageSize int, unpaged bool, req DemandsReq) *DemandsResp {
	items = filterWorkItemsByPriority(items, req.Priority)
	items = filterWorkItemsByKeyword(items, req.Keyword)
	if unpaged {
		return &DemandsResp{Items: items, Total: int64(len(items))}
	}
	pageItems, total := paginateWorkItems(items, page, pageSize)
	return &DemandsResp{Items: pageItems, Total: total, Page: page, PageSize: pageSize}
}

// filterWorkItemsByPriority p1/p2 精确匹配；p3 含 P3+P4；all 不过滤。
func filterWorkItemsByPriority(items []WorkItemDetail, priority string) []WorkItemDetail {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "", "all":
		return items
	case "p1", "p2", "p3":
	default:
		return items
	}
	out := make([]WorkItemDetail, 0, len(items))
	for _, item := range items {
		if matchWorkItemPriority(item.Pri, priority) {
			out = append(out, item)
		}
	}
	return out
}

func matchWorkItemPriority(pri, want string) bool {
	p := strings.ToUpper(strings.TrimSpace(pri))
	switch strings.ToLower(strings.TrimSpace(want)) {
	case "p1":
		return p == "P1" || p == "1"
	case "p2":
		return p == "P2" || p == "2"
	case "p3":
		return p == "P3" || p == "P4" || p == "3" || p == "4"
	default:
		return true
	}
}

// filterWorkItemsByKeyword 按编号/标题模糊匹配（不区分大小写）。
func filterWorkItemsByKeyword(items []WorkItemDetail, keyword string) []WorkItemDetail {
	kw := strings.ToLower(strings.TrimSpace(keyword))
	if kw == "" {
		return items
	}
	out := make([]WorkItemDetail, 0, len(items))
	for _, item := range items {
		id := strings.ToLower(strings.TrimSpace(item.ID))
		title := strings.ToLower(strings.TrimSpace(item.Title))
		if strings.Contains(id, kw) || strings.Contains(strings.TrimPrefix(id, "#"), kw) || strings.Contains(title, kw) {
			out = append(out, item)
		}
	}
	return out
}
