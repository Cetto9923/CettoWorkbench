// =============================================================================
// 文件: internal/module/po/service_weekly_util.go
// 模块: PO 工作台
// 类型: action
// 职责: 项目周报业务辅助函数（提交状态判定、异常口径、统计与过滤、日期处理、部门树）
// 依赖: 无
// =============================================================================

package po

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

func isWeeklySubmitted(row projectWeeklyReportRow, edited map[uint]bool) bool {
	if row.ID == 0 {
		return false
	}
	if edited[row.ID] {
		return true
	}
	if strings.TrimSpace(row.OverallSituationDesc) != "" {
		return true
	}
	return row.OverallSituation != 0
}

func projectWeeklyAbnormal(item ProjectWeeklyListItem) bool {
	if item.OverallSituation == 1 || item.OverallSituation == 2 {
		return true
	}
	if item.OpenIssueCount > 0 || item.OpenRiskCount > 0 {
		return true
	}
	if item.PlanDeviationDays > 0 {
		return true
	}
	if releaseRiskAbnormal(item.ReleaseRisk) {
		return true
	}
	if item.SubmitStatus == "overdue" {
		return true
	}
	return false
}

func releaseRiskAbnormal(code string) bool {
	switch code {
	case "delayRelease", "hasRisk", "abnormalRelease":
		return true
	default:
		return false
	}
}

func computeProjectWeeklyStats(items []ProjectWeeklyListItem) ProjectWeeklyStats {
	stats := ProjectWeeklyStats{Watched: len(items)}
	for _, item := range items {
		if item.SubmitStatus == "submitted" {
			stats.Submitted++
		} else {
			stats.Waiting++
		}
		if item.HasAbnormal {
			stats.Abnormal++
		}
		if item.OverallSituation == 1 || item.OverallSituation == 2 {
			stats.Attention++
		}
		if item.OpenIssueCount > 0 || item.OpenRiskCount > 0 {
			stats.Risk++
		}
		if item.PlanDeviationDays > 0 || releaseRiskAbnormal(item.ReleaseRisk) || item.ReleaseRisk == "earlyRelease" {
			stats.Deviation++
		}
	}
	return stats
}

