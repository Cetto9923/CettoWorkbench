// =============================================================================
// 文件: internal/module/po/repo_weekly_enrich.go
// 模块: PO 工作台
// 类型: action
// 职责: 项目周报指标查询与偏离计算（任务统计、问题、风险、工时、上线偏离）
// 依赖: gorm.io/gorm
// =============================================================================

package po

import (
	"context"
	"math"
	"strings"
	"time"
)

type projectWeeklyTaskCounts struct {
	Finished   int
	Unfinished int
	NextWeek   int
}

type releaseDevRow struct {
	ProjectID     uint   `gorm:"column:project_id"`
	ReleaseID     uint   `gorm:"column:release_id"`
	BaselineDate  string `gorm:"column:baseline_date"`
	ActualDate    string `gorm:"column:actual_date"`
	ReleaseStatus string `gorm:"column:release_status"`
	Urgent        string `gorm:"column:urgent"`
	IsMainSystem  string `gorm:"column:is_main_system"`
	Deleted       string `gorm:"column:deleted"`
	FromBaseline  int    `gorm:"column:from_baseline"`
}

// FindOpenIssueCountsByProject 未关闭问题数。
func (r *Repo) FindOpenIssueCountsByProject(ctx context.Context, projectIDs []uint) (map[uint]int, error) {
	if len(projectIDs) == 0 {
		return map[uint]int{}, nil
	}
	return r.scanProjectCounts(ctx, `
SELECT project AS project_id, COUNT(*) AS cnt
FROM zt_issue
WHERE deleted = '0' AND status <> 'closed' AND project IN ?
GROUP BY project`, projectIDs)
}

// FindOpenRiskCountsByProject 未关闭风险数。
func (r *Repo) FindOpenRiskCountsByProject(ctx context.Context, projectIDs []uint) (map[uint]int, error) {
	if len(projectIDs) == 0 {
		return map[uint]int{}, nil
	}
	return r.scanProjectCounts(ctx, `
SELECT project AS project_id, COUNT(*) AS cnt
FROM zt_risk
WHERE deleted = '0' AND status <> 'closed' AND project IN ?
GROUP BY project`, projectIDs)
}

// FindStaffCountsByProject 本周工时记录去重人数。
func (r *Repo) FindStaffCountsByProject(ctx context.Context, projectIDs []uint, monday, sunday string) (map[uint]int, error) {
	if len(projectIDs) == 0 {
		return map[uint]int{}, nil
	}
	return r.scanProjectCounts(ctx, `
SELECT e.project AS project_id, COUNT(DISTINCT ef.account) AS cnt
FROM zt_effort AS ef
INNER JOIN zt_project AS e ON e.id = ef.execution AND e.deleted = '0'
WHERE ef.objectType = 'task'
  AND ef.deleted = '0'
  AND ef.date >= ? AND ef.date <= ?
  AND e.project IN ?
GROUP BY e.project`, monday, sunday, projectIDs)
}

// FindWeeklyTaskCountsByProject 批量统计本周完成 / 未完成 / 下周计划任务数。
func (r *Repo) FindWeeklyTaskCountsByProject(ctx context.Context, projectIDs []uint, monday, sunday string) (map[uint]projectWeeklyTaskCounts, error) {
	out := map[uint]projectWeeklyTaskCounts{}
	if len(projectIDs) == 0 {
		return out, nil
	}
	nextMonday := addDaysYMD(sunday, 1)
	secondMonday := addDaysYMD(sunday, 8)

	finished, err := r.scanProjectCounts(ctx, `
SELECT e.project AS project_id, COUNT(*) AS cnt
FROM zt_task AS t
INNER JOIN zt_project AS e ON e.id = t.execution AND e.deleted = '0'
WHERE t.deleted = '0'
  AND e.project IN ?
  AND (t.status = 'done' OR t.closedReason = 'done')
  AND t.finishedDate >= ? AND t.finishedDate <= ?
GROUP BY e.project`, projectIDs, monday, sunday)
	if err != nil {
		return out, err
	}

	unfinishedOpen, err := r.scanProjectCounts(ctx, `
SELECT e.project AS project_id, COUNT(*) AS cnt
FROM zt_task AS t
INNER JOIN zt_project AS e ON e.id = t.execution AND e.deleted = '0'
WHERE t.deleted = '0'
  AND e.project IN ?
  AND t.status IN ('wait','doing','pause')
  AND t.deadline >= ? AND t.deadline <= ?
GROUP BY e.project`, projectIDs, monday, sunday)
	if err != nil {
		return out, err
	}

	unfinishedLate, err := r.scanProjectCounts(ctx, `
SELECT e.project AS project_id, COUNT(*) AS cnt
FROM zt_task AS t
INNER JOIN zt_project AS e ON e.id = t.execution AND e.deleted = '0'
WHERE t.deleted = '0'
  AND e.project IN ?
  AND t.finishedDate > ?
  AND t.deadline >= ? AND t.deadline < ?
GROUP BY e.project`, projectIDs, nextMonday, monday, nextMonday)
	if err != nil {
		return out, err
	}

	nextWeek, err := r.scanProjectCounts(ctx, `
SELECT e.project AS project_id, COUNT(*) AS cnt
FROM zt_task AS t
INNER JOIN zt_project AS e ON e.id = t.execution AND e.deleted = '0'
WHERE t.deleted = '0'
  AND e.project IN ?
  AND (
    (t.deadline >= ? AND t.deadline < ?)
    OR (t.estStarted >= ? AND t.estStarted < ?)
    OR (t.estStarted < ? AND t.deadline > ?)
  )
GROUP BY e.project`, projectIDs, nextMonday, secondMonday, nextMonday, secondMonday, nextMonday, secondMonday)
	if err != nil {
		return out, err
	}

	for _, id := range projectIDs {
		out[id] = projectWeeklyTaskCounts{
			Finished:   finished[id],
			Unfinished: unfinishedOpen[id] + unfinishedLate[id],
			NextWeek:   nextWeek[id],
		}
	}
	return out, nil
}

