// =============================================================================
// 文件: internal/module/po/service_weekly.go
// 模块: PO 工作台
// 类型: action
// 职责: 项目周报聚合列表 / 详情 / 历史 / 承建团队业务组装
// 依赖: internal/model
//       internal/pkg/zentao
// =============================================================================

package po

import (
	"context"
	"fmt"
	"strings"
	"time"

	"workbench/internal/model"
	"workbench/internal/pkg/zentao"
)

var overallSituationLabels = map[int]string{
	0: "正常",
	1: "低风险",
	2: "高风险",
}

var releaseRiskLabels = map[string]string{
	"normal":          "正常",
	"delayRelease":    "延期",
	"hasRisk":         "延期风险",
	"earlyRelease":    "提前上线",
	"abnormalRelease": "异常发布",
}

type weeklyReportDeadlineCfg struct {
	weekday int    // 1=Mon … 7=Sun
	clock   string // HH:MM
}

func defaultWeeklyReportDeadline() weeklyReportDeadlineCfg {
	return weeklyReportDeadlineCfg{weekday: 5, clock: "17:30"}
}

// ProjectWeeklies 项目周报聚合列表 + 同源统计（scope=watched|all）。
func (s *Service) ProjectWeeklies(ctx context.Context, actor *model.User, req ProjectWeeklyListReq) (ProjectWeeklyListResp, error) {
	teamIDs := req.TeamIDList()
	resp := ProjectWeeklyListResp{Items: []ProjectWeeklyListItem{}, Stats: ProjectWeeklyStats{}, Scope: req.Scope, TeamID: req.TeamID}
	if len(teamIDs) == 1 {
		resp.TeamID = teamIDs[0]
	}
	if s.repo == nil || actor == nil || strings.TrimSpace(actor.Account) == "" {
		return resp, nil
	}
	now := time.Now().In(time.Local)
	monday, sunday := naturalWeekBounds(now)
	deadlineAt := weeklyDeadlineAt(monday, defaultWeeklyReportDeadline())
	resp.WeekStart = monday
	resp.WeekEnd = sunday
	resp.DeadlineAt = deadlineAt.Format("2006-01-02 15:04:05")

	var projects []projectWeeklyProjectRow
	var err error
	if req.Scope == "all" {
		projects, err = s.repo.FindAllProjectWeeklyProjects(ctx, teamIDs, req.Limit)
	} else {
		projects, err = s.repo.FindWatchedProjectWeeklyProjects(ctx, strings.TrimSpace(actor.Account), teamIDs, req.Limit)
	}
	if err != nil {
		return resp, err
	}
	if len(projects) == 0 {
		return resp, nil
	}
	ids := make([]uint, 0, len(projects))
	for _, p := range projects {
		ids = append(ids, p.ID)
	}

	currentReports, err := s.repo.FindWeeklyReportsByWeekStart(ctx, ids, monday)
	if err != nil {
		return resp, err
	}
	latestReports, err := s.repo.FindLatestWeeklyReports(ctx, ids)
	if err != nil {
		return resp, err
	}
	currentByProject := map[uint]projectWeeklyReportRow{}
	for _, row := range currentReports {
		currentByProject[row.Project] = row
	}
	latestByProject := map[uint]projectWeeklyReportRow{}
	for _, row := range latestReports {
		if _, ok := latestByProject[row.Project]; !ok {
			latestByProject[row.Project] = row
		}
	}

	reportIDs := make([]uint, 0, len(currentReports)+len(latestReports))
	for _, row := range currentReports {
		reportIDs = append(reportIDs, row.ID)
	}
	for _, row := range latestReports {
		reportIDs = append(reportIDs, row.ID)
	}
	edited, err := s.repo.FindWeeklyEditedReportIDs(ctx, reportIDs)
	if err != nil {
		return resp, err
	}

	taskCounts, err := s.repo.FindWeeklyTaskCountsByProject(ctx, ids, monday, sunday)
	if err != nil {
		return resp, err
	}
	issueCounts, err := s.repo.FindOpenIssueCountsByProject(ctx, ids)
	if err != nil {
		return resp, err
	}
	riskCounts, err := s.repo.FindOpenRiskCountsByProject(ctx, ids)
	if err != nil {
		return resp, err
	}
	staffCounts, err := s.repo.FindStaffCountsByProject(ctx, ids, monday, sunday)
	if err != nil {
		return resp, err
	}
	planDays, err := s.repo.FindPlanDeviationMaxDays(ctx, ids)
	if err != nil {
		return resp, err
	}
	releaseRisks, err := s.repo.FindReleaseRiskByProject(ctx, ids)
	if err != nil {
		return resp, err
	}

	items := make([]ProjectWeeklyListItem, 0, len(projects))
	for _, p := range projects {
		item := s.buildProjectWeeklyItem(p, monday, sunday, now, deadlineAt,
			currentByProject[p.ID], latestByProject[p.ID], edited,
			taskCounts[p.ID], issueCounts[p.ID], riskCounts[p.ID],
			staffCounts[p.ID], planDays[p.ID], releaseRisks[p.ID])
		items = append(items, item)
	}

	resp.Stats = computeProjectWeeklyStats(items)
	filtered := filterProjectWeeklyItems(items, req.Filter, req.Keyword)
	resp.Items = filtered
	resp.Total = len(filtered)
	return resp, nil
}

