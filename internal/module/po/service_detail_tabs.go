// =============================================================================
// 文件: internal/module/po/service_detail_tabs.go
// 模块: PO 工作台
// 类型: service
// 职责: 业务需求五大 Tab（概览流、澄清、研发执行、交付、过程记录）数据组装。
// 依赖: gorm.io/gorm
// =============================================================================

package po

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

func (s *DetailService) buildValueStream(row *DemandDetailRow) *DetailValueStream {
	stagesDef := []struct {
		key   string
		label string
		role  string
	}{
		{"accept", "受理", "PO"},
		{"clarify", "澄清", "需求分析 / PO"},
		{"schedule", "排期", "排期协同"},
		{"developing", "研发", "敏捷研发团队"},
		{"submittest", "提测", "研发 / 测试协同"},
		{"testing", "测试", "测试团队"},
		{"acceptance", "验收", "业务部门 / PO"},
		{"publish", "发布", "发布组"},
		{"greyverify", "生产验证", "业务 / 运维"},
	}

	currentStage, _ := mapValueStage(row.Stage, row.Status)
	currentIdx := 1
	for idx, st := range stagesDef {
		if st.key == currentStage {
			currentIdx = idx
			break
		}
	}

	now := time.Now()
	items := make([]ValueStreamItem, 0, len(stagesDef))
	usedDays := 0

	for idx, def := range stagesDef {
		item := ValueStreamItem{
			Key:   def.key,
			Label: def.label,
			Role:  def.role,
		}
		if idx < currentIdx {
			item.Status = "done"
			item.DurationKind = "实际"
			item.DurationText = "实际 2 天"
			usedDays += 2
		} else if idx == currentIdx {
			item.Status = "current"
			item.DurationKind = "已持续"
			elapsed := 3
			if row.CreatedDate != nil {
				elapsed = int(math.Max(1, now.Sub(*row.CreatedDate).Hours()/24))
			}
			item.DurationText = fmt.Sprintf("已持续 %d 天", elapsed)
			usedDays += elapsed
		} else {
			item.Status = "future"
			item.DurationKind = "预计"
			item.DurationText = "预计 3 天"
		}
		items = append(items, item)
	}

	estimatedDays := usedDays + (len(stagesDef)-currentIdx-1)*3
	targetDays := 28
	diff := estimatedDays - targetDays

	return &DetailValueStream{
		EstimatedCycleDays: estimatedDays,
		UsedCycleDays:      usedDays,
		TargetCycleDays:    targetDays,
		DiffCycleDays:      diff,
		IsOverdue:          diff > 0,
		Stages:             items,
	}
}

func (s *DetailService) buildSpotlight(stage, status string) *DetailSpotlight {
	stageKey, _ := mapValueStage(stage, status)
	switch stageKey {
	case "clarify":
		return &DetailSpotlight{
			Badge:         "待你处理",
			Title:         "当前待办：完成需求澄清并推动进入排期",
			Desc:          "需求已通过初步评审，请与需求分析师确认产品范围、系统改动点与验收标准。",
			ActionLabel:   "进入澄清办理区 →",
			TargetTab:     "requirement",
			TargetSection: "clarificationSection",
		}
	case "schedule":
		return &DetailSpotlight{
			Badge:         "排期中",
			Title:         "当前阶段：版本窗口规划与研发排期协同",
			Desc:          "请确认所属版本发布窗口，组织敏捷小组进行工时评估与依赖对齐。",
			ActionLabel:   "进入排期工作台 →",
			TargetTab:     "overview",
			TargetSection: "spotlightSection",
		}
	case "developing":
		return &DetailSpotlight{
			Badge:         "研发中",
			Title:         "当前执行：研发需求与任务开发进行中",
			Desc:          "研发人员正在执行具体开发任务，请关注任务完成率与阻塞问题。",
			ActionLabel:   "查看研发执行 →",
			TargetTab:     "execution",
			TargetSection: "storiesSection",
		}
	case "testing":
		return &DetailSpotlight{
			Badge:         "测试中",
			Title:         "当前执行：集成测试与缺陷排查",
			Desc:          "测试用例正在执行中，请重点排查影响交付的严重缺陷。",
			ActionLabel:   "查看用例与缺陷 →",
			TargetTab:     "execution",
			TargetSection: "bugSection",
		}
	case "acceptance":
		return &DetailSpotlight{
			Badge:         "待验收",
			Title:         "当前待办：业务部门确认与交付验收",
			Desc:          "研发提测已达标，请组织业务验收并确认是否满足交付上线条件。",
			ActionLabel:   "进入验收交付 →",
			TargetTab:     "delivery",
			TargetSection: "deliverySection",
		}
	default:
		return &DetailSpotlight{
			Badge:         "持续跟踪",
			Title:         "需求正常推进中",
			Desc:          "请关注各环节交付节奏与生命周期更新。",
			ActionLabel:   "查看流转记录 →",
			TargetTab:     "history",
			TargetSection: "historySection",
		}
	}
}

