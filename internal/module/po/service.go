// =============================================================================
// 文件: internal/module/po/service.go
// 模块: PO 工作台
// 类型: action
// 职责: 组装 PO 首页价值流统计与需求列表（各阶段读只读备库，「全部」为其余阶段去重总和；排期窗口走主库 schedule）。
// 依赖: internal/model
//       internal/module/schedule
//       internal/module/user
//       internal/pkg/zentao
// =============================================================================

package po

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"workbench/internal/model"
	"workbench/internal/module/schedule"
	"workbench/internal/module/user"
	"workbench/internal/pkg/zentao"
)

var valueStreamStages = []struct {
	label  string
	status string
}{
	{label: "全部", status: "all"},
	{label: "受理", status: "accept"},
	{label: "澄清", status: "clarify"},
	{label: "排期", status: "schedule"},
	{label: "提测", status: "developing"},
	{label: "联调测试", status: "testing"},
	{label: "验收", status: "waitacceptance"},
	{label: "发起交付", status: "acceptanced"},
	{label: "发布", status: "publish"},
	{label: "评价反馈", status: "released"},
}

// valueStreamLabel 是阶段中文名的唯一出口：按编码查首页价值流阶段表。
// closed 不在首页表内（首页无终态阶段），单独补「已关闭」；其余未收录编码显示「未知」。
func valueStreamLabel(code string) string {
	for _, def := range valueStreamStages[1:] {
		if def.status == code {
			return def.label
		}
	}
	if code == "closed" {
		return "已关闭"
	}
	return "未知"
}

// Service PO 工作台业务逻辑。
type Service struct {
	repo         *Repo
	detailSvc    *DetailService
	schedule     *schedule.Service
	userSvc      *user.Service
	logger       *zap.Logger
	issueActions zentao.IssueActionGateway
	taskActions  taskStatusGateway
	ztAPI        *zentao.Client
}

// NewService 创建 Service。兼容 4 参数 (repo, schedule, user, logger) 与 5 参数 (repo, schedule, user, ztAPI, logger)。
func NewService(repo *Repo, scheduleSvc *schedule.Service, userSvc *user.Service, args ...any) *Service {
	var logger *zap.Logger
	client := zentao.DefaultClient()
	for _, arg := range args {
		switch value := arg.(type) {
		case *zap.Logger:
			logger = value
		case *zentao.Client:
			client = value
		}
	}
	var detailSvc *DetailService
	if repo != nil && repo.db != nil {
		detailSvc = NewDetailService(NewDemandDetailRepo(repo.db))
	}
	s := &Service{repo: repo, detailSvc: detailSvc, schedule: scheduleSvc, userSvc: userSvc, logger: logger,
		issueActions: zentao.NewUnavailableIssueActionGateway("当前禅道 API 未提供问题解决、关闭或重新激活动作接口"),
		taskActions:  client, ztAPI: client}
	if detailSvc != nil {
		detailSvc.attachParent(s)
	}
	return s
}

type taskStatusGateway interface {
	UpdateTask(ctx context.Context, p zentao.UpdateTaskParams) error
}

// SetTaskStatusGateway 注入禅道任务状态网关，供测试替换。
func (s *Service) SetTaskStatusGateway(gateway taskStatusGateway) {
	if s == nil {
		return
	}
	s.taskActions = gateway
}

// DetailService 返回统一详情服务。
func (s *Service) DetailService() *DetailService {
	if s == nil {
		return nil
	}
	return s.detailSvc
}

// CountValueStreamAll 首页价值流「全部」条数（业需 + 研需 kind+id 去重并集），供看板等模块调用。
func (s *Service) CountValueStreamAll(ctx context.Context, account string) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, nil
	}
	stages, err := s.countAllStageBreakdown(ctx, account)
	if err != nil {
		return 0, err
	}
	if len(stages) > 0 && stages[0].Status == "all" {
		return stages[0].Count, nil
	}
	var total int64
	for _, st := range stages {
		total += st.Count
	}
	return total, nil
}

