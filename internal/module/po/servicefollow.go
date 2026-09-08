// =============================================================================
// 文件: internal/module/po/servicefollow.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的关注服务。V10.1 04 节: 只有 2 个对象视图（业务需求默认 / 项目报告），无"全部"。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"fmt"
	"strconv"
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
		items, total, err := s.repo.FindFollowedDemands(ctx, RepoFindFollowedDemandsReq{
			Account: actor.Account, Scope: req.Scope, Keyword: req.Keyword, Page: req.Page, PageSize: req.PageSize,
		})
		if err != nil {
			return nil, err
		}
		if err := s.attachFollowPrimaryActions(ctx, actor, items); err != nil {
			return nil, err
		}
		return &FollowListResp{Items: items, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
	case FollowTabProjectReport:
		items, total, err := s.repo.FindFollowedProjectReports(ctx, RepoFindFollowedProjectReportsReq{
			Account: actor.Account, Keyword: req.Keyword, Page: req.Page, PageSize: req.PageSize,
		})
		if err != nil {
			return nil, err
		}
		return &FollowListResp{Items: items, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
	}
	return &FollowListResp{Items: []FollowItem{}, Page: req.Page, PageSize: req.PageSize}, nil
}

// attachFollowPrimaryActions 为关注列表业需批量挂主操作（Stage 5 单一真源）。
func (s *Service) attachFollowPrimaryActions(ctx context.Context, actor *model.User, items []FollowItem) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(items))
	for _, it := range items {
		if it.ID > 0 {
			ids = append(ids, uint(it.ID))
		}
	}
	actions, err := s.DeriveDemandPrimaryActions(ctx, actor, ids)
	if err != nil {
		return err
	}
	for i := range items {
		if pa, ok := actions[uint(items[i].ID)]; ok {
			paCopy := pa
			items[i].PrimaryAction = &paCopy
		}
	}
	return nil
}

// FollowSetDemand 切换对业务需求的关注。
func (s *Service) FollowSetDemand(ctx context.Context, actor *model.User, req FollowSetReq) error {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil
	}
	return s.repo.SaveDemandFollow(ctx, RepoSaveDemandFollowReq{
		Account: actor.Account, DemandID: req.ID, Followed: *req.Followed,
	})
}

