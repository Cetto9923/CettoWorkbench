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

// buildExecution 组装研发执行页签：研需进度、测试单、用例与缺陷概况。
// 取数顺序固定为研需 → 任务 → 缺陷 → 用例 → 测试单，任一查询失败即向上返回。
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
	testTasks, err := s.repo.FindStoryTestTasks(ctx, storyIDs)
	if err != nil {
		return nil, err
	}

	sItems, bugSummary := buildStoryItems(stories, taskMap, bugMap)
	ttItems, doingCount, doneCount := buildTestOrderItems(testTasks)
	testCaseSummary := buildTestCaseSummary(testCaseCounts)

	return &DetailExecution{
		Stories:    sItems,
		TestOrders: ttItems,
		TestOrderSummary: TestOrderSummary{
			TotalCount:   len(testTasks),
			DoingCount:   doingCount,
			DoneCount:    doneCount,
			StoriesCount: len(stories),
		},
		TestCaseSummary: testCaseSummary,
		BugSummary:      bugSummary,
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
			DevLeads: collectDevLeads(stories),
			TestLead: "—",
		},
	}, nil
}

// buildStoryItems 逐条研需合并任务与缺陷统计，并累加缺陷汇总。
// 缺陷总数只由命中 bugMap 的研需贡献，缺统计的按 0 计入明细。
func buildStoryItems(stories []DemandStoryRow, taskMap map[uint]TaskCountRow, bugMap map[uint]BugCountRow) ([]StoryItem, BugSummary) {
	items := make([]StoryItem, 0, len(stories))
	summary := BugSummary{}
	for _, st := range stories {
		tDone, tTotal := 0, 0
		if t, ok := taskMap[st.ID]; ok {
			tDone = t.Done
			tTotal = t.Total
		}
		sBugsTotal, sBugsActive := 0, 0
		if b, ok := bugMap[st.ID]; ok {
			summary.TotalCount += b.Total
			summary.ActiveCount += b.Active
			summary.ResolvedCount += b.Resolved
			summary.DeliveryBlocking += b.DeliveryBlocking
			sBugsTotal = b.Total
			sBugsActive = b.Active
		}

		items = append(items, StoryItem{
			ID:         st.ID,
			Code:       fmt.Sprintf("%d", st.ID),
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
	return items, summary
}

// buildTestOrderItems 展开测试单并统计进行中 / 已完成数量。
// 阶段按名称是否含 UAT 判定；缺开始或结束日期一律显示占位符。
func buildTestOrderItems(testTasks []DemandTestTaskRow) ([]TestOrderItem, int, int) {
	items := make([]TestOrderItem, 0, len(testTasks))
	doingCount, doneCount := 0, 0
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
		items = append(items, TestOrderItem{
			ID:          tt.ID,
			Code:        fmt.Sprintf("%d", tt.ID),
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
	return items, doingCount, doneCount
}

// buildTestCaseSummary 由用例聚合计数推导执行率与通过率，保留一位小数；
// 分母为 0 时保持 0，未执行数不为负。
func buildTestCaseSummary(counts TestCaseCountRow) TestCaseSummary {
	execRate := 0.0
	if counts.TotalCount > 0 {
		execRate = math.Round(float64(counts.ExecutedCount)/float64(counts.TotalCount)*1000) / 10
	}
	passRate := 0.0
	if counts.ExecutedCount > 0 {
		passRate = math.Round(float64(counts.PassedCount)/float64(counts.ExecutedCount)*1000) / 10
	}
	unexec := counts.TotalCount - counts.ExecutedCount
	if unexec < 0 {
		unexec = 0
	}
	return TestCaseSummary{
		TotalCount:      counts.TotalCount,
		ExecutedCount:   counts.ExecutedCount,
		UnexecutedCount: unexec,
		PassedCount:     counts.PassedCount,
		FailedCount:     counts.FailedCount,
		ExecutionRate:   execRate,
		PassRate:        passRate,
	}
}

// collectDevLeads 收集研需负责人去重列表；空值与占位符 "—" 不计入。
func collectDevLeads(stories []DemandStoryRow) []string {
	seen := make(map[string]struct{})
	for _, st := range stories {
		if st.AssignedToName != "" && st.AssignedToName != "—" {
			seen[st.AssignedToName] = struct{}{}
		}
	}
	leads := make([]string, 0, len(seen))
	for name := range seen {
		leads = append(leads, name)
	}
	return leads
}

// buildAppQualityTree F03：扫描系统未接入前返回空树，禁止拼造分支/MR/分数。
func buildAppQualityTree(stories []DemandStoryRow) []AppQualityNode {
	_ = stories
	return nil
}