// Home 加载首页价值流阶段统计。
func (s *Service) Home(ctx context.Context, actor *model.User) (*HomeResp, error) {
	account := ""
	if actor != nil {
		account = actor.Account
	}

	if s.repo == nil {
		return nil, fmt.Errorf("po repo is not configured")
	}

	t0 := time.Now()
	var (
		breakdown    []ValueStreamStage
		allErr       error
		windows      []schedule.HomeVersionWindowCard
		winErr       error
		myPending    int64
		pendingErr   error
		overdue      int64
		overdueErr   error
		ownership    KPICounts
		ownershipErr error
		kpiSummary   KPISummaryResult
		kpiErr       error
		wg           sync.WaitGroup
	)

	// 并行加载价值流全景统计、版本窗口与 KPI 指标，大幅缩短首屏加载耗时。
	runQuerySafely(&wg, &allErr, func() {
		breakdown, allErr = s.countAllStageBreakdown(ctx, account)
	})

	if s.schedule != nil {
		runQuerySafely(&wg, &winErr, func() {
			relatedIDs, err := s.repo.FindHomeRelatedDemandIDs(ctx, account)
			if err != nil {
				winErr = err
				return
			}
			windows, winErr = s.schedule.ListHomeVersionWindows(ctx, actor, relatedIDs...)
		})
	}

	if strings.TrimSpace(account) != "" {
		runQuerySafely(&wg, &ownershipErr, func() { ownership, ownershipErr = s.countHomeOwnership(ctx, account) })
		runQuerySafely(&wg, &pendingErr, func() {
			myPending, pendingErr = s.repo.CountHomeFocus(ctx, account, DemandsReq{Status: "all", Focus: "my_action"})
		})
		runQuerySafely(&wg, &overdueErr, func() {
			overdue, overdueErr = s.repo.CountHomeFocus(ctx, account, DemandsReq{Status: "all", Focus: "overdue"})
		})
		runQuerySafely(&wg, &kpiErr, func() {
			kpiSummary, kpiErr = s.repo.CountKPISummary(ctx, account)
		})
	}

	wg.Wait()

	if allErr != nil {
		return nil, allErr
	}
	if pendingErr != nil {
		return nil, pendingErr
	}
	if overdueErr != nil {
		return nil, overdueErr
	}
	if ownershipErr != nil {
		return nil, ownershipErr
	}
	if kpiErr != nil {
		return nil, kpiErr
	}

	stages := make([]ValueStreamStage, len(breakdown))
	copy(stages, breakdown)
	allIdx := -1
	for i, st := range stages {
		if st.Status == "all" {
			allIdx = i
			break
		}
	}
	if allIdx >= 0 {
		var allDemand, allStory int64
		for i, stage := range stages {
			if i == allIdx {
				continue
			}
			allDemand += stage.DemandCount
			allStory += stage.StoryCount
		}
		stages[allIdx].DemandCount = allDemand
		stages[allIdx].StoryCount = allStory
		stages[allIdx].Count = allDemand + allStory
	}

	versionWindows := []schedule.HomeVersionWindowCard{}
	versionWindowsError := ""
	if s.schedule == nil {
		versionWindowsError = "排期服务不可用"
		if s.logger != nil {
			s.logger.Error("po home schedule service is nil, version windows skipped")
		}
	} else if winErr != nil {
		versionWindowsError = "版本窗口查询失败"
		if s.logger != nil {
			s.logger.Warn("po home version windows", zap.Error(winErr))
		}
	} else {
		versionWindows = windows
	}

	// 全部与待我处理是两个独立口径。
	allCount := int64(0)
	if allIdx >= 0 {
		allCount = stages[allIdx].Count
	}
	kpi := KPICounts{
		Today:     kpiSummary.Today,
		Overdue:   overdue,
		Suspended: kpiSummary.Suspended,
		Blocked:   kpiSummary.Blocked,
		MyPending: myPending,
		MyManaged: ownership.MyManaged,
		MyRelated: ownership.MyRelated,
	}

	if s.logger != nil {
		s.logger.Info("po home kpi",
			zap.String("account", account),
			zap.Int64("today", kpi.Today),
			zap.Int64("overdue", kpi.Overdue),
			zap.Int64("suspended", kpi.Suspended),
			zap.Int64("blocked", kpi.Blocked),
			zap.Int64("my_pending", kpi.MyPending),
		)
		s.logger.Info("po home parallel load completed",
			zap.Duration("total_duration", time.Since(t0)),
		)
	}

	return &HomeResp{
		AllCount:            allCount,
		Stages:              stages,
		StagesValid:         true,
		VersionWindows:      versionWindows,
		VersionWindowsError: versionWindowsError,
		KPI:                 kpi,
	}, nil
}

