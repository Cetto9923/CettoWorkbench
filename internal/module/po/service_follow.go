// =============================================================================
// 文件: internal/module/po/service_follow.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的关注服务。V10.1 04 节: 只有 2 个对象视图（业务需求默认 / 项目报告），无"全部"。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"fmt"
	"strings"

	"workbench/internal/model"
	"workbench/internal/pkg/zentao"
)

// FollowList 我的关注列表服务。V10.1 04 节：只有 2 个对象视图（业务需求默认 / 项目报告）。
// 项目报告本期占位（数据源待后续接入）。
func (s *Service) FollowList(ctx context.Context, actor *model.User, req FollowListReq) (*FollowListResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return &FollowListResp{Items: []FollowItem{}, Page: req.Page, PageSize: req.PageSize}, nil
	}
	switch req.Tab {
	case FollowTabDemand:
		items, total, err := s.repo.FindFollowedDemands(ctx, actor.Account, req.Scope, req.Keyword, req.Page, req.PageSize)
		if err != nil {
			return nil, err
		}
		return &FollowListResp{Items: items, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
	case FollowTabProjectReport:
		// 项目报告 Tab 本期占位（依赖 zt_project.follow + zt_projectweekly，后续阶段接入）
		return &FollowListResp{Items: []FollowItem{}, Total: 0, Page: req.Page, PageSize: req.PageSize}, nil
	}
	return &FollowListResp{Items: []FollowItem{}, Page: req.Page, PageSize: req.PageSize}, nil
}

// FollowSetDemand 切换对业务需求的关注。
func (s *Service) FollowSetDemand(ctx context.Context, actor *model.User, demandID int64, followed bool) error {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil
	}
	return s.repo.SetDemandFollow(ctx, actor.Account, demandID, followed)
}

func (s *Service) loadAccountDisplayMap(ctx context.Context, actor *model.User) (map[string]string, error) {
	if s.userSvc == nil {
		return map[string]string{}, nil
	}
	return s.userSvc.AccountDisplayMap(ctx, actor)
}

// countAllStageUniq 各阶段只查 ID，按 kind+id 去重后返回业需/研需数量（与 listAllStageDemands 并集语义一致）。
func (s *Service) countAllStageUniq(ctx context.Context, account string) (demandSum, storySum int64, err error) {
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
		ids, idErr := s.repo.FindRoleDemandIDs(ctx, account, filter)
		if idErr != nil {
			return 0, 0, idErr
		}
		for _, id := range ids {
			seenDemand[id] = struct{}{}
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

// listAllStageDemands 「全部」列表 = 其余各阶段列表按阶段顺序拼接，按 kind+id 去重（保留首次出现）。
func (s *Service) listAllStageDemands(ctx context.Context, actor *model.User, displayMap map[string]string) (*DemandsResp, error) {
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
		resp, err := s.listMySQLDemands(ctx, actor, def.status, filter, displayMap)
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
	return &DemandsResp{Items: items}, nil
}

// listMySQLDemands 从 MySQL 加载指定价值流阶段的业需列表（排期/交付阶段额外合并独立研发需求）。
func (s *Service) listMySQLDemands(ctx context.Context, actor *model.User, stageStatus string, filter mysqlStageFilter, displayMap map[string]string) (*DemandsResp, error) {
	account := ""
	if actor != nil {
		account = actor.Account
	}
	rows, err := s.repo.FindRoleDemands(ctx, account, filter)
	if err != nil {
		return nil, err
	}
	label := valueStreamLabelForStatus(stageStatus)
	items := make([]WorkItemDetail, 0, len(rows))
	for _, row := range rows {
		pri := ""
		if row.Pri != "" {
			pri = "P" + row.Pri
		}
		ownerDisp := resolveNextOwnerDisplay(row, displayMap)
		items = append(items, WorkItemDetail{
			Kind:         "demand",
			ID:           fmt.Sprintf("US%d", row.ID),
			Pri:          pri,
			Title:        row.Name,
			Owner:        ownerDisp,
			NextOwner:    ownerDisp,
			ZentaoUrl:    zentao.URL("demand", "view", fmt.Sprintf("demandID=%d", row.ID)),
			ValueStream:  label,
			ZentaoStatus: row.Status,
		})
	}
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
	return &DemandsResp{Items: items}, nil
}

func resolveNextOwnerDisplay(row DemandRow, displayMap map[string]string) string {
	_, disp := DeriveCurrentHandler(row.Status, row.AssignedTo, row.QD, row.RD, row.BRA, row.PM,
		lookupAccountsDisplay(displayMap, row.PM),
		lookupAccountDisplay(displayMap, row.AssignedTo),
		lookupAccountDisplay(displayMap, row.QD),
		lookupAccountDisplay(displayMap, row.RD),
		lookupAccountDisplay(displayMap, row.BRA))
	return disp
}

func lookupAccountDisplay(displayMap map[string]string, account string) string {
	acc := strings.TrimSpace(account)
	if acc == "" {
		return ""
	}
	if displayMap != nil {
		if d := strings.TrimSpace(displayMap[acc]); d != "" {
			return d
		}
	}
	return acc
}

// lookupAccountsDisplay 支持 GROUP_CONCAT 多账号，逐个映射后用 ", " 拼接。
func lookupAccountsDisplay(displayMap map[string]string, accountsCSV string) string {
	raw := strings.TrimSpace(accountsCSV)
	if raw == "" {
		return ""
	}
	parts := strings.Split(raw, ",")
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		names = append(names, lookupAccountDisplay(displayMap, p))
	}
	return strings.Join(names, ", ")
}

func storyWorkItems(rows []StoryRow, label string, actor *model.User, displayMap map[string]string) []WorkItemDetail {
	account := ""
	if actor != nil {
		account = actor.Account
	}
	owner := lookupAccountDisplay(displayMap, account)
	if owner == "" && actor != nil {
		owner = FormatAccountName(actor.Account, actor.DisplayName)
	}
	items := make([]WorkItemDetail, 0, len(rows))
	for _, row := range rows {
		items = append(items, WorkItemDetail{
			Kind:         "story",
			ID:           fmt.Sprintf("U%d", row.ID),
			Pri:          fmt.Sprintf("P%d", row.Pri),
			Title:        row.Title,
			Owner:        owner,
			NextOwner:    owner,
			ZentaoUrl:    zentao.URL("story", "view", fmt.Sprintf("storyID=%d", row.ID)),
			ValueStream:  label,
			ZentaoStatus: row.Status,
		})
	}
	return items
}

func isValidValueStreamStatus(status string) bool {
	for _, def := range valueStreamStages {
		if def.status == status {
			return true
		}
	}
	return false
}

func valueStreamLabelForStatus(status string) string {
	for _, def := range valueStreamStages {
		if def.status == status {
			return def.label
		}
	}
	return ""
}

func workItemKey(kind, id string) string {
	return kind + ":" + id
}