func filterProjectWeeklyItems(items []ProjectWeeklyListItem, filter, keyword string) []ProjectWeeklyListItem {
	out := make([]ProjectWeeklyListItem, 0, len(items))
	kw := strings.ToLower(strings.TrimSpace(keyword))
	for _, item := range items {
		if kw != "" {
			blob := strings.ToLower(strings.Join([]string{
				item.ProjectCode, item.ProjectName, item.PMName, item.PMAccount,
			}, " "))
			if !strings.Contains(blob, kw) {
				continue
			}
		}
		switch filter {
		case "waiting":
			if item.SubmitStatus != "waiting" && item.SubmitStatus != "overdue" {
				continue
			}
		case "submitted":
			if item.SubmitStatus != "submitted" {
				continue
			}
		case "attention":
			if item.OverallSituation != 1 && item.OverallSituation != 2 {
				continue
			}
		case "risk":
			if item.OpenIssueCount <= 0 && item.OpenRiskCount <= 0 {
				continue
			}
		case "deviation":
			if item.PlanDeviationDays <= 0 && !releaseRiskAbnormal(item.ReleaseRisk) && item.ReleaseRisk != "earlyRelease" {
				continue
			}
		case "abnormal":
			if !item.HasAbnormal {
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

func overallSituationLabel(v int) string {
	if label, ok := overallSituationLabels[v]; ok {
		return label
	}
	return fmt.Sprintf("%d", v)
}

func releaseRiskLabel(code string) string {
	if code == "" {
		return ""
	}
	if label, ok := releaseRiskLabels[code]; ok {
		return label
	}
	return code
}

func naturalWeekBounds(now time.Time) (monday, sunday string) {
	wd := int(now.Weekday())
	if wd == 0 {
		wd = 7
	}
	mon := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(wd - 1))
	sun := mon.AddDate(0, 0, 6)
	return mon.Format("2006-01-02"), sun.Format("2006-01-02")
}

func weeklyDeadlineAt(monday string, cfg weeklyReportDeadlineCfg) time.Time {
	mon := parseDateYMD(monday)
	if mon.IsZero() {
		mon = time.Now()
	}
	offset := cfg.weekday - 1
	if offset < 0 {
		offset = 4
	}
	day := mon.AddDate(0, 0, offset)
	hour, minute := 17, 30
	parts := strings.Split(cfg.clock, ":")
	if len(parts) >= 2 {
		if h, err := strconv.Atoi(parts[0]); err == nil {
			hour = h
		}
		if m, err := strconv.Atoi(parts[1]); err == nil {
			minute = m
		}
	}
	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, time.Local)
}

func weekSN(projectBegin, weekStart string) int {
	b := parseDateYMD(projectBegin)
	w := parseDateYMD(weekStart)
	if b.IsZero() || w.IsZero() {
		return 0
	}
	days := int(w.Sub(b).Hours() / 24)
	if days < 0 {
		return 1
	}
	return days/7 + 1
}

func uniquePositiveUints(ids []uint) []uint {
	if len(ids) == 0 {
		return nil
	}
	seen := map[uint]struct{}{}
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func collectDeptIDsFromPaths(leaves []projectWeeklyTeamRow) []uint {
	seen := map[uint]struct{}{}
	out := make([]uint, 0, len(leaves)*3)
	add := func(id uint) {
		if id == 0 {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, leaf := range leaves {
		add(leaf.ID)
		for _, part := range strings.Split(leaf.Path, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			n, err := strconv.ParseUint(part, 10, 64)
			if err != nil {
				continue
			}
			add(uint(n))
		}
	}
	return out
}

func buildProjectWeeklyDeptTree(rows []projectWeeklyTeamRow) []ProjectWeeklyTeamNode {
	byID := make(map[uint]projectWeeklyTeamRow, len(rows))
	for _, row := range rows {
		if row.ID == 0 {
			continue
		}
		byID[row.ID] = row
	}
	childrenOf := make(map[uint][]uint, len(rows))
	rootIDs := make([]uint, 0, 8)
	seenRoot := map[uint]struct{}{}
	addRoot := func(id uint) {
		if _, ok := seenRoot[id]; ok {
			return
		}
		seenRoot[id] = struct{}{}
		rootIDs = append(rootIDs, id)
	}
	for _, row := range rows {
		if row.ID == 0 {
			continue
		}
		parent, ok := byID[row.ParentID]
		if !ok || parent.Grade == 1 {
			if row.Grade == 1 {
				continue
			}
			addRoot(row.ID)
			continue
		}
		childrenOf[row.ParentID] = append(childrenOf[row.ParentID], row.ID)
	}
	for _, row := range rows {
		if row.Grade != 1 {
			continue
		}
		if len(childrenOf[row.ID]) > 0 {
			continue
		}
		hasDisplayChild := false
		for _, r := range rows {
			if r.ParentID == row.ID && r.Grade != 1 {
				hasDisplayChild = true
				break
			}
		}
		if !hasDisplayChild {
			addRoot(row.ID)
		}
	}
	var walk func(id uint) ProjectWeeklyTeamNode
	walk = func(id uint) ProjectWeeklyTeamNode {
		row := byID[id]
		node := ProjectWeeklyTeamNode{
			ID:       row.ID,
			Name:     strings.TrimSpace(row.Name),
			ParentID: row.ParentID,
			Grade:    row.Grade,
			Path:     strings.TrimSpace(row.Path),
		}
		kids := childrenOf[id]
		sort.SliceStable(kids, func(i, j int) bool {
			a, b := byID[kids[i]], byID[kids[j]]
			if a.Path != b.Path {
				return a.Path < b.Path
			}
			if a.Order != b.Order {
				return a.Order < b.Order
			}
			return a.ID < b.ID
		})
		for _, cid := range kids {
			node.Children = append(node.Children, walk(cid))
		}
		return node
	}
	out := make([]ProjectWeeklyTeamNode, 0, len(rootIDs))
	sort.SliceStable(rootIDs, func(i, j int) bool {
		a, b := byID[rootIDs[i]], byID[rootIDs[j]]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Order != b.Order {
			return a.Order < b.Order
		}
		return a.ID < b.ID
	})
	for _, rid := range rootIDs {
		out = append(out, walk(rid))
	}
	return out
}

func ensureDeptInTree(tree []ProjectWeeklyTeamNode, deptID uint, label string) []ProjectWeeklyTeamNode {
	found := false
	var walk func(nodes []ProjectWeeklyTeamNode)
	walk = func(nodes []ProjectWeeklyTeamNode) {
		for _, n := range nodes {
			if n.ID == deptID {
				found = true
				return
			}
			walk(n.Children)
		}
	}
	walk(tree)
	if found {
		return tree
	}
	node := ProjectWeeklyTeamNode{ID: deptID, Name: label}
	return append([]ProjectWeeklyTeamNode{node}, tree...)
}