// FindPlanDeviationMaxDays 基线阶段最大偏离天数。
func (r *Repo) FindPlanDeviationMaxDays(ctx context.Context, projectIDs []uint) (map[uint]int, error) {
	out := map[uint]int{}
	if len(projectIDs) == 0 {
		return out, nil
	}
	type stageRow struct {
		Project     uint   `gorm:"column:project"`
		BaselineEnd string `gorm:"column:baseline_end"`
		RealEnd     string `gorm:"column:real_end"`
	}
	var rows []stageRow
	err := r.db.WithContext(ctx).Raw(`
SELECT bs.project AS project,
       DATE_FORMAT(bs.end, '%Y-%m-%d') AS baseline_end,
       DATE_FORMAT(ex.realEnd, '%Y-%m-%d') AS real_end
FROM zt_baselinestage AS bs
INNER JOIN (
  SELECT project, MAX(id) AS mid
  FROM zt_baseline
  WHERE project IN ?
  GROUP BY project
) AS lb ON lb.mid = bs.baseline
LEFT JOIN zt_project AS ex ON ex.id = bs.stage AND ex.deleted = '0'
WHERE bs.project IN ?`, projectIDs, projectIDs).Scan(&rows).Error
	if err != nil {
		return out, err
	}
	today := time.Now().In(time.Local).Format("2006-01-02")
	for _, row := range rows {
		if strings.TrimSpace(row.BaselineEnd) == "" || row.BaselineEnd == "0000-00-00" {
			continue
		}
		end := row.RealEnd
		if end == "" || end == "0000-00-00" {
			end = today
		}
		days := diffDateDays(end, row.BaselineEnd)
		if days > out[row.Project] {
			out[row.Project] = days
		}
	}
	return out, nil
}

