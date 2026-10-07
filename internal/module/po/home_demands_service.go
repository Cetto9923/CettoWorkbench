// =============================================================================
// 文件: internal/module/po/home_demands_service.go
// 模块: PO 工作台
// 类型: service
// 职责: 首页与需求列表 (/demands) 数据组装、逾期计算、主行动聚合与人员展示。
// =============================================================================

package po

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"workbench/internal/model"
	"workbench/internal/module/po/primaryaction"
	"workbench/internal/pkg/zentao"
)

func (s *Service) loadAccountDisplayMap(ctx context.Context, actor *model.User) (map[string]string, error) {
	if s.userSvc == nil {
		return map[string]string{}, nil
	}
	return s.userSvc.AccountDisplayMap(ctx, actor)
}

// listMySQLDemands 从 MySQL 加载指定价值流阶段的业需列表（排期/交付阶段额外合并独立研发需求），并分页返回。
func (s *Service) listMySQLDemands(ctx context.Context, actor *model.User, stageStatus string, filter mysqlStageFilter, req DemandsReq, displayMap map[string]string) (*DemandsResp, error) {
	account := ""
	if actor != nil {
		account = actor.Account
	}
	includeDemand := req.ObjectType != "story"
	label := valueStreamLabelForStatus(stageStatus)
	offset := (req.Page - 1) * req.PageSize

	// 受理阶段（accept）：SQL 中待我评审置顶，再分页。
	// 受理阶段 status 跨 draft/wait/refuse，canReview 仅 wait 子集，
	// 必须在分页前排好序，否则待我评审的需求可能散落在各页。
	if stageStatus == "accept" && account != "" {
		if !includeDemand {
			return &DemandsResp{Items: []WorkItemDetail{}, Total: 0, Page: req.Page, PageSize: req.PageSize}, nil
		}
		refs, total, err := s.repo.acceptRefsPaged(ctx, account, req)
		if err != nil {
			return nil, err
		}
		return s.populateWorkItems(ctx, actor, refs, total, req.Page, req.PageSize, displayMap, req)
	}

	// 纯业需阶段：直接单 SQL Count + 单 SQL 分页查
	if !filter.scheduleIncomplete && !filter.deliverStories {
		if !includeDemand {
			return &DemandsResp{Items: []WorkItemDetail{}, Total: 0, Page: req.Page, PageSize: req.PageSize}, nil
		}
		total, err := s.repo.CountRoleDemandsWithFilters(ctx, account, filter, req)
		if err != nil {
			return nil, err
		}
		if total == 0 || int64(offset) >= total {
			return &DemandsResp{Items: []WorkItemDetail{}, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
		}
		rows, err := s.repo.FindRoleDemandsPagedWithFilters(ctx, account, filter, req, offset, req.PageSize)
		if err != nil {
			return nil, err
		}
		items := make([]WorkItemDetail, 0, len(rows))
		demandMap := make(map[int]DemandRow, len(rows))
		demandIDs := make([]uint, 0, len(rows))
		for _, row := range rows {
			items = append(items, buildDemandWorkItem(row, label, displayMap))
			demandMap[row.ID] = row
			demandIDs = append(demandIDs, uint(row.ID))
		}
		actions, actionErr := s.DeriveDemandPrimaryActions(ctx, actor, demandIDs)
		if actionErr != nil {
			return nil, actionErr
		}
		for i := range items {
			id := parseDemandNumericID(items[i].ID)
			if pa, ok := actions[uint(id)]; ok {
				paCopy := pa
				items[i].PrimaryAction = &paCopy
			}
		}
		if err := s.enrichDemandCanReview(ctx, account, items, demandMap); err != nil {
			return nil, err
		}
		return &DemandsResp{Items: items, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
	}

	// 包含独立研发需求阶段（排期/交付）：SQL 合并业需/研需 ID 后分页，再按需加载详情
	refs, total, err := s.repo.FindStageMixedRefsPaged(ctx, account, stageStatus, filter, req)
	if err != nil {
		return nil, err
	}
	if total == 0 || len(refs) == 0 {
		return &DemandsResp{Items: []WorkItemDetail{}, Total: int64(total), Page: req.Page, PageSize: req.PageSize}, nil
	}
	return s.populateWorkItems(ctx, actor, refs, total, req.Page, req.PageSize, displayMap, req)
}

func (s *Service) populateWorkItems(ctx context.Context, actor *model.User, pageRefs []itemRef, total, page, pageSize int, displayMap map[string]string, req DemandsReq) (*DemandsResp, error) {
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

	// 批量派生主操作，避免行内 N+1。
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
	participateStoryIDs, err := s.prepareParticipateStoryActions(ctx, actor, pageRefs, req, storyActions)
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
				item.PrimaryAction = replaceParticipateScheduleAction(item.PrimaryAction, participateStoryIDs[ref.id], storyActions)
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
	if req.Focus != "" {
		if err := s.enrichHomeDeadlines(ctx, items, demandMap, storyMap); err != nil {
			return nil, err
		}
	}
	// 与 SQL 分页前排序一致：评审 > 提交 > 查看，同优先级超期在前。
	sort.SliceStable(items, func(i, j int) bool {
		if homeActionPriority(items[i]) != homeActionPriority(items[j]) {
			return homeActionPriority(items[i]) < homeActionPriority(items[j])
		}
		return items[i].Overdue && !items[j].Overdue
	})

	return &DemandsResp{
		Items:    items,
		Total:    int64(total),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func homeActionPriority(item WorkItemDetail) int {
	if item.CanReview || (item.PrimaryAction != nil && item.PrimaryAction.Key == string(primaryaction.KeyApprove)) {
		return 0
	}
	if item.PrimaryAction != nil && item.PrimaryAction.Key == string(primaryaction.KeySubmitReview) {
		return 1
	}
	return 2
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
	deadline, isOverdue, days := calcOverdue(row.Deadline, row.Status)
	return WorkItemDetail{
		Kind:           "demand",
		ID:             fmt.Sprintf("US%d", row.ID),
		Pri:            pri,
		Title:          row.Name,
		AssignedToName: lookupAccountDisplay(displayMap, row.AssignedTo),
		Owner:          ownerDisp,
		NextOwner:      ownerDisp,
		ZentaoUrl:      zentao.URL("demand", "view", fmt.Sprintf("demandID=%d", row.ID)),
		ZentaoEditUrl:  zentao.DemandEditURL(uint(row.ID)),
		ValueStream:    label,
		ZentaoStatus:   row.Status,
		Suspended:      strings.EqualFold(strings.TrimSpace(row.Hang), "1") || strings.EqualFold(strings.TrimSpace(row.Hang), "true"),
		Blocked:        strings.EqualFold(strings.TrimSpace(row.Status), "refuse"),
		Deadline:       deadline,
		Overdue:        isOverdue,
		OverdueDays:    days,
	}
}

func resolveNextOwnerDisplay(row DemandRow, displayMap map[string]string) string {
	_, disp := DeriveCurrentHandler(row.Status, row.AssignedTo, row.QD, row.RD, row.BRA, row.PM,
		lookupAccountsDisplay(displayMap, row.PM), lookupAccountDisplay(displayMap, row.AssignedTo),
		lookupAccountDisplay(displayMap, row.QD), lookupAccountDisplay(displayMap, row.RD), lookupAccountDisplay(displayMap, row.BRA))
	return disp
}

func lookupAccountDisplay(displayMap map[string]string, account string) string {
	acc := strings.TrimSpace(account)
	if acc == "" || displayMap == nil {
		return acc
	}
	if d := strings.TrimSpace(displayMap[acc]); d != "" {
		return d
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
		if p = strings.TrimSpace(p); p != "" {
			names = append(names, lookupAccountDisplay(displayMap, p))
		}
	}
	return strings.Join(names, ", ")
}

func buildStoryWorkItem(row StoryRow, label string, actor *model.User, displayMap map[string]string) WorkItemDetail {
	owner := lookupAccountDisplay(displayMap, row.AssignedTo)
	deadline, isOverdue, days := "", false, 0
	return WorkItemDetail{
		Kind:           "story",
		ID:             fmt.Sprintf("%d", row.ID),
		Pri:            fmt.Sprintf("P%d", row.Pri),
		Title:          row.Title,
		AssignedToName: owner,
		Owner:          owner,
		NextOwner:      owner,
		ZentaoUrl:      zentao.URL("story", "view", fmt.Sprintf("storyID=%d", row.ID)),
		ValueStream:    label,
		ZentaoStatus:   row.Status,
		Deadline:       deadline,
		Overdue:        isOverdue,
		OverdueDays:    days,
	}
}

func calcOverdue(dateStr, status string) (string, bool, int) {
	d := strings.TrimSpace(dateStr)
	if isTerminal(status) {
		return d, false, 0
	}
	if isUnsetDateText(d) {
		return "", false, 0
	}
	if len(d) > 10 {
		d = d[:10]
	}
	// 按本地自然日比较：time.Parse 默认落 UTC，会在 0~8 点窗口把「今天到期」算成逾期一天。
	t, err := time.ParseInLocation("2006-01-02", d, time.Local)
	if err != nil || t.Year() < 2000 {
		return "", false, 0
	}
	days := dayDiff(t, time.Now())
	if days <= 0 {
		return d, false, 0
	}
	return d, true, days
}

// isUnsetDateText 判断禅道日期文本是否等价于“未填”：空串、零日期（0000-00-00），驱动解析成的 0001-01-01，或早于 2000-01-01 的脏数据。
func isUnsetDateText(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, "0000-00-00") || strings.HasPrefix(s, "0001-01-01") {
		return true
	}
	if len(s) >= 10 {
		s = s[:10]
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		if t.Year() < 2000 {
			return true
		}
	}
	return false
}

func isValidValueStreamStatus(status string) bool {
	return valueStreamLabelForStatus(status) != ""
}

func valueStreamLabelForStatus(status string) string {
	for _, def := range valueStreamStages {
		if def.status == status {
			return def.label
		}
	}
	return ""
}