// ProjectWeeklyDetail 侧滑详情。
func (s *Service) ProjectWeeklyDetail(ctx context.Context, actor *model.User, projectID uint, scope string) (ProjectWeeklyDetailResp, error) {
	out := ProjectWeeklyDetailResp{}
	if s.repo == nil || actor == nil || projectID == 0 {
		return out, fmt.Errorf("project weekly not found")
	}
	if scope == "" {
		scope = "watched"
	}
	list, err := s.ProjectWeeklies(ctx, actor, ProjectWeeklyListReq{Filter: "all", Limit: 500, Scope: scope})
	if err != nil {
		return out, err
	}
	for _, item := range list.Items {
		if item.ProjectID == projectID {
			out.ProjectWeeklyListItem = item
			if item.OverallSituation == 0 && (item.PlanDeviationDays > 0 || releaseRiskAbnormal(item.ReleaseRisk)) {
				out.Warning = "项目状态评估为「正常」，但系统数据存在计划/上线偏离，请结合实际情况确认。"
			}
			return out, nil
		}
	}
	return out, fmt.Errorf("project weekly not found")
}

// ProjectWeeklyTeams 承建团队树。
func (s *Service) ProjectWeeklyTeams(ctx context.Context, actor *model.User, req ProjectWeeklyTeamsReq) (ProjectWeeklyTeamsResp, error) {
	out := ProjectWeeklyTeamsResp{Teams: []ProjectWeeklyTeamNode{}}
	if s.repo == nil || actor == nil {
		return out, nil
	}
	leaves, err := s.repo.FindProjectWeeklyTeamLeaves(ctx)
	if err != nil {
		return out, err
	}
	ids := collectDeptIDsFromPaths(leaves)
	rows, err := s.repo.FindDeptsByIDs(ctx, ids)
	if err != nil {
		return out, err
	}
	out.Teams = buildProjectWeeklyDeptTree(rows)
	deptID, err := s.repo.FindUserDeptID(ctx, strings.TrimSpace(actor.Account))
	if err != nil {
		return out, err
	}
	out.ActorDeptID = deptID
	if req.PreferDefaultMine() && deptID > 0 {
		out.DefaultTeamID = deptID
		out.Teams = ensureDeptInTree(out.Teams, deptID, "我的团队")
	}
	return out, nil
}

