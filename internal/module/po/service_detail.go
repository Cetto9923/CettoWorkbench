// =============================================================================
// 文件: internal/module/po/service_detail.go
// 模块: PO 工作台
// 类型: service
// 职责: 业务需求统一详情（入口、摘要、父需求聚合与父子同级关系处理）。
//       F01 对象级授权与 F02 API 出口富文本净化集中在
//       service_detail_authz.go；本文件保留组装与摘要/聚合逻辑。
// 依赖: gorm.io/gorm
// =============================================================================

package po

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/module/po/primaryaction"
	"workbench/internal/pkg/personlabel"
	"workbench/internal/pkg/zentao"
)

// DetailService 业务需求详情服务。
type DetailService struct {
	repo   *DemandDetailRepo
	parent *Service // 用于复用 Service.DeriveDemandPrimaryActions 批量主操作派生
}

// NewDetailService 创建详情服务。
func NewDetailService(repo *DemandDetailRepo) *DetailService {
	return &DetailService{repo: repo}
}

// attachParent 注入父 Service 引用（仅 Service 构造时调用）。
//
// 用于详情行复用 Service 内已经聚合的 primaryaction.Derive 路径，
// 避免在 DetailService 重复构建 actor capability / 评价批量查询等逻辑。
func (s *DetailService) attachParent(parent *Service) {
	if s == nil {
		return
	}
	s.parent = parent
}

