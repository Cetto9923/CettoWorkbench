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
	"workbench/internal/pkg/errorx"
)

type teamFamily struct {
	Parent   ListItem
	Children []ListItem
}

// List 敏捷小组列表（按父级+子级成组分页）。
func (s *Service) List(ctx context.Context, actor *model.User, req ListReq) (ListResp, error) {
	requestedScope := strings.ToLower(strings.TrimSpace(req.Scope))
	req.Normalize()
	var availableScopes []string
	if req.View == "lead" {
		var err error
		availableScopes, err = s.availableLeadScopes(ctx, actor)
		if err != nil {
			return ListResp{}, err
		}
		req.Scope = selectLeadScope(requestedScope, availableScopes)
	}
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
			OrgDeptID: r.OrgDeptID, OrgDeptName: r.OrgDeptName, OrgDeptInherited: r.OrgDeptInherited,
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
	for _, it := range allItems {
		if it.ParentID == 0 {
			continue
		}
		if _, exists := byID[it.ParentID]; !exists {
			byID[it.ParentID] = ListItem{
				ID: it.ParentID, Name: it.ParentName, Type: "parent", ContextOnly: true,
			}
		}
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
		ScopeOptions: scopeOpts, AvailableScopes: availableScopes, ActiveScope: req.Scope, CanEdit: req.View != "lead",
	}, nil
}

func (s *Service) availableLeadScopes(ctx context.Context, actor *model.User) ([]string, error) {
	if actor != nil && actor.IsSuperAdmin {
		return []string{"team", "dept"}, nil
	}
	account := ""
	if actor != nil {
		account = strings.TrimSpace(actor.Account)
	}
	managedIDs, err := s.repo.ListManagedTeamgroupIDs(ctx, account)
	if err != nil {
		return nil, err
	}
	scopes := make([]string, 0, 2)
	if len(managedIDs) > 0 {
		scopes = append(scopes, "team")
	}
	isDeptManager, err := s.repo.IsDeptManager(ctx, account)
	if err != nil {
		return nil, err
	}
	if isDeptManager {
		scopes = append(scopes, "dept")
	}
	return scopes, nil
}

// LeadScopeMemberAccounts resolves formal member accounts only inside the actor's
// authorized team or department scope. A selected parent team includes its
// direct child agile groups; a selected child group remains isolated.
func (s *Service) LeadScopeMemberAccounts(ctx context.Context, actor *model.User, scope string, scopeID uint) ([]string, error) {
	groupIDs, err := s.leadScopeTeamgroupIDs(ctx, actor, scope, scopeID)
	if err != nil {
		return nil, err
	}
	if len(groupIDs) == 0 {
		return []string{}, nil
	}
	membersByID, err := s.repo.ListMembersByGroupIDs(ctx, groupIDs)
	if err != nil {
		return nil, err
	}
	accounts := make([]string, 0)
	seen := make(map[string]bool)
	for _, groupID := range groupIDs {
		for _, member := range membersByID[groupID] {
			memberAccount := strings.TrimSpace(member.Account)
			if memberAccount == "" || seen[memberAccount] {
				continue
			}
			seen[memberAccount] = true
			accounts = append(accounts, memberAccount)
		}
	}
	return accounts, nil
}

// LeadScopeTeamgroupIDs resolves the authorized agile groups for one team or department selection.
// A selected parent includes its direct agile subgroups; a selected subgroup remains isolated.
func (s *Service) LeadScopeTeamgroupIDs(ctx context.Context, actor *model.User, scope string, scopeID uint) ([]uint, error) {
	return s.leadScopeTeamgroupIDs(ctx, actor, scope, scopeID)
}

