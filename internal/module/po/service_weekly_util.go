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