// TeamHomeVersionWindows returns upcoming window summaries for authorized agile-group IDs.
func (s *Service) TeamHomeVersionWindows(ctx context.Context, groupIDs []uint, limit int) ([]schedule.TeamHomeVersionWindowCard, error) {
	if s == nil || s.schedule == nil {
		return nil, fmt.Errorf("schedule service is not configured")
	}
	return s.schedule.ListTeamHomeVersionWindows(ctx, groupIDs, limit)
}

// TeamHomeValueStream returns the same nine stage categories as demand management,
// scoped by the authorized agile groups and formal group members.
func (s *Service) TeamHomeValueStream(ctx context.Context, groupIDs []uint, memberAccounts []string) ([]TeamHomeValueStreamStage, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("po repo is not configured")
	}
	counts, err := s.repo.CountTeamValueStreamStages(ctx, groupIDs, memberAccounts)
	if err != nil {
		return nil, err
	}
	byStage := make(map[int]map[string]int64, len(valueStreamStages))
	for _, row := range counts {
		if row.StageIndex <= 0 || row.StageIndex >= len(valueStreamStages) {
			continue
		}
		if byStage[row.StageIndex] == nil {
			byStage[row.StageIndex] = map[string]int64{}
		}
		byStage[row.StageIndex][row.Kind] = row.Count
	}
	out := make([]TeamHomeValueStreamStage, 0, len(valueStreamStages)-1)
	for index, stage := range valueStreamStages {
		if stage.status == "all" {
			continue
		}
		demandCount, storyCount := byStage[index]["demand"], byStage[index]["story"]
		out = append(out, TeamHomeValueStreamStage{
			Status: stage.status, Label: stage.label,
			Count: demandCount + storyCount, DemandCount: demandCount, StoryCount: storyCount,
		})
	}
	return out, nil
}

// Demands 按价值流状态返回当前用户关联的需求/故事详情。
func (s *Service) Demands(ctx context.Context, actor *model.User, req DemandsReq) (*DemandsResp, error) {
	req.Normalize()
	displayMap, err := s.loadAccountDisplayMap(ctx, actor)
	if err != nil {
		return nil, err
	}
	// “全部”按 Main 口径合并业务需求与研发需求；带焦点筛选时使用 SQL 候选集。
	if req.Status == "all" && (req.Focus == "" || req.Focus == "all") {
		return s.listAllStageDemands(ctx, actor, req, displayMap)
	}
	if req.Focus != "" && req.Focus != "all" {
		if errs := req.Validate(); len(errs) > 0 {
			return nil, fmt.Errorf("invalid home focus request")
		}
		account := ""
		if actor != nil {
			account = actor.Account
		}
		reviewIDs, err := s.repo.FindAccountPendingReviewDemandIDs(ctx, account)
		if err != nil {
			return nil, err
		}
		var (
			refs     []itemRef
			total    int
			summary  []ValueStreamStage
			focusErr error
			sumErr   error
			wg       sync.WaitGroup
		)
		runQuerySafely(&wg, &focusErr, func() {
			refs, total, focusErr = s.repo.findHomeFocusWithReviews(ctx, account, req, reviewIDs)
		})
		runQuerySafely(&wg, &sumErr, func() {
			summary, sumErr = s.repo.homeFocusStageSummaryWithReviews(ctx, account, req, reviewIDs)
		})
		wg.Wait()
		if focusErr != nil {
			return nil, focusErr
		}
		if sumErr != nil {
			return nil, sumErr
		}
		resp, err := s.populateWorkItems(ctx, actor, refs, total, req.Page, req.PageSize, displayMap, req)
		if err != nil {
			return nil, err
		}
		resp.StageSummary = summary
		return resp, nil
	}
	if filter, ok := mysqlStageFilters[req.Status]; ok {
		return s.listMySQLDemands(ctx, actor, req.Status, filter, req, displayMap)
	}
	return &DemandsResp{Items: []WorkItemDetail{}, Total: 0, Page: req.Page, PageSize: req.PageSize}, nil
}