func (s *Service) leadScopeTeamgroupIDs(ctx context.Context, actor *model.User, scope string, scopeID uint) ([]uint, error) {
	account, err := requireActorAccount(actor)
	if err != nil {
		return nil, err
	}
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope != "team" && scope != "dept" {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "无效的团队查看范围")
	}
	available, err := s.availableLeadScopes(ctx, actor)
	if err != nil {
		return nil, err
	}
	if !containsString(available, scope) {
		return nil, errorx.New(errorx.ErrCodeForbidden, "无权查看该团队范围")
	}

	var allowedIDs []uint
	var rows []TeamgroupRow
	if actor.IsSuperAdmin {
		rows, err = s.repo.ListTeamgroups(ctx)
		if err != nil {
			return nil, err
		}
		allowedIDs = make([]uint, 0, len(rows))
		for _, row := range rows {
			allowedIDs = append(allowedIDs, row.ID)
		}
	} else if scope == "team" {
		allowedIDs, err = s.repo.ListManagedTeamgroupIDs(ctx, account)
		if err != nil {
			return nil, err
		}
	} else {
		var deptIDs []uint
		deptIDs, err = s.repo.ListDeptTreeIDs(ctx, account)
		if err != nil {
			return nil, err
		}
		allowedIDs, err = s.repo.ListMappedTeamgroupIDsByDepts(ctx, deptIDs)
		if err != nil {
			return nil, err
		}
	}
	if rows == nil {
		rows, err = s.repo.ListTeamgroups(ctx)
		if err != nil {
			return nil, err
		}
	}
	groupIDs, visibleRows := restrictTeamgroupRows(rows, allowedIDs, scopeID)
	if scopeID > 0 && len(visibleRows) == 0 {
		return nil, errorx.New(errorx.ErrCodeForbidden, "所选团队不在你的授权范围内")
	}
	if len(groupIDs) == 0 {
		return []uint{}, nil
	}
	return groupIDs, nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func selectLeadScope(requested string, available []string) string {
	if (requested == "team" || requested == "dept") && containsString(available, requested) {
		return requested
	}
	if containsString(available, "team") {
		return "team"
	}
	return "dept"
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
	var err error
	if req.Scope == "dept" {
		var mappedIDs []uint
		if actor != nil && actor.IsSuperAdmin {
			mappedIDs = make([]uint, 0, len(rows))
			for _, row := range rows {
				mappedIDs = append(mappedIDs, row.ID)
			}
		} else {
			isMgr, err := s.repo.IsDeptManager(ctx, acc)
			if err != nil {
				return nil, nil, err
			}
			if isMgr {
				deptIDs, err := s.repo.ListDeptTreeIDs(ctx, acc)
				if err != nil {
					return nil, nil, err
				}
				mappedIDs, err = s.repo.ListMappedTeamgroupIDsByDepts(ctx, deptIDs)
				if err != nil {
					return nil, nil, err
				}
			}
		}
		_, filtered := restrictTeamgroupRows(rows, mappedIDs, req.ScopeID)
		opts, err := s.repo.ListTeamgroupOptionsByIDs(ctx, mappedIDs)
		if err != nil {
			return nil, nil, err
		}
		return opts, filtered, nil
	}
	var managedIDs []uint
	if actor != nil && actor.IsSuperAdmin {
		managedIDs = make([]uint, 0, len(rows))
		for _, row := range rows {
			managedIDs = append(managedIDs, row.ID)
		}
	} else {
		managedIDs, err = s.repo.ListManagedTeamgroupIDs(ctx, acc)
		if err != nil {
			return nil, nil, err
		}
	}
	_, filtered := restrictTeamgroupRows(rows, managedIDs, req.ScopeID)
	opts, err := s.repo.ListTeamgroupOptionsByIDs(ctx, managedIDs)
	if err != nil {
		return nil, nil, err
	}
	return opts, filtered, nil
}

func restrictTeamgroupRows(rows []TeamgroupRow, groupIDs []uint, scopeID uint) ([]uint, []TeamgroupRow) {
	available := make(map[uint]bool, len(groupIDs))
	for _, id := range groupIDs {
		available[id] = true
	}
	if scopeID != 0 {
		var selected *TeamgroupRow
		for i := range rows {
			if rows[i].ID == scopeID {
				selected = &rows[i]
				break
			}
		}
		if !available[scopeID] || selected == nil {
			groupIDs = []uint{}
		} else if selected.Type == "parent" || selected.Parent == 0 {
			children := make([]uint, 0, len(groupIDs))
			for _, row := range rows {
				if available[row.ID] && (row.ID == scopeID || row.Parent == scopeID) {
					children = append(children, row.ID)
				}
			}
			groupIDs = children
		} else {
			groupIDs = []uint{scopeID}
		}
	}
	allowed := make(map[uint]bool, len(groupIDs))
	for _, id := range groupIDs {
		allowed[id] = true
	}
	filtered := make([]TeamgroupRow, 0, len(groupIDs))
	for _, row := range rows {
		if allowed[row.ID] {
			if scopeID == 0 || row.ID == scopeID || row.Parent == scopeID || (row.Parent == 0 && row.ID == scopeID) {
				filtered = append(filtered, row)
			}
		}
	}
	return groupIDs, filtered
}

func containsUint(ids []uint, want uint) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}
