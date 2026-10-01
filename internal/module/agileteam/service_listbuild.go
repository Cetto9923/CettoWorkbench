// =============================================================================
// 文件: internal/module/agileteam/service_listbuild.go
// 模块: 敏捷小组治理
// 类型: action
// 职责: 列表行的批量数据装载与逐行装配，供 service_list.go 的 List 调用。
// 依赖: internal/module/agileteam/service_list.go
// =============================================================================

package agileteam

import (
	"context"
	"strings"
	"time"
)

// listRowData 列表逐行装配所需的批量数据。
type listRowData struct {
	membersByID map[uint][]TeamMemberRow
	pendingByID map[uint]Adjustment
	counts      map[int64]struct{ Add, Remove, Change int }
	lastTimes   map[uint]time.Time
	names       map[string]string
}

// loadListRowData 按小组 ID 批量装载成员、待调整、计数、最近调整时间与姓名。
func (s *Service) loadListRowData(ctx context.Context, rows []TeamgroupRow) (listRowData, error) {
	ids := make([]uint, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	membersByID, err := s.repo.ListMembersByGroupIDs(ctx, ids)
	if err != nil {
		return listRowData{}, err
	}
	pendingByID, err := s.repo.ListPendingByTeamgroupIDs(ctx, ids)
	if err != nil {
		return listRowData{}, err
	}
	adjIDs := make([]int64, 0, len(pendingByID))
	for _, p := range pendingByID {
		adjIDs = append(adjIDs, p.ID)
	}
	counts, err := s.repo.CountPendingItems(ctx, adjIDs)
	if err != nil {
		return listRowData{}, err
	}
	lastTimes, err := s.repo.LastAdjustTimes(ctx, ids)
	if err != nil {
		return listRowData{}, err
	}

	accounts := []string{}
	for _, r := range rows {
		accounts = append(accounts, r.PO, strings.TrimSpace(r.Manager))
	}
	names, _ := s.repo.ResolveRealnames(ctx, accounts)

	return listRowData{
		membersByID: membersByID,
		pendingByID: pendingByID,
		counts:      counts,
		lastTimes:   lastTimes,
		names:       names,
	}, nil
}

// listStats 列表的全量状态计数。
type listStats struct {
	all     int64
	enable  int64
	disable int64
	pending int64
}

// buildListItems 逐行装配列表项，并按过滤条件挑出匹配行，同时累计状态计数。
func buildListItems(rows []TeamgroupRow, data listRowData, req ListReq) ([]ListItem, []ListItem, listStats) {
	var stats listStats
	allItems := make([]ListItem, 0, len(rows))
	matched := make([]ListItem, 0, len(rows))

	for _, r := range rows {
		st := strings.TrimSpace(r.Status)
		if st == "" || st == "enable" || st == "doing" {
			stats.enable++
		} else {
			stats.disable++
		}
		stats.all++

		formal := data.membersByID[r.ID]
		pad, prem := 0, 0
		var pendID int64
		if p, ok := data.pendingByID[r.ID]; ok {
			stats.pending++
			pendID = p.ID
			c := data.counts[p.ID]
			pad, prem = c.Add, c.Remove
		}
		coachAcc := firstAccount(r.Manager)
		poAcc := strings.TrimSpace(r.PO)
		lastAt := ""
		if t, ok := data.lastTimes[r.ID]; ok {
			lastAt = formatTime(t)
		}
		item := ListItem{
			ID: r.ID, Name: r.Name, ParentID: r.Parent, ParentName: r.ParentName,
			OrgDeptID: r.OrgDeptID, OrgDeptName: r.OrgDeptName, OrgDeptInherited: r.OrgDeptInherited,
			Type: teamTypeOf(r), CoachAccount: coachAcc, CoachName: data.names[coachAcc],
			POAccount: poAcc, POName: data.names[poAcc],
			FormalCount: len(formal), PendingAdd: pad, PendingRemove: prem,
			Status: st, StatusLabel: statusLabel(st),
			LastAdjustAt: lastAt, PendingAdjustID: pendID,
		}
		allItems = append(allItems, item)
		if matchListFilters(r, formal, pad > 0 || prem > 0, data.names, req) {
			matched = append(matched, item)
		}
	}
	return allItems, matched, stats
}

// indexListItems 建立全量行索引，并为只有子行出现的父小组补一个上下文占位项。
func indexListItems(allItems []ListItem) map[uint]ListItem {
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
	return byID
}
