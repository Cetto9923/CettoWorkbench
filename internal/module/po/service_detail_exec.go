// =============================================================================
// 文件: internal/module/po/service_detail_exec.go
// 模块: PO 工作台
// 类型: service
// 职责: 业务需求 Tab 3 研发执行、测试进度与代码质量概况组装。
//       F03：未接入扫描源时不返回虚构 MR / 分数 / 通过门禁。
//       F04：Repo 查询错误向上传递，禁止吞错后冒充健康零值。
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

func (s *DetailService) buildExecution(ctx context.Context, demandID uint) (*DetailExecution, error) {
	stories, err := s.repo.FindDemandStories(ctx, demandID)
	if err != nil {
		return nil, err
	}
	storyIDs := make([]uint, 0, len(stories))
	for _, st := range stories {
		storyIDs = append(storyIDs, st.ID)
	}

	taskMap, err := s.repo.FindStoryTaskCounts(ctx, storyIDs)
	if err != nil {
		return nil, err
	}
	bugMap, err := s.repo.FindStoryBugCounts(ctx, storyIDs)
	if err != nil {
		return nil, err
	}
	testCaseCounts, err := s.repo.FindStoryTestCaseCounts(ctx, storyIDs)
	if err != nil {
		return nil, err
	}

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

	testTasks, err := s.repo.FindStoryTestTasks(ctx, storyIDs)
	if err != nil {
		return nil, err
	}
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
			Available:     false,
			Source:        "none",
			AvgScore:      0,
			BranchesCount: 0,
			PassedGates:   0,
			TotalGates:    0,
		},
		QualityOverview: QualityGateOverview{
			Available:     false,
			Source:        "none",
			AppsCount:     0,
			BranchesCount: 0,
			PassedGates:   0,
			FailedGates:   0,
		},
		// F03：扫描未接入 → 空树；前端依据 Available=false 展示「未接入」。
		AppQualityTree: buildAppQualityTree(stories),
		LeadsMatrix: LeadsMatrix{
			DevLeads: devLeads,
			TestLead: "—",
		},
	}, nil
}

// buildAppQualityTree F03：扫描系统未接入前返回空树，禁止拼造分支/MR/分数。
func buildAppQualityTree(stories []DemandStoryRow) []AppQualityNode {
	_ = stories
	return nil
}