// FindReleaseRiskByProject 项目最严重上线风险码。
func (r *Repo) FindReleaseRiskByProject(ctx context.Context, projectIDs []uint) (map[uint]string, error) {
	out := map[uint]string{}
	if len(projectIDs) == 0 {
		return out, nil
	}
	var baselineReleases []releaseDevRow
	err := r.db.WithContext(ctx).Raw(`
SELECT br.project AS project_id,
       br.release AS release_id,
       DATE_FORMAT(br.date, '%Y-%m-%d') AS baseline_date,
       DATE_FORMAT(rl.date, '%Y-%m-%d') AS actual_date,
       COALESCE(rl.releaseStatus, '') AS release_status,
       COALESCE(rl.urgent, 'no') AS urgent,
       COALESCE(br.isMainSystem, '') AS is_main_system,
       COALESCE(rl.deleted, '1') AS deleted,
       1 AS from_baseline
FROM zt_baselinerelease AS br
LEFT JOIN zt_release AS rl ON rl.id = br.release
WHERE br.project IN ?`, projectIDs).Scan(&baselineReleases).Error
	if err != nil {
		return out, err
	}

	var extraReleases []releaseDevRow
	err = r.db.WithContext(ctx).Raw(`
SELECT p.id AS project_id,
       rl.id AS release_id,
       '' AS baseline_date,
       DATE_FORMAT(rl.date, '%Y-%m-%d') AS actual_date,
       COALESCE(rl.releaseStatus, '') AS release_status,
       COALESCE(rl.urgent, 'no') AS urgent,
       '' AS is_main_system,
       COALESCE(rl.deleted, '0') AS deleted,
       0 AS from_baseline
FROM zt_project AS p
JOIN zt_release AS rl ON FIND_IN_SET(p.id, REPLACE(rl.project, ' ', '')) > 0
WHERE p.id IN ? AND rl.deleted = '0'`, projectIDs).Scan(&extraReleases).Error
	if err != nil {
		return out, err
	}

	byProject := map[uint]map[uint]releaseDevRow{}
	mergeRelease := func(row releaseDevRow) {
		if row.ProjectID == 0 || row.ReleaseID == 0 {
			return
		}
		if byProject[row.ProjectID] == nil {
			byProject[row.ProjectID] = map[uint]releaseDevRow{}
		}
		prev, ok := byProject[row.ProjectID][row.ReleaseID]
		if !ok {
			byProject[row.ProjectID][row.ReleaseID] = row
			return
		}
		if row.FromBaseline == 1 {
			if prev.ActualDate != "" {
				row.ActualDate = prev.ActualDate
				row.ReleaseStatus = prev.ReleaseStatus
				row.Urgent = prev.Urgent
				row.Deleted = prev.Deleted
			}
			byProject[row.ProjectID][row.ReleaseID] = row
			return
		}
		if prev.FromBaseline == 1 {
			prev.ActualDate = row.ActualDate
			prev.ReleaseStatus = row.ReleaseStatus
			prev.Urgent = row.Urgent
			prev.Deleted = row.Deleted
			byProject[row.ProjectID][row.ReleaseID] = prev
		}
	}
	for _, row := range baselineReleases {
		mergeRelease(row)
	}
	for _, row := range extraReleases {
		mergeRelease(row)
	}

	today := time.Now().In(time.Local).Format("2006-01-02")
	for pid, releases := range byProject {
		worst := "normal"
		hasAny := false
		for _, rel := range releases {
			if rel.FromBaseline == 0 && rel.Deleted == "1" {
				continue
			}
			if rel.FromBaseline == 1 && rel.Deleted == "1" {
				continue
			}
			hasAny = true
			risk := calculateReleaseRiskSituation(rel, today)
			if releaseRiskRank(risk) > releaseRiskRank(worst) {
				worst = risk
			}
		}
		if hasAny {
			out[pid] = worst
		}
	}
	return out, nil
}

func calculateReleaseRiskSituation(rel releaseDevRow, today string) string {
	if rel.IsMainSystem == "auxiliary" || rel.IsMainSystem == "other" {
		return "normal"
	}
	deviationDays := 0.0
	if rel.BaselineDate != "" && rel.BaselineDate != "0000-00-00" {
		actual := rel.ActualDate
		if rel.ReleaseStatus == "draft" {
			actual = today
		}
		if actual != "" && actual != "0000-00-00" {
			deviationDays = float64(diffDateDays(actual, rel.BaselineDate))
		}
	}
	if rel.BaselineDate != "" && rel.BaselineDate != "0000-00-00" {
		if rel.ReleaseStatus != "draft" {
			if deviationDays > 0 {
				return "delayRelease"
			}
			if deviationDays < 0 {
				return "earlyRelease"
			}
			return "normal"
		}
		if rel.Urgent == "no" {
			if deviationDays > 0 {
				return "delayRelease"
			}
			if deviationDays <= 0 && deviationDays >= -7 {
				return "hasRisk"
			}
			return "normal"
		}
		if deviationDays > 0 {
			return "delayRelease"
		}
		return "normal"
	}
	if rel.Urgent == "no" {
		return "abnormalRelease"
	}
	return "normal"
}

func releaseRiskRank(code string) int {
	switch code {
	case "abnormalRelease":
		return 5
	case "delayRelease":
		return 4
	case "hasRisk":
		return 3
	case "earlyRelease":
		return 2
	case "normal":
		return 1
	default:
		return 0
	}
}

func diffDateDays(a, b string) int {
	ta := parseDateYMD(a)
	tb := parseDateYMD(b)
	if ta.IsZero() || tb.IsZero() {
		return 0
	}
	hours := ta.Sub(tb).Hours()
	return int(math.Floor(hours / 24))
}

func addDaysYMD(s string, days int) string {
	t := parseDateYMD(s)
	if t.IsZero() {
		return s
	}
	return t.AddDate(0, 0, days).Format("2006-01-02")
}
