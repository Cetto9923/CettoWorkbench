// =============================================================================
// 文件: internal/module/agileteam/service_list.go
// 模块: 敏捷小组治理
// 类型: action
// 职责: 列表查询：父子成组分页、团队管理可见范围。
// 依赖: internal/model
// =============================================================================

package agileteam

import (
	"context"
	"sort"
	"strings"

	"workbench/internal/model"
)

type teamFamily struct {
	Parent   ListItem
	Children []ListItem
}

// List 敏捷小组列表（按父级+子级成组分页）。
func (s *Service) List(ctx context.Context, actor *model.User, req ListReq) (ListResp, error) {
	req.Normalize()
	rows, err := s.repo.ListTeamgroups(ctx)
	if err != nil {
		return ListResp{}, err
	}
	scopeOpts, rows, err := s.applyLeadScope(ctx, actor, req, rows)
	if err != nil {
		return ListResp{}, err
	}

	ids := make([]uint, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	membersByID, err := s.repo.ListMembersByGroupIDs(ctx, ids)
	if err != nil {
		return ListResp{}, err
	}
	pendingByID, err := s.repo.ListPendingByTeamgroupIDs(ctx, ids)
	if err != nil {
		return ListResp{}, err
	}
	adjIDs := make([]int64, 0, len(pendingByID))
	for _, p := range pendingByID {
		adjIDs = append(adjIDs, p.ID)
	}
	counts, err := s.repo.CountPendingItems(ctx, adjIDs)
	if err != nil {
		return ListResp{}, err
	}
	lastTimes, err := s.repo.LastAdjustTimes(ctx, ids)
	if err != nil {
		return ListResp{}, err
	}

	accounts := []string{}
	for _, r := range rows {
		accounts = append(accounts, r.PO, strings.TrimSpace(r.Manager))
	}
	names, _ := s.repo.ResolveRealnames(ctx, accounts)

	var all, enable, disable, pending int64
	allItems := make([]ListItem, 0, len(rows))
	matched := make([]ListItem, 0, len(rows))
	for _, r := range rows {
		st := strings.TrimSpace(r.Status)
		if st == "" || st == "enable" || st == "doing" {
			enable++
		} else {
			disable++
		}
		all++
		formal := membersByID[r.ID]
		pad, prem := 0, 0
		var pendID int64
		if p, ok := pendingByID[r.ID]; ok {
			pending++
			pendID = p.ID
			c := counts[p.ID]
			pad, prem = c.Add, c.Remove
		}
		coachAcc := firstAccount(r.Manager)
		poAcc := strings.TrimSpace(r.PO)
		lastAt := ""
		if t, ok := lastTimes[r.ID]; ok {
			lastAt = formatTime(t)
		}
		item := ListItem{
			ID: r.ID, Name: r.Name, ParentID: r.Parent, ParentName: r.ParentName,
			Type: teamTypeOf(r), CoachAccount: coachAcc, CoachName: names[coachAcc],
			POAccount: poAcc, POName: names[poAcc],
			FormalCount: len(formal), PendingAdd: pad, PendingRemove: prem,
			Status: st, StatusLabel: statusLabel(st),
			LastAdjustAt: lastAt, PendingAdjustID: pendID,
		}
		allItems = append(allItems, item)
		if matchListFilters(r, formal, pad > 0 || prem > 0, names, req) {
			matched = append(matched, item)
		}
	}

	byID := map[uint]ListItem{}
	for _, it := range allItems {
		byID[it.ID] = it
	}
	families := buildTeamFamilies(matched, byID)
	total := int64(0)
	for _, f := range families {
		total += int64(1 + len(f.Children))
	}
	pageItems, pageCount := pageFamilies(families, req.Page, req.PageSize)
	if pageItems == nil {
		pageItems = []ListItem{}
	}
	return ListResp{
		Items: pageItems, Total: total,
		AllCount: all, EnableCount: enable, DisableCount: disable, PendingCount: pending,
		Page: req.Page, PageSize: req.PageSize, PageCount: pageCount,
		ScopeOptions: scopeOpts, CanEdit: req.View != "lead",
	}, nil
}

func teamTypeOf(r TeamgroupRow) string {
	if strings.TrimSpace(r.Type) == "child" || r.Parent != 0 {
		return "child"
	}
	return "parent"
}

func matchListFilters(r TeamgroupRow, formal []TeamMemberRow, hasPending bool, names map[string]string, req ListReq) bool {
	if req.Name != "" && !strings.Contains(r.Name, req.Name) {
		return false
	}
	if req.ParentName != "" && !strings.Contains(r.ParentName, req.ParentName) {
		return false
	}
	st := strings.TrimSpace(r.Status)
	switch req.Status {
	case "enable":
		if !(st == "" || st == "enable" || st == "doing") {
			return false
		}
	case "disable":
		if st == "" || st == "enable" || st == "doing" {
			return false
		}
	}
	switch req.AdjustStatus {
	case "pending":
		if !hasPending {
			return false
		}
	case "none":
		if hasPending {
			return false
		}
	}
	if req.PO != "" {
		po := names[strings.TrimSpace(r.PO)]
		if !strings.Contains(r.PO, req.PO) && !strings.Contains(po, req.PO) {
			return false
		}
	}
	if req.Coach != "" {
		coach := firstAccount(r.Manager)
		cn := names[coach]
		if !strings.Contains(coach, req.Coach) && !strings.Contains(cn, req.Coach) {
			return false
		}
	}
	if req.Member != "" {
		hit := false
		for _, m := range formal {
			if strings.Contains(m.Account, req.Member) || strings.Contains(m.Name, req.Member) {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	return true
}

func buildTeamFamilies(matched []ListItem, all map[uint]ListItem) []teamFamily {
	if len(matched) == 0 {
		return nil
	}
	includeAllChildren := map[uint]bool{}
	parentOrder := []uint{}
	seenParent := map[uint]bool{}
	mark := func(pid uint) {
		if pid == 0 || seenParent[pid] {
			return
		}
		seenParent[pid] = true
		parentOrder = append(parentOrder, pid)
	}
	for _, it := range matched {
		if it.Type == "child" && it.ParentID != 0 {
			mark(it.ParentID)
			continue
		}
		mark(it.ID)
		includeAllChildren[it.ID] = true
	}
	sort.SliceStable(parentOrder, func(i, j int) bool { return parentOrder[i] < parentOrder[j] })

	childrenOf := map[uint][]ListItem{}
	for _, it := range all {
		if it.ParentID != 0 {
			childrenOf[it.ParentID] = append(childrenOf[it.ParentID], it)
		}
	}
	matchedChild := map[uint]bool{}
	for _, it := range matched {
		if it.Type == "child" && it.ParentID != 0 {
			matchedChild[it.ID] = true
		}
	}

	out := make([]teamFamily, 0, len(parentOrder))
	for _, pid := range parentOrder {
		parent, ok := all[pid]
		if !ok {
			continue
		}
		kids := childrenOf[pid]
		sort.SliceStable(kids, func(i, j int) bool { return kids[i].ID < kids[j].ID })
		if !includeAllChildren[pid] {
			filtered := make([]ListItem, 0, len(kids))
			for _, c := range kids {
				if matchedChild[c.ID] {
					filtered = append(filtered, c)
				}
			}
			kids = filtered
		}
		parent.Type = "parent"
		parent.ChildCount = len(kids)
		out = append(out, teamFamily{Parent: parent, Children: kids})
	}
	return out
}

func pageFamilies(families []teamFamily, page, pageSize int) ([]ListItem, int) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	if len(families) == 0 {
		return []ListItem{}, 1
	}
	pages := [][]ListItem{{}}
	cur := 0
	for _, f := range families {
		unit := flattenFamily(f)
		if len(pages[cur]) > 0 && len(pages[cur])+len(unit) > pageSize {
			pages = append(pages, []ListItem{})
			cur++
		}
		pages[cur] = append(pages[cur], unit...)
	}
	pageCount := len(pages)
	if page > pageCount {
		return []ListItem{}, pageCount
	}
	return pages[page-1], pageCount
}

func flattenFamily(f teamFamily) []ListItem {
	out := make([]ListItem, 0, 1+len(f.Children))
	out = append(out, f.Parent)
	out = append(out, f.Children...)
	return out
}

func (s *Service) applyLeadScope(ctx context.Context, actor *model.User, req ListReq, rows []TeamgroupRow) ([]ScopeOption, []TeamgroupRow, error) {
	if req.View != "lead" {
		return nil, rows, nil
	}
	acc := ""
	if actor != nil {
		acc = strings.TrimSpace(actor.Account)
	}
	parentIDs, err := s.repo.ListUserParentTeamIDs(ctx, acc)
	if err != nil {
		return nil, nil, err
	}
	if req.Scope == "dept" {
		isMgr, err := s.repo.IsDeptManager(ctx, acc)
		if err != nil {
			return nil, nil, err
		}
		if (actor != nil && actor.IsSuperAdmin) || isMgr {
			if actor != nil && actor.IsSuperAdmin {
				parentIDs = parentIDsOfRows(rows)
			} else {
				deptIDs, err := s.repo.ListDeptTreeIDs(ctx, acc)
				if err != nil {
					return nil, nil, err
				}
				parentIDs, err = s.repo.ListParentTeamIDsByDepts(ctx, deptIDs)
				if err != nil {
					return nil, nil, err
				}
			}
		}
	}
	if actor != nil && actor.IsSuperAdmin && len(parentIDs) == 0 {
		parentIDs = parentIDsOfRows(rows)
	}
	if req.ScopeID != 0 {
		if containsUint(parentIDs, req.ScopeID) {
			parentIDs = []uint{req.ScopeID}
		} else {
			parentIDs = []uint{}
		}
	}
	opts, err := s.repo.ListParentOptionsByIDs(ctx, parentIDs)
	if err != nil {
		return nil, nil, err
	}
	familyIDs, err := s.repo.ListFamilyIDs(ctx, parentIDs)
	if err != nil {
		return nil, nil, err
	}
	allow := map[uint]bool{}
	for _, id := range familyIDs {
		allow[id] = true
	}
	filtered := make([]TeamgroupRow, 0, len(rows))
	for _, r := range rows {
		if allow[r.ID] {
			filtered = append(filtered, r)
		}
	}
	return opts, filtered, nil
}

func parentIDsOfRows(rows []TeamgroupRow) []uint {
	seen := map[uint]bool{}
	var ids []uint
	for _, r := range rows {
		id := r.Parent
		if id == 0 {
			id = r.ID
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

func containsUint(ids []uint, want uint) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}