// FollowRemoveProjectReport 解除当前用户对项目周报的关注。
func (s *Service) FollowRemoveProjectReport(ctx context.Context, actor *model.User, projectID int64) error {
	if actor == nil || strings.TrimSpace(actor.Account) == "" || projectID <= 0 {
		return nil
	}
	return s.repo.RemoveProjectReportFollow(ctx, RepoRemoveProjectReportFollowReq{Account: actor.Account, ProjectID: projectID})
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

type itemRef struct {
	kind        string
	id          int
	stageStatus string
}

// listAllStageDemands 「全部」列表 = 其余各阶段列表按阶段顺序拼接，按 kind+id 去重（保留首次出现），并分页返回。
func (s *Service) listAllStageDemands(ctx context.Context, actor *model.User, req DemandsReq, displayMap map[string]string) (*DemandsResp, error) {
	account := ""
	if actor != nil {
		account = actor.Account
	}
	allRefs := make([]itemRef, 0)
	seen := make(map[string]struct{})
	for _, def := range valueStreamStages {
		if def.status == "all" {
			continue
		}
		filter, ok := mysqlStageFilters[def.status]
		if !ok {
			continue
		}
		demandIDs, err := s.repo.FindRoleDemandIDs(ctx, account, filter)
		if err != nil {
			return nil, err
		}
		for _, id := range demandIDs {
			key := fmt.Sprintf("demand:%d", id)
			if _, exists := seen[key]; !exists {
				seen[key] = struct{}{}
				allRefs = append(allRefs, itemRef{kind: "demand", id: id, stageStatus: def.status})
			}
		}
		if filter.scheduleIncomplete {
			storyIDs, sErr := s.repo.FindScheduleStoryIDs(ctx, account)
			if sErr != nil {
				return nil, sErr
			}
			for _, id := range storyIDs {
				key := fmt.Sprintf("story:%d", id)
				if _, exists := seen[key]; !exists {
					seen[key] = struct{}{}
					allRefs = append(allRefs, itemRef{kind: "story", id: id, stageStatus: def.status})
				}
			}
		}
		if filter.deliverStories {
			storyIDs, sErr := s.repo.FindDeliverStoryIDs(ctx, account)
			if sErr != nil {
				return nil, sErr
			}
			for _, id := range storyIDs {
				key := fmt.Sprintf("story:%d", id)
				if _, exists := seen[key]; !exists {
					seen[key] = struct{}{}
					allRefs = append(allRefs, itemRef{kind: "story", id: id, stageStatus: def.status})
				}
			}
		}
	}

	total := len(allRefs)
	offset := (req.Page - 1) * req.PageSize
	if offset >= total || total == 0 {
		return &DemandsResp{Items: []WorkItemDetail{}, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
	}
	end := offset + req.PageSize
	if end > total {
		end = total
	}
	pageRefs := allRefs[offset:end]

	return s.populateWorkItems(ctx, actor, pageRefs, total, req.Page, req.PageSize, displayMap)
}

// listMySQLDemands 从 MySQL 加载指定价值流阶段的业需列表（排期/交付阶段额外合并独立研发需求），并分页返回。
func (s *Service) listMySQLDemands(ctx context.Context, actor *model.User, stageStatus string, filter mysqlStageFilter, req DemandsReq, displayMap map[string]string) (*DemandsResp, error) {
	account := ""
	if actor != nil {
		account = actor.Account
	}
	label := valueStreamLabelForStatus(stageStatus)
	offset := (req.Page - 1) * req.PageSize

	// 纯业需阶段：直接单 SQL Count + 单 SQL 分页查
	if !filter.scheduleIncomplete && !filter.deliverStories {
		total, err := s.repo.CountRoleDemands(ctx, account, filter)
		if err != nil {
			return nil, err
		}
		if total == 0 || int64(offset) >= total {
			return &DemandsResp{Items: []WorkItemDetail{}, Total: int(total), Page: req.Page, PageSize: req.PageSize}, nil
		}
		rows, err := s.repo.FindRoleDemandsPaged(ctx, account, filter, offset, req.PageSize)
		if err != nil {
			return nil, err
		}
		items := make([]WorkItemDetail, 0, len(rows))
		demandMap := make(map[int]DemandRow, len(rows))
		for _, row := range rows {
			items = append(items, buildDemandWorkItem(row, label, displayMap))
			demandMap[row.ID] = row
		}
		if err := s.enrichDemandCanReview(ctx, account, items, demandMap); err != nil {
			return nil, err
		}
		return &DemandsResp{Items: items, Total: int(total), Page: req.Page, PageSize: req.PageSize}, nil
	}

	// 包含独立研发需求阶段（排期/交付）：按 ID 投影分页后按需加载详情
	demandIDs, err := s.repo.FindRoleDemandIDs(ctx, account, filter)
	if err != nil {
		return nil, err
	}
	refs := make([]itemRef, 0, len(demandIDs))
	for _, id := range demandIDs {
		refs = append(refs, itemRef{kind: "demand", id: id, stageStatus: stageStatus})
	}
	if filter.scheduleIncomplete {
		storyIDs, sErr := s.repo.FindScheduleStoryIDs(ctx, account)
		if sErr != nil {
			return nil, sErr
		}
		for _, id := range storyIDs {
			refs = append(refs, itemRef{kind: "story", id: id, stageStatus: stageStatus})
		}
	}
	if filter.deliverStories {
		storyIDs, sErr := s.repo.FindDeliverStoryIDs(ctx, account)
		if sErr != nil {
			return nil, sErr
		}
		for _, id := range storyIDs {
			refs = append(refs, itemRef{kind: "story", id: id, stageStatus: stageStatus})
		}
	}

	total := len(refs)
	if offset >= total || total == 0 {
		return &DemandsResp{Items: []WorkItemDetail{}, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
	}
	end := offset + req.PageSize
	if end > total {
		end = total
	}
	pageRefs := refs[offset:end]

	return s.populateWorkItems(ctx, actor, pageRefs, total, req.Page, req.PageSize, displayMap)
}

func (s *Service) populateWorkItems(ctx context.Context, actor *model.User, pageRefs []itemRef, total, page, pageSize int, displayMap map[string]string) (*DemandsResp, error) {
	var demandIDs []int
	var storyIDs []int
	for _, ref := range pageRefs {
		if ref.kind == "demand" {
			demandIDs = append(demandIDs, ref.id)
		} else if ref.kind == "story" {
			storyIDs = append(storyIDs, ref.id)
		}
	}

	demandMap := make(map[int]DemandRow, len(demandIDs))
	if len(demandIDs) > 0 {
		dRows, err := s.repo.FindRoleDemandsByIDs(ctx, demandIDs)
		if err != nil {
			return nil, err
		}
		for _, r := range dRows {
			demandMap[r.ID] = r
		}
	}

	storyMap := make(map[int]StoryRow, len(storyIDs))
	if len(storyIDs) > 0 {
		sRows, err := s.repo.FindStoriesByIDs(ctx, storyIDs)
		if err != nil {
			return nil, err
		}
		for _, r := range sRows {
			storyMap[r.ID] = r
		}
	}

	// Stage 5：批量派生主操作（避免行内 N+1）。
	demandIDsUint := toUintSlice(demandIDs)
	storyIDsUint := toUintSlice(storyIDs)
	demandActions, err := s.DeriveDemandPrimaryActions(ctx, actor, demandIDsUint)
	if err != nil {
		return nil, err
	}
	storyActions, err := s.DeriveStoryPrimaryActions(ctx, actor, storyIDsUint, false)
	if err != nil {
		return nil, err
	}

	items := make([]WorkItemDetail, 0, len(pageRefs))
	for _, ref := range pageRefs {
		label := valueStreamLabelForStatus(ref.stageStatus)
		if ref.kind == "demand" {
			if row, ok := demandMap[ref.id]; ok {
				item := buildDemandWorkItem(row, label, displayMap)
				if pa, ok := demandActions[uint(ref.id)]; ok {
					paCopy := pa
					item.PrimaryAction = &paCopy
				}
				items = append(items, item)
			}
		} else if ref.kind == "story" {
			if row, ok := storyMap[ref.id]; ok {
				item := buildStoryWorkItem(row, label, actor, displayMap)
				if pa, ok := storyActions[uint(ref.id)]; ok {
					paCopy := pa
					item.PrimaryAction = &paCopy
				}
				items = append(items, item)
			}
		}
	}

	account := ""
	if actor != nil {
		account = actor.Account
	}
	if err := s.enrichDemandCanReview(ctx, account, items, demandMap); err != nil {
		return nil, err
	}

	return &DemandsResp{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// toUintSlice 把 []int 安全转为 []uint（ref.id 已经是正数）。
func toUintSlice(in []int) []uint {
	if len(in) == 0 {
		return nil
	}
	out := make([]uint, 0, len(in))
	for _, v := range in {
		if v > 0 {
			out = append(out, uint(v))
		}
	}
	return out
}


// enrichDemandCanReview 为当前页业需批量标记 canReview（仅 status=wait 才查 zt_demandreview）。
func (s *Service) enrichDemandCanReview(ctx context.Context, account string, items []WorkItemDetail, demandRows map[int]DemandRow) error {
	if s == nil || s.repo == nil || strings.TrimSpace(account) == "" || len(items) == 0 {
		return nil
	}
	waitIDs := make([]int, 0, len(items))
	for i := range items {
		if items[i].Kind != "demand" {
			continue
		}
		// 优先用已加载行的 status；无 map 时退化为仅对 wait 展示字段已有值的项查询。
		id := parseDemandNumericID(items[i].ID)
		if id <= 0 {
			continue
		}
		if demandRows != nil {
			if row, ok := demandRows[id]; ok {
				if strings.TrimSpace(row.Status) != "wait" {
					continue
				}
			} else {
				continue
			}
		} else if strings.TrimSpace(items[i].ZentaoStatus) != "wait" {
			continue
		}
		waitIDs = append(waitIDs, id)
	}
	if len(waitIDs) == 0 {
		return nil
	}
	pending, err := s.repo.FindPendingReviewDemandIDs(ctx, account, waitIDs)
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].Kind != "demand" {
			continue
		}
		id := parseDemandNumericID(items[i].ID)
		if id <= 0 {
			continue
		}
		_, items[i].CanReview = pending[id]
	}
	return nil
}

// parseDemandNumericID 解析列表展示号 US{id} 或纯数字主键。
func parseDemandNumericID(raw string) int {
	raw = strings.TrimSpace(raw)
	if len(raw) >= 2 && strings.EqualFold(raw[:2], "US") {
		raw = raw[2:]
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func buildDemandWorkItem(row DemandRow, label string, displayMap map[string]string) WorkItemDetail {
	pri := ""
	if row.Pri != "" {
		pri = "P" + row.Pri
	}
	ownerDisp := resolveNextOwnerDisplay(row, displayMap)
	return WorkItemDetail{
		Kind:         "demand",
		ID:           fmt.Sprintf("US%d", row.ID),
		Pri:          pri,
		Title:        row.Name,
		Owner:        ownerDisp,
		NextOwner:    ownerDisp,
		ZentaoUrl:    zentao.URL("demand", "view", fmt.Sprintf("demandID=%d", row.ID)),
		ValueStream:  label,
		ZentaoStatus: row.Status,
		Suspended:    strings.EqualFold(strings.TrimSpace(row.Hang), "1") || strings.EqualFold(strings.TrimSpace(row.Hang), "true"),
		Blocked:      strings.EqualFold(strings.TrimSpace(row.Status), "refuse"),
	}
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

func buildStoryWorkItem(row StoryRow, label string, actor *model.User, displayMap map[string]string) WorkItemDetail {
	account := ""
	if actor != nil {
		account = actor.Account
	}
	owner := lookupAccountDisplay(displayMap, account)
	if owner == "" && actor != nil {
		owner = FormatAccountName(actor.Account, actor.DisplayName)
	}
	return WorkItemDetail{
		Kind:         "story",
		ID:           fmt.Sprintf("%d", row.ID),
		Pri:          fmt.Sprintf("P%d", row.Pri),
		Title:        row.Title,
		Owner:        owner,
		NextOwner:    owner,
		ZentaoUrl:    zentao.URL("story", "view", fmt.Sprintf("storyID=%d", row.ID)),
		ValueStream:  label,
		ZentaoStatus: row.Status,
	}
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
