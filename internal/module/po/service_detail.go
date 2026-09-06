// =============================================================================
// 文件: internal/module/po/service_detail.go
// 模块: PO 工作台
// 类型: service
// 职责: 业务需求统一详情（入口、摘要、父需求聚合与父子同级关系处理）。
// 依赖: gorm.io/gorm
// =============================================================================

package po

import (
	"context"
	"fmt"

	"workbench/internal/model"
)

// DetailService 业务需求详情服务。
type DetailService struct {
	repo *DemandDetailRepo
}

// NewDetailService 创建详情服务。
func NewDetailService(repo *DemandDetailRepo) *DetailService {
	return &DetailService{repo: repo}
}

// GetDemandDetail 组装业务需求详情完整结构。
func (s *DetailService) GetDemandDetail(ctx context.Context, actor *model.User, demandID uint) (*DemandDetailResp, error) {
	row, err := s.repo.FindDemandDetailByID(ctx, demandID)
	if err != nil {
		return nil, err
	}

	mode := "selfUnit"
	var childDemands []DemandChildRow
	var parentDemand *DemandDetailRow
	var siblings []DemandChildRow

	if row.Parent == 0 {
		childDemands, _ = s.repo.FindChildDemands(ctx, row.ID)
		if len(childDemands) > 0 {
			mode = "parentAggregate"
		}
	} else {
		mode = "childUnit"
		parentDemand, _ = s.repo.FindDemandDetailByID(ctx, row.Parent)
		siblings, _ = s.repo.FindChildDemands(ctx, row.Parent)
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
		resp.ParentAggregate = s.buildParentAggregate(ctx, childDemands)
	} else {
		if mode == "childUnit" && parentDemand != nil {
			resp.RelationContext = s.buildRelationContext(parentDemand, siblings, row.ID)
		}
		resp.ValueStream = s.buildValueStream(row)
		resp.Spotlight = s.buildSpotlight(row.Stage, row.Status)
		resp.Requirement = s.buildRequirement(ctx, row)
		resp.Execution = s.buildExecution(ctx, row.ID)
		resp.Delivery = s.buildDelivery(row, resp.Execution)
	}

	resp.History = s.buildHistory(ctx, row)

	return resp, nil
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

	return DemandSummary{
		ID:              code,
		Code:            code,
		DemandID:        row.ID,
		Title:           row.Name,
		Source:          defaultDash(row.Source),
		SourceNote:      defaultDash(row.SourceNote),
		Category:        defaultDash(row.Category),
		BSA:             defaultDash(row.BSA),
		Duration:        defaultDash(row.Duration),
		FeedbackedBy:    defaultDash(row.FeedbackedBy),
		ProposerName:    defaultDash(row.OriginatorName),
		ProposerDept:    defaultDash(row.OriginatorDept),
		Originator:      row.Originator,
		OwnerName:       defaultDash(row.BRAName),
		TestOwner:       defaultDash(row.QDName),
		Product:         defaultDash(row.ProductName),
		PoolName:        defaultDash(row.PoolName),
		Priority:        formatPriority(row.Pri),
		Status:          defaultDash(row.Status),
		ZentaoStatus:    defaultDash(row.Status),
		ValueStage:      stageKey,
		ValueStageLabel: stageLabel,
		EstimateLaunch:  launchStr,
		CreatedDate:     createdStr,
		EditedDate:      editedStr,
	}
}

func (s *DetailService) buildParentAggregate(ctx context.Context, children []DemandChildRow) *ParentAggregateData {
	total := len(children)
	online := 0
	risks := 0
	attention := make([]AttentionItem, 0)
	units := make([]DeliveryUnitItem, 0, total)

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

		// 检查是否有严重未闭环缺陷
		stories, _ := s.repo.FindDemandStories(ctx, c.ID)
		storyIDs := make([]uint, 0, len(stories))
		for _, st := range stories {
			storyIDs = append(storyIDs, st.ID)
		}
		bugMap, _ := s.repo.FindStoryBugCounts(ctx, storyIDs)
		taskMap, _ := s.repo.FindStoryTaskCounts(ctx, storyIDs)

		blockingBugs := 0
		tasksTotal := 0
		for _, b := range bugMap {
			blockingBugs += b.DeliveryBlocking
		}
		for _, t := range taskMap {
			tasksTotal += t.Total
		}

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
			StoriesNum: len(stories),
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
	}
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