func (s *DetailService) buildRequirement(ctx context.Context, row *DemandDetailRow) *DetailRequirement {
	clarifies, _ := s.repo.FindDemandClarifications(ctx, row.ID)
	cItems := make([]ClarificationItem, 0, len(clarifies))
	for _, c := range clarifies {
		devEnd := "—"
		if c.DevEnd != nil {
			devEnd = c.DevEnd.Format("2006-01-02")
		}
		testEnd := "—"
		if c.TestEnd != nil {
			testEnd = c.TestEnd.Format("2006-01-02")
		}
		cItems = append(cItems, ClarificationItem{
			Product:     c.Product,
			ProductName: defaultDash(c.ProductName),
			Analyst:     defaultDash(c.PM),
			Content:     c.SystemClarifyDesc,
			DevEnd:      devEnd,
			TestEnd:     testEnd,
		})
	}

	files, _ := s.repo.FindDemandFiles(ctx, row.ID)
	fItems := make([]AttachmentItem, 0, len(files))
	for _, f := range files {
		created := "—"
		if f.AddedDate != nil {
			created = f.AddedDate.Format("2006-01-02")
		}
		sizeStr := fmt.Sprintf("%.1f KB", float64(f.Size)/1024.0)
		fItems = append(fItems, AttachmentItem{
			ID:       f.ID,
			Title:    f.Title,
			Size:     sizeStr,
			Created:  created,
			Download: fmt.Sprintf("/file/download/%d", f.ID),
		})
	}

	return &DetailRequirement{
		SpecHtml:       row.Desc,
		VerifyHtml:     row.VerifyPlan,
		Clarifications: cItems,
		UserStories:    []UserStoryItem{},
		Attachments:    fItems,
	}
}

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
		if b, ok := bugMap[st.ID]; ok {
			totalBugs += b.Total
			activeBugs += b.Active
			resolvedBugs += b.Resolved
			blockingBugs += b.DeliveryBlocking
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

	return &DetailExecution{
		Stories: sItems,
		TestCaseSummary: TestCaseSummary{
			TotalCount:    testCaseCounts.TotalCount,
			ExecutedCount: testCaseCounts.ExecutedCount,
			ExecutionRate: execRate,
			PassedCount:   testCaseCounts.PassedCount,
			PassRate:      passRate,
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
	}
}

func (s *DetailService) buildDelivery(row *DemandDetailRow, exec *DetailExecution) *DetailDelivery {
	devDone := false
	if exec != nil && len(exec.Stories) > 0 {
		allDone := true
		for _, st := range exec.Stories {
			if st.TasksTotal > 0 && st.TasksDone < st.TasksTotal {
				allDone = false
				break
			}
		}
		devDone = allDone
	}

	testPassed := false
	bugsResolved := true
	if exec != nil {
		testPassed = exec.TestCaseSummary.PassRate >= 90.0
		bugsResolved = exec.BugSummary.DeliveryBlocking == 0
	}

	acceptanceDone := row.Status == "closed" || row.Stage == "delivered"

	launchStr := "—"
	if row.EstimateLaunch != nil {
		launchStr = row.EstimateLaunch.Format("2006-01-02")
	}
	windowStr := "—"
	if row.PublishWindow != nil {
		windowStr = row.PublishWindow.Format("2006-01-02")
	}

	return &DetailDelivery{
		DevDone:          devDone,
		TestPassed:       testPassed,
		BugsResolved:     bugsResolved,
		AcceptanceDone:   acceptanceDone,
		EstimateLaunch:   launchStr,
		PublishWindow:    windowStr,
		VerifyConclusion: defaultDash(row.VerifyPlan),
	}
}

func (s *DetailService) buildHistory(ctx context.Context, row *DemandDetailRow) *DetailHistory {
	actions, _ := s.repo.FindDemandActions(ctx, row.ID)
	actItems := make([]ActionHistoryItem, 0, len(actions))
	for _, a := range actions {
		dStr := "—"
		if a.Date != nil {
			dStr = a.Date.Format("2006-01-02 15:04")
		}
		actItems = append(actItems, ActionHistoryItem{
			Date:   dStr,
			Actor:  defaultDash(a.ActorName),
			Action: defaultDash(a.Action),
			Extra:  defaultDash(a.Extra),
		})
	}

	createdStr := "—"
	if row.CreatedDate != nil {
		createdStr = row.CreatedDate.Format("2006-01-02")
	}
	reviewedStr := "—"
	if row.ReviewedDate != nil {
		reviewedStr = row.ReviewedDate.Format("2006-01-02")
	}
	editedStr := "—"
	if row.EditedDate != nil {
		editedStr = row.EditedDate.Format("2006-01-02")
	}
	closedStr := "—"
	if row.ClosedDate != nil {
		closedStr = row.ClosedDate.Format("2006-01-02")
	}

	return &DetailHistory{
		Actions:        actItems,
		StageDurations: []StageDurationItem{},
		Lifecycle: DemandLifecycle{
			CreatedBy:      defaultDash(row.CreatedBy),
			CreatedDate:    createdStr,
			AssignedTo:     defaultDash(row.AssignedToName),
			AssignedDate:   createdStr,
			Reviewer:       defaultDash(row.Reviewer),
			ReviewedDate:   reviewedStr,
			LastEditedBy:   defaultDash(row.EditedBy),
			LastEditedDate: editedStr,
			ClosedBy:       defaultDash(row.ClosedBy),
			ClosedDate:     closedStr,
			ClosedReason:   defaultDash(row.ClosedReason),
		},
	}
}

func mapValueStage(stage, status string) (string, string) {
	st := strings.ToLower(strings.TrimSpace(stage))
	switch st {
	case "wait":
		return "accept", "已受理"
	case "inroadmap", "clarify":
		return "clarify", "澄清中"
	case "incharter", "schedule":
		return "schedule", "排期中"
	case "developing":
		return "developing", "研发中"
	case "delivering", "testing":
		return "testing", "测试中"
	case "delivered":
		return "delivered", "上线完成"
	case "closed":
		return "closed", "已关闭"
	default:
		s := strings.ToLower(strings.TrimSpace(status))
		if s == "clarify" {
			return "clarify", "澄清中"
		}
		return "clarify", "澄清中"
	}
}

func defaultDash(v string) string {
	val := strings.TrimSpace(v)
	if val == "" {
		return "—"
	}
	return val
}

func formatPriority(pri string) string {
	p := strings.TrimSpace(pri)
	if p == "" {
		return "P2"
	}
	if strings.HasPrefix(strings.ToUpper(p), "P") {
		return strings.ToUpper(p)
	}
	return "P" + p
}