// GetDemandDetail 组装业务需求详情完整结构。
//
// F01：入口通过 GetDemandDetailAuthZ 做对象级授权与 not-found 区分；
// 父需求与子需求查询错误一律向上抛错，handler_detail.go 据此映射 500。
func (s *DetailService) GetDemandDetail(ctx context.Context, actor *model.User, demandID uint) (*DemandDetailResp, error) {
	row, err := s.GetDemandDetailAuthZ(ctx, actor, demandID)
	if err != nil {
		return nil, err
	}

	mode := "selfUnit"
	var childDemands []DemandChildRow
	var parentDemand *DemandDetailRow
	var siblings []DemandChildRow

	if row.Parent <= 0 {
		childDemands, err = s.repo.FindChildDemands(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		if len(childDemands) > 0 {
			mode = "parentAggregate"
		}
	} else {
		mode = "childUnit"
		parentDemand, err = s.repo.FindDemandDetailByID(ctx, uint(row.Parent))
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		siblings, err = s.repo.FindChildDemands(ctx, uint(row.Parent))
		if err != nil {
			return nil, err
		}
	}

	summary := s.buildSummary(row)

	resp := &DemandDetailResp{
		Success: true,
		Mode:    mode,
		Summary: summary,
		Config: DetailConfig{
			DeliveryCycleTargetDays: 28,
		},
	}

	if mode == "parentAggregate" {
		resp.ParentAggregate, err = s.buildParentAggregate(ctx, childDemands)
		if err != nil {
			return nil, err
		}
	} else {
		if mode == "childUnit" && parentDemand != nil {
			resp.RelationContext = s.buildRelationContext(parentDemand, siblings, row.ID)
		}
		resp.ValueStream = s.buildValueStream(row)
		resp.Spotlight = s.buildSpotlight(row.Stage, row.Status)
		reqTab, reqErr := s.buildRequirement(ctx, row)
		if reqErr != nil {
			return nil, reqErr
		}
		resp.Requirement = reqTab
		exec, execErr := s.buildExecution(ctx, row.ID)
		if execErr != nil {
			return nil, execErr
		}
		resp.Execution = exec
		resp.Delivery = s.buildDelivery(row, resp.Execution)

		if resp.Execution != nil {
			resp.Summary.StoriesCount = len(resp.Execution.Stories)
			for _, st := range resp.Execution.Stories {
				resp.Summary.TasksDone += st.TasksDone
				resp.Summary.TasksTotal += st.TasksTotal
			}
			resp.Summary.CasesExecuted = resp.Execution.TestCaseSummary.ExecutedCount
			resp.Summary.CasesTotal = resp.Execution.TestCaseSummary.TotalCount
			resp.Summary.BugsUnresolved = resp.Execution.BugSummary.ActiveCount
		}
	}

	history, histErr := s.buildHistory(ctx, row)
	if histErr != nil {
		return nil, histErr
	}
	resp.History = history

	// 单行详情主操作派生。
	// 详情行已包含 stage/status/accepter/assignedTo，无需 IN 批量。
	if actor != nil && row != nil {
		pa := s.buildPrimaryActionForDetail(ctx, actor, row)
		// 提测改走四步弹窗：详情 JSON 不再下发旧整页 URL，避免任何入口误跳 submit_test.html。
		if pa.Key == string(primaryaction.KeySubmitTest) {
			pa.URL = ""
		}
		resp.PrimaryAction = &pa
		bindPrimaryActionSpotlight(resp.Spotlight, pa)
		account := strings.TrimSpace(actor.Account)
		if account != "" {
			resp.Summary.IsCreator = strings.TrimSpace(row.CreatedBy) == account
			resp.Summary.IsAssignee = strings.TrimSpace(row.AssignedTo) == account
		}
		resp.Summary.CanWithdrawReview = canWithdrawReviewForDetail(actor, row)
		if pa.Key == string(primaryaction.KeyApprove) && pa.Enabled {
			resp.Summary.CanReview = true
		}
		s.populateDemandEditability(ctx, actor, row, &resp.Summary)
	}

	// F02：API 出口对富文本字段做白名单净化，确保 specHtml / verifyHtml 即便
	// 未来绕过 renderer 也不会带 script/on*/javascript: 等危险形态。
	if resp.Requirement != nil {
		resp.Requirement.SpecHtml = SanitizeRichTextHTML(resp.Requirement.SpecHtml)
		resp.Requirement.VerifyHtml = SanitizeRichTextHTML(resp.Requirement.VerifyHtml)
	}
	resp.Summary.Desc = SanitizeRichTextHTML(resp.Summary.Desc)
	resp.Summary.VerifyPlan = SanitizeRichTextHTML(resp.Summary.VerifyPlan)

	return resp, nil
}

// bindPrimaryActionSpotlight 将列表同源的主操作挂到详情 spotlight。
// 排期：挂站内 URL（跳转排期页）。
// 提测：只刷新文案，不挂 /submit-test 旧整页 URL——前端应打开四步弹窗（与首页一致）。
func bindPrimaryActionSpotlight(spotlight *DetailSpotlight, action primaryaction.PrimaryAction) {
	if spotlight == nil || !action.Enabled {
		return
	}
	label := strings.TrimSpace(action.Label)
	switch action.Key {
	case string(primaryaction.KeySchedule):
		if label == "" || strings.TrimSpace(action.URL) == "" {
			return
		}
		spotlight.ActionLabel = label
		spotlight.ActionURL = strings.TrimSpace(action.URL)
	case string(primaryaction.KeySubmitTest):
		if label == "" {
			return
		}
		spotlight.ActionLabel = label
		// ActionURL 留空：详情打开提测弹窗，不链到旧整页。
		spotlight.ActionURL = ""
	case string(primaryaction.KeyWithdrawReview):
		spotlight.ActionLabel = "撤销评审"
		spotlight.ActionURL = ""
	default:
		return
	}
}

func canWithdrawReviewForDetail(actor *model.User, row *DemandDetailRow) bool {
	if actor == nil || row == nil {
		return false
	}
	if strings.TrimSpace(row.Status) != "wait" {
		return false
	}
	account := strings.TrimSpace(actor.Account)
	if account == "" {
		return false
	}
	return actor.IsSuperAdmin || strings.TrimSpace(row.CreatedBy) == account
}

func (s *DetailService) buildSummary(row *DemandDetailRow) DemandSummary {
	code := fmt.Sprintf("US%d", row.ID)
	stageKey, stageLabel := mapValueStage(row.Stage, row.Status)
	createdStr := ""
	if row.CreatedDate != nil {
		createdStr = row.CreatedDate.Format("2006-01-02 15:04")
	}
	editedStr := createdStr
	if row.EditedDate != nil {
		editedStr = row.EditedDate.Format("2006-01-02 15:04")
	}
	launchStr := "—"
	if row.EstimateLaunch != nil {
		launchStr = row.EstimateLaunch.Format("2006-01-02")
	}
	devFinishStr := "—"
	if row.DevelopFinish != nil {
		devFinishStr = row.DevelopFinish.Format("2006-01-02")
	}
	testFinishStr := "—"
	if row.TestFinish != nil {
		testFinishStr = row.TestFinish.Format("2006-01-02")
	}
	verifyFinishStr := "—"
	if row.VerifyFinish != nil {
		verifyFinishStr = row.VerifyFinish.Format("2006-01-02")
	}
	prodName := defaultDash(row.ProductName)
	if prodName == "—" && row.MainSystemName != "" && row.MainSystemName != "—" {
		prodName = row.MainSystemName
	}
	acceptStatus := "—"
	if row.VerifyFinish != nil || row.Status == "acceptanced" || row.Status == "waitdeliver" || row.Status == "released" {
		acceptStatus = "待发起"
	}

	return DemandSummary{
		ID:               code,
		Code:             code,
		DemandID:         row.ID,
		Title:            row.Name,
		Source:           defaultDash(row.Source),
		SourceNote:       defaultDash(row.SourceNote),
		Category:         defaultDash(row.Category),
		BSA:              defaultDash(row.BSA),
		Duration:         defaultDash(row.Duration),
		FeedbackedBy:     defaultDash(row.FeedbackedBy),
		ProposerName:     defaultDash(row.OriginatorName),
		ProposerDept:     defaultDash(row.OriginatorDept),
		Originator:       row.Originator,
		OwnerName:        defaultDash(row.BRAName),
		TestOwner:        defaultDash(row.QDName),
		AcceptOwner:      defaultDash(firstNonEmpty(row.AccepterName, row.Accepter)),
		Reviewer:         defaultDash(firstNonEmpty(row.ReviewerName, row.Reviewer)),
		CurrentOwner:     defaultDash(firstNonEmpty(row.AssignedToName, row.AssignedTo)),
		Product:          prodName,
		MainSystem:       defaultDash(row.MainSystem),
		MainSystemName:   defaultDash(firstNonEmpty(row.MainSystemName, row.MainSystem)),
		PoolName:         defaultDash(row.PoolName),
		Priority:         formatPriority(row.Pri),
		Status:           defaultDash(row.Status),
		ZentaoStatus:     defaultDash(row.Status),
		ValueStage:       stageKey,
		ValueStageLabel:  stageLabel,
		EstimateLaunch:   launchStr,
		DevelopFinish:    devFinishStr,
		TestFinish:       testFinishStr,
		VerifyFinish:     verifyFinishStr,
		Desc:             row.Desc,
		VerifyPlan:       row.VerifyPlan,
		EstimateDelivery: row.EstimateDelivery,
		AcceptanceStatus: acceptStatus,
		CreatedDate:      createdStr,
		EditedDate:       editedStr,
		CreatedBy:        row.CreatedBy,
		CreatedName:      personlabel.Format(row.CreatedBy, row.CreatedByName),
		ZentaoEditURL:    zentao.DemandEditURL(row.ID),
		ZentaoURL:        zentao.DemandViewURL(row.ID),
	}
}

func (s *DetailService) buildParentAggregate(ctx context.Context, children []DemandChildRow) (*ParentAggregateData, error) {
	total := len(children)
	online := 0
	risks := 0
	attention := make([]AttentionItem, 0)
	units := make([]DeliveryUnitItem, 0, total)

	ids := make([]uint, 0, len(children))
	for _, child := range children {
		ids = append(ids, child.ID)
	}
	counts, err := s.repo.FindChildExecutionCounts(ctx, ids)
	if err != nil {
		return nil, err
	}

	for _, c := range children {
		code := fmt.Sprintf("US%d", c.ID)
		isDone := c.Status == "closed" || c.Stage == "delivered" || c.Status == "delivered"
		if isDone {
			online++
		}

		launch := "—"
		if c.EstimateLaunch != nil {
			launch = c.EstimateLaunch.Format("2006-01-02")
		}

		_, stageLabel := mapValueStage(c.Stage, c.Status)

		count := counts[c.ID]
		blockingBugs := count.BlockingBugs
		tasksTotal := count.Tasks

		hasRisk := blockingBugs > 0
		if hasRisk {
			risks++
			attention = append(attention, AttentionItem{
				DemandID: c.ID,
				Code:     code,
				Title:    c.Name,
				Stage:    stageLabel,
				Owner:    defaultDash(c.AssignedToName),
				RiskDesc: fmt.Sprintf("存在 %d 个影响交付的缺陷", blockingBugs),
				IsRisk:   true,
			})
		} else if !isDone && (c.Stage == "testing" || c.Stage == "developing") {
			attention = append(attention, AttentionItem{
				DemandID: c.ID,
				Code:     code,
				Title:    c.Name,
				Stage:    stageLabel,
				Owner:    defaultDash(c.AssignedToName),
				RiskDesc: fmt.Sprintf("处于%s阶段，计划上线 %s", stageLabel, launch),
				IsRisk:   false,
			})
		}

		units = append(units, DeliveryUnitItem{
			DemandID:   c.ID,
			Code:       code,
			Title:      c.Name,
			Stage:      stageLabel,
			Owner:      defaultDash(c.AssignedToName),
			StoriesNum: count.Stories,
			TasksNum:   tasksTotal,
			LaunchDate: launch,
			IsDone:     isDone,
			HasRisk:    hasRisk,
		})
	}

	return &ParentAggregateData{
		UnitTotal:      total,
		UnitOnline:     online,
		RisksCount:     risks,
		AttentionItems: attention,
		DeliveryUnits:  units,
	}, nil
}

func (s *DetailService) buildRelationContext(parent *DemandDetailRow, siblings []DemandChildRow, currentID uint) *RelationContextData {
	pCode := fmt.Sprintf("US%d", parent.ID)
	_, pStageLabel := mapValueStage(parent.Stage, parent.Status)
	pDemand := &RelationDemand{
		DemandID:  parent.ID,
		Code:      pCode,
		Title:     parent.Name,
		Stage:     pStageLabel,
		Owner:     defaultDash(parent.AssignedToName),
		IsCurrent: false,
	}

	sList := make([]RelationDemand, 0, len(siblings))
	for _, sib := range siblings {
		sCode := fmt.Sprintf("US%d", sib.ID)
		_, sStage := mapValueStage(sib.Stage, sib.Status)
		launch := "—"
		if sib.EstimateLaunch != nil {
			launch = sib.EstimateLaunch.Format("2006-01-02")
		}
		isDone := sib.Status == "closed" || sib.Stage == "delivered"
		sList = append(sList, RelationDemand{
			DemandID:   sib.ID,
			Code:       sCode,
			Title:      sib.Name,
			Stage:      sStage,
			Owner:      defaultDash(sib.AssignedToName),
			LaunchDate: launch,
			IsCurrent:  sib.ID == currentID,
			IsDone:     isDone,
		})
	}

	return &RelationContextData{
		Parent:   pDemand,
		Siblings: sList,
	}
}

// buildPrimaryActionForDetail 详情行主操作。
//
// 复用 Service 内聚合函数 DeriveDemandPrimaryActions；单行 ID 走同样的批量
// 路径，避免新增"单行特化"代码分支导致与批量派生结果不一致。
//
// 测试单与评价事实走 IN (?) 批量查询（即便 ID 数量为 1 也复用同一函数），保证
//
//	primaryAction.test_link / evaluate / view_evaluate 与列表页完全一致。
func (s *DetailService) buildPrimaryActionForDetail(
	ctx context.Context,
	actor *model.User,
	row *DemandDetailRow,
) primaryaction.PrimaryAction {
	if row == nil || row.ID == 0 {
		return primaryaction.None()
	}
	svc := s.parentService()
	if svc == nil {
		return primaryaction.None()
	}
	out, err := svc.DeriveDemandPrimaryActions(ctx, actor, []uint{row.ID})
	if err != nil {
		return primaryaction.None()
	}
	if pa, ok := out[row.ID]; ok {
		return pa
	}
	return primaryaction.None()
}

// parentService 返回 Service 提供的父服务（如未注入则返回 nil）。
// service_detail.go 内不直接持有 Service 指针；构造时由 Service.NewService 注入。
func (s *DetailService) parentService() *Service {
	if s == nil {
		return nil
	}
	return s.parent
}