// ProjectWeeklyHistory 历史周报。
func (s *Service) ProjectWeeklyHistory(ctx context.Context, actor *model.User, projectID uint, scope string) ([]ProjectWeeklyHistoryItem, error) {
	if s.repo == nil || actor == nil || projectID == 0 {
		return nil, fmt.Errorf("project weekly not found")
	}
	if scope == "" {
		scope = "watched"
	}
	var proj *projectWeeklyProjectRow
	var err error
	if scope == "all" {
		proj, err = s.repo.FindProjectWeeklyByIDAny(ctx, projectID)
	} else {
		proj, err = s.repo.FindProjectWeeklyByID(ctx, strings.TrimSpace(actor.Account), projectID)
	}
	if err != nil {
		return nil, err
	}
	if proj == nil {
		return nil, fmt.Errorf("project weekly not found")
	}
	rows, err := s.repo.FindWeeklyReportHistory(ctx, projectID, 50)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	edited, err := s.repo.FindWeeklyEditedReportIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	planDays, _ := s.repo.FindPlanDeviationMaxDays(ctx, []uint{projectID})
	releaseRisks, _ := s.repo.FindReleaseRiskByProject(ctx, []uint{projectID})
	issueCounts, _ := s.repo.FindOpenIssueCountsByProject(ctx, []uint{projectID})
	riskCounts, _ := s.repo.FindOpenRiskCountsByProject(ctx, []uint{projectID})

	out := make([]ProjectWeeklyHistoryItem, 0, len(rows))
	for _, row := range rows {
		ws := row.WeekStart
		we := addDaysYMD(ws, 6)
		submitted := isWeeklySubmitted(row, edited)
		status := "waiting"
		if submitted {
			status = "submitted"
		}
		out = append(out, ProjectWeeklyHistoryItem{
			ReportID:              row.ID,
			WeekStart:             ws,
			WeekEnd:               we,
			OverallSituation:      row.OverallSituation,
			OverallSituationLabel: overallSituationLabel(row.OverallSituation),
			OpenIssueCount:        issueCounts[projectID],
			OpenRiskCount:         riskCounts[projectID],
			PlanDeviationDays:     planDays[projectID],
			ReleaseRisk:           releaseRisks[projectID],
			ReleaseRiskLabel:      releaseRiskLabel(releaseRisks[projectID]),
			SubmitStatus:          status,
			Staff:                 row.Staff,
			WorkloadHours:         sumWorkloadHours(row.Workload),
		})
	}
	return out, nil
}

func (s *Service) buildProjectWeeklyItem(
	p projectWeeklyProjectRow,
	monday, sunday string,
	now, deadlineAt time.Time,
	current, latest projectWeeklyReportRow,
	edited map[uint]bool,
	tasks projectWeeklyTaskCounts,
	issueCnt, riskCnt, staff, planDays int,
	releaseRisk string,
) ProjectWeeklyListItem {
	display := current
	useCurrent := current.ID > 0
	if !useCurrent {
		display = latest
	}
	weekStart := monday
	weekEnd := sunday
	if display.ID > 0 && display.WeekStart != "" {
		weekStart = display.WeekStart
		weekEnd = addDaysYMD(weekStart, 6)
	}
	submittedCurrent := isWeeklySubmitted(current, edited)
	submitStatus := "waiting"
	if submittedCurrent {
		submitStatus = "submitted"
	} else if now.After(deadlineAt) {
		submitStatus = "overdue"
	}

	staffVal := staff
	workload := 0.0
	if useCurrent && current.ID > 0 {
		if current.Staff > 0 {
			staffVal = current.Staff
		}
		workload = sumWorkloadHours(current.Workload)
	} else if display.ID > 0 {
		staffVal = display.Staff
		workload = sumWorkloadHours(display.Workload)
	}

	situation := display.OverallSituation
	desc := strings.TrimSpace(display.OverallSituationDesc)
	zentaoURL := ""
	if display.ID > 0 {
		zentaoURL = zentao.WeeklyIndexURL(p.ID, weekStart)
	}
	item := ProjectWeeklyListItem{
		ProjectID:             p.ID,
		ProjectCode:           strings.TrimSpace(p.Code),
		ProjectName:           strings.TrimSpace(p.Name),
		PMAccount:             strings.TrimSpace(p.PM),
		PMName:                strings.TrimSpace(p.PMName),
		PMDeptID:              p.PMDeptID,
		PMDeptName:            strings.TrimSpace(p.PMDeptName),
		ReportID:              display.ID,
		WeekStart:             weekStart,
		WeekEnd:               weekEnd,
		WeekSN:                weekSN(p.Begin, weekStart),
		SubmitStatus:          submitStatus,
		OverallSituation:      situation,
		OverallSituationLabel: overallSituationLabel(situation),
		OverallSituationDesc:  desc,
		FinishedCount:         tasks.Finished,
		UnfinishedCount:       tasks.Unfinished,
		NextWeekCount:         tasks.NextWeek,
		OpenIssueCount:        issueCnt,
		OpenRiskCount:         riskCnt,
		PlanDeviationDays:     planDays,
		ReleaseRisk:           releaseRisk,
		ReleaseRiskLabel:      releaseRiskLabel(releaseRisk),
		Staff:                 staffVal,
		WorkloadHours:         workload,
		URL:                   zentaoURL,
	}
	item.HasAbnormal = projectWeeklyAbnormal(item)
	return item
}
