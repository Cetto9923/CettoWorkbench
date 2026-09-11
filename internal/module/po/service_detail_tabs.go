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

	"workbench/internal/pkg/zentao"
)

func (s *DetailService) buildValueStream(row *DemandDetailRow) *DetailValueStream {
	// F05：阶段集合与 mapValueStage 共用同一套 key，终态单独入列，禁止匹配失败回退澄清。
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
		{"closed", "已关闭", "系统"},
	}

	currentStage, _ := mapValueStage(row.Stage, row.Status)
	currentIdx := -1
	for idx, st := range stagesDef {
		if st.key == currentStage {
			currentIdx = idx
			break
		}
	}

	now := time.Now()
	items := make([]ValueStreamItem, 0, len(stagesDef)+1)
	usedDays := 0

	for idx, def := range stagesDef {
		item := ValueStreamItem{
			Key:   def.key,
			Label: def.label,
			Role:  def.role,
		}
		if currentIdx < 0 {
			item.Status = "future"
			item.DurationKind = "未知"
			item.DurationText = "时长未知"
		} else if idx < currentIdx {
			// F03：无阶段进入事件时不得伪造「实际 2 天」。
			item.Status = "done"
			item.DurationKind = "未知"
			item.DurationText = "时长未知"
		} else if idx == currentIdx {
			item.Status = "current"
			item.DurationKind = "已持续"
			if row.CreatedDate != nil {
				elapsed := int(math.Max(1, now.Sub(*row.CreatedDate).Hours()/24))
				item.DurationText = fmt.Sprintf("已持续约 %d 天（缺阶段进入时间，按创建日估算）", elapsed)
				usedDays += elapsed
			} else {
				item.DurationText = "时长未知"
			}
		} else {
			item.Status = "future"
			item.DurationKind = "预计"
			item.DurationText = "预计时长未知"
		}
		items = append(items, item)
	}

	if currentIdx < 0 {
		items = append(items, ValueStreamItem{
			Key:          "unknown",
			Label:        "未知阶段",
			Role:         "—",
			Status:       "current",
			DurationKind: "未知",
			DurationText: "状态未映射，请核对禅道 status/stage",
		})
	}

	estimatedDays := row.EstimateDelivery
	if estimatedDays <= 0 {
		estimatedDays = usedDays
	}
	targetDays := 28
	diff := estimatedDays - targetDays

	return &DetailValueStream{
		EstimatedCycleDays: estimatedDays,
		UsedCycleDays:      usedDays,
		TargetCycleDays:    targetDays,
		DiffCycleDays:      diff,
		IsOverdue:          estimatedDays > 0 && diff > 0,
		Stages:             items,
	}
}

func (s *DetailService) buildSpotlight(stage, status string) *DetailSpotlight {
	stageKey, _ := mapValueStage(stage, status)
	switch stageKey {
	case "accept":
		return &DetailSpotlight{
			Badge:         "待评审",
			Title:         "当前待办：业务需求评审",
			Desc:          "需求处于受理评审阶段，业务评审人出具结论后将进入澄清排期。",
			ActionLabel:   "查看评审",
			TargetTab:     "overview",
			TargetSection: "spotlightSection",
		}
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
			ActionLabel:   "排期",
			TargetTab:     "overview",
			TargetSection: "spotlightSection",
		}
	case "developing", "submittest":
		return &DetailSpotlight{
			Badge:         "待提测",
			Title:         "当前待办：提交测试，推动进入联调测试",
			Desc:          "需求已进入提测阶段。请确认关联研发需求与交付单元后发起提测；提测完成后将同步集成/验收测试单。",
			ActionLabel:   "去提测 →",
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

func (s *DetailService) buildRequirement(ctx context.Context, row *DemandDetailRow) (*DetailRequirement, error) {
	clarifies, err := s.repo.FindDemandClarifications(ctx, row.ID)
	if err != nil {
		return nil, err
	}
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

	files, err := s.repo.FindDemandFiles(ctx, row.ID)
	if err != nil {
		return nil, err
	}
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
		DemandID:       row.ID,
		SpecHtml:       row.Desc,
		VerifyHtml:     row.VerifyPlan,
		Clarifications: cItems,
		UserStories:    []UserStoryItem{},
		Attachments:    fItems,
		ClarifyZtURL:   zentao.DemandClarifyURL(row.ID),
	}, nil
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

func (s *DetailService) buildHistory(ctx context.Context, row *DemandDetailRow) (*DetailHistory, error) {
	actions, err := s.repo.FindDemandActions(ctx, row.ID)
	if err != nil {
		return nil, err
	}
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
			CreatedBy:      defaultDash(FormatAccountName(row.CreatedBy, row.CreatedByName)),
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
	}, nil
}

func mapValueStage(stage, status string) (string, string) {
	// F05：返回值必须落在 buildValueStream.stagesDef（含 closed）或 unknown；禁止默认回退 clarify。
	st := strings.ToLower(strings.TrimSpace(status))
	switch st {
	case "closed":
		return "closed", "已关闭"
	case "released":
		return "greyverify", "生产验证"
	case "waitdeliver", "delivered":
		return "publish", "发布"
	case "acceptanced":
		return "acceptance", "已验收"
	case "waitacceptance":
		return "acceptance", "待验收"
	case "testing":
		return "testing", "测试中"
	case "developing":
		// 与首页阶段卡「提测」对齐：禅道 developing 在价值流上落在提测节点。
		return "submittest", "提测"
	case "clarified":
		return "schedule", "已排期"
	case "active", "clarify":
		return "clarify", "澄清中"
	case "draft", "refuse", "wait":
		return "accept", "已受理"
	}

	sg := strings.ToLower(strings.TrimSpace(stage))
	switch sg {
	case "wait":
		return "accept", "已受理"
	case "inroadmap", "clarify":
		return "clarify", "澄清中"
	case "incharter", "schedule":
		return "schedule", "排期中"
	case "developing":
		return "submittest", "提测"
	case "delivering", "testing":
		return "testing", "测试中"
	case "delivered":
		return "publish", "发布"
	case "closed":
		return "closed", "已关闭"
	default:
		if st == "" && sg == "" {
			return "unknown", "未知"
		}
		if st != "" || sg != "" {
			return "unknown", "未知"
		}
		return "unknown", "未知"
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
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
