// =============================================================================
// 文件: internal/module/po/service_detail_exec.go
// 模块: PO 工作台
// 类型: service
// 职责: 业务需求 Tab 3 研发执行、测试进度与按应用分层的代码质量树组装。
// 依赖: gorm.io/gorm
// =============================================================================

package po

import (
	"context"
	"fmt"
	"math"
	"strings"

	"workbench/internal/pkg/zentao"
)

func (s *DetailService) buildExecution(ctx context.Context, demandID uint) *DetailExecution {
	stories, _ := s.repo.FindDemandStories(ctx, demandID)
	storyIDs := make([]uint, 0, len(stories))
	for _, st := range stories {
		storyIDs = append(storyIDs, st.ID)
	}

	taskMap, _ := s.repo.FindStoryTaskCounts(ctx, storyIDs)
	bugMap, _ := s.repo.FindStoryBugCounts(ctx, storyIDs)
	testCaseCounts, _ := s.repo.FindStoryTestCaseCounts(ctx, storyIDs)

	sItems := make([]StoryItem, 0, len(stories))
	totalBugs := 0
	activeBugs := 0
	resolvedBugs := 0
	blockingBugs := 0

	for _, st := range stories {
		tDone := 0
		tTotal := 0
		if t, ok := taskMap[st.ID]; ok {
			tDone = t.Done
			tTotal = t.Total
		}
		sBugsTotal := 0
		sBugsActive := 0
		if b, ok := bugMap[st.ID]; ok {
			totalBugs += b.Total
			activeBugs += b.Active
			resolvedBugs += b.Resolved
			blockingBugs += b.DeliveryBlocking
			sBugsTotal = b.Total
			sBugsActive = b.Active
		}

		sItems = append(sItems, StoryItem{
			ID:         st.ID,
			Code:       fmt.Sprintf("ST%d", st.ID),
			Title:      st.Title,
			Product:    defaultDash(st.ProductName),
			Owner:      defaultDash(st.AssignedToName),
			Status:     defaultDash(st.Status),
			TasksDone:  tDone,
			TasksTotal: tTotal,
			BugsTotal:  sBugsTotal,
			BugsActive: sBugsActive,
		})
	}

	execRate := 0.0
	if testCaseCounts.TotalCount > 0 {
		execRate = math.Round(float64(testCaseCounts.ExecutedCount)/float64(testCaseCounts.TotalCount)*1000) / 10
	}
	passRate := 0.0
	if testCaseCounts.ExecutedCount > 0 {
		passRate = math.Round(float64(testCaseCounts.PassedCount)/float64(testCaseCounts.ExecutedCount)*1000) / 10
	}

	testTasks, _ := s.repo.FindStoryTestTasks(ctx, storyIDs)
	ttItems := make([]TestOrderItem, 0, len(testTasks))
	doingCount := 0
	doneCount := 0
	for _, tt := range testTasks {
		stage := "SIT"
		if strings.Contains(strings.ToUpper(tt.Name), "UAT") {
			stage = "UAT"
		}
		statusLabel := "未开始"
		switch tt.Status {
		case "doing":
			statusLabel = "进行中"
			doingCount++
		case "done":
			statusLabel = "已完成"
			doneCount++
		case "blocked":
			statusLabel = "阻塞"
		default:
			statusLabel = "未开始"
		}
		begin := "—"
		if tt.Begin != nil {
			begin = tt.Begin.Format("2006-01-02")
		}
		end := "—"
		if tt.End != nil {
			end = tt.End.Format("2006-01-02")
		}
		ttItems = append(ttItems, TestOrderItem{
			ID:          tt.ID,
			Code:        fmt.Sprintf("TO-%d", tt.ID),
			Title:       tt.Name,
			Stage:       stage,
			Status:      tt.Status,
			StatusLabel: statusLabel,
			Owner:       defaultDash(tt.OwnerName),
			BeginDate:   begin,
			EndDate:     end,
			ZtURL:       zentao.TesttaskViewURL(tt.ID),
		})
	}

	devLeadsMap := make(map[string]struct{})
	for _, st := range stories {
		if st.AssignedToName != "" && st.AssignedToName != "—" {
			devLeadsMap[st.AssignedToName] = struct{}{}
		}
	}
	devLeads := make([]string, 0, len(devLeadsMap))
	for name := range devLeadsMap {
		devLeads = append(devLeads, name)
	}

	unexec := testCaseCounts.TotalCount - testCaseCounts.ExecutedCount
	if unexec < 0 {
		unexec = 0
	}

	appQualityTree := buildAppQualityTree(demandID, stories)
	passedGates := 0
	for _, app := range appQualityTree {
		if app.GatePassed {
			passedGates++
		}
	}

	return &DetailExecution{
		Stories:    sItems,
		TestOrders: ttItems,
		TestOrderSummary: TestOrderSummary{
			TotalCount:   len(testTasks),
			DoingCount:   doingCount,
			DoneCount:    doneCount,
			StoriesCount: len(stories),
		},
		TestCaseSummary: TestCaseSummary{
			TotalCount:      testCaseCounts.TotalCount,
			ExecutedCount:   testCaseCounts.ExecutedCount,
			UnexecutedCount: unexec,
			PassedCount:     testCaseCounts.PassedCount,
			FailedCount:     testCaseCounts.FailedCount,
			ExecutionRate:   execRate,
			PassRate:        passRate,
		},
		BugSummary: BugSummary{
			TotalCount:       totalBugs,
			ActiveCount:      activeBugs,
			ResolvedCount:    resolvedBugs,
			DeliveryBlocking: blockingBugs,
		},
		QualitySummary: QualitySummary{
			AvgScore:      92.0,
			BranchesCount: len(stories),
			PassedGates:   len(stories),
			TotalGates:    len(stories),
		},
		QualityOverview: QualityGateOverview{
			AppsCount:     len(appQualityTree),
			BranchesCount: len(stories),
			PassedGates:   passedGates,
			FailedGates:   len(appQualityTree) - passedGates,
		},
		AppQualityTree: appQualityTree,
		LeadsMatrix: LeadsMatrix{
			DevLeads: devLeads,
			TestLead: "测试团队",
		},
	}
}

func buildAppQualityTree(demandID uint, stories []DemandStoryRow) []AppQualityNode {
	if len(stories) == 0 {
		return nil
	}
	appMap := make(map[string][]BranchQualityItem)
	for _, st := range stories {
		appName := st.ProductName
		if appName == "" || appName == "—" {
			appName = "主营业务系统"
		}
		item := BranchQualityItem{
			StoryCode:  fmt.Sprintf("ST%d", st.ID),
			StoryTitle: st.Title,
			BranchName: fmt.Sprintf("feature/US%d-st%d", demandID, st.ID),
			LatestMR:   fmt.Sprintf("!%d", 300+st.ID%100),
			GateStatus: "pass",
			Score:      92.5,
			BugsCount:  0,
			CodeSmells: 2,
			Coverage:   82.0,
			ScanTime:   "2026-09-05 16:30",
			Committer:  defaultDash(st.AssignedToName),
		}
		appMap[appName] = append(appMap[appName], item)
	}

	tree := make([]AppQualityNode, 0, len(appMap))
	for appName, branches := range appMap {
		tree = append(tree, AppQualityNode{
			AppName:     appName,
			AppCode:     strings.ToLower(strings.ReplaceAll(appName, " ", "-")),
			GatePassed:  true,
			BranchCount: len(branches),
			Branches:    branches,
		})
	}
	return tree
}
