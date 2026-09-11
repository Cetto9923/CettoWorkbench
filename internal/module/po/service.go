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

// Service PO 工作台业务逻辑。
type Service struct {
	repo         *Repo
	detailSvc    *DetailService
	schedule     *schedule.Service
	userSvc      *user.Service
	logger       *zap.Logger
	issueActions zentao.IssueActionGateway
	taskActions  taskStatusGateway
}

// NewService 创建 Service。
func NewService(repo *Repo, scheduleSvc *schedule.Service, userSvc *user.Service, logger *zap.Logger) *Service {
	var detailSvc *DetailService
	if repo != nil && repo.db != nil {
		detailSvc = NewDetailService(NewDemandDetailRepo(repo.db))
	}
	s := &Service{repo: repo, detailSvc: detailSvc, schedule: scheduleSvc, userSvc: userSvc, logger: logger,
		issueActions: zentao.NewUnavailableIssueActionGateway("当前禅道 API 未提供问题解决、关闭或重新激活动作接口"),
		taskActions:  zentao.DefaultClient()}
	if detailSvc != nil {
		detailSvc.attachParent(s)
	}
	return s
}

type taskStatusGateway interface {
	UpdateTaskStatus(ctx context.Context, p zentao.TaskStatusParams) error
	UpdateTask(ctx context.Context, p zentao.UpdateTaskParams) error
}

// SetTaskStatusGateway 注入禅道任务状态网关，供测试替换。
func (s *Service) SetTaskStatusGateway(gateway taskStatusGateway) {
	if s == nil {
		return
	}
	s.taskActions = gateway
}

// SetIssueActionGateway 注入禅道问题原生动作网关；nil 始终失败关闭。
func (s *Service) SetIssueActionGateway(gateway zentao.IssueActionGateway) {
	if s == nil {
		return
	}
	if gateway == nil {
		gateway = zentao.NewUnavailableIssueActionGateway("当前禅道原生问题操作接口不可用")
	}
	s.issueActions = gateway
}

// DetailService 返回统一详情服务。
func (s *Service) DetailService() *DetailService {
	if s == nil {
		return nil
	}
	return s.detailSvc
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
		breakdown  []ValueStreamStage
		allErr     error
		windows    []schedule.HomeVersionWindowCard
		winErr     error
		myPending  int64
		pendingErr error
		kpiSummary KPISummaryResult
		kpiErr     error
		wg         sync.WaitGroup
	)

	// 并行加载价值流全景统计、版本窗口与 KPI 指标，大幅缩短首屏加载耗时。
	wg.Add(1)
	go func() {
		defer wg.Done()
		breakdown, allErr = s.countAllStageBreakdown(ctx, account)
	}()

	if s.schedule != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			windows, winErr = s.schedule.ListHomeVersionWindows(ctx, actor)
		}()
	}

	if strings.TrimSpace(account) != "" {
		wg.Add(2)
		go func() {
			defer wg.Done()
			myPending, pendingErr = s.repo.CountHomeFocus(ctx, account, DemandsReq{Status: "all", Focus: "my_action"})
		}()
		go func() {
			defer wg.Done()
			kpiSummary, kpiErr = s.repo.CountKPISummary(ctx, account)
		}()
	}

	wg.Wait()

	if allErr != nil {
		return nil, allErr
	}
	if pendingErr != nil {
		return nil, pendingErr
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
		Overdue:   kpiSummary.Overdue,
		Suspended: kpiSummary.Suspended,
		Blocked:   kpiSummary.Blocked,
		MyPending: myPending,
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

// Demands 按价值流状态返回当前用户关联的需求/故事详情。
func (s *Service) Demands(ctx context.Context, actor *model.User, req DemandsReq) (*DemandsResp, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 15
	} else if req.PageSize > 100 {
		req.PageSize = 100
	}
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
		var (
			refs     []itemRef
			total    int
			summary  []ValueStreamStage
			focusErr error
			sumErr   error
			wg       sync.WaitGroup
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			refs, total, focusErr = s.repo.FindHomeFocus(ctx, account, req)
		}()
		go func() {
			defer wg.Done()
			summary, sumErr = s.repo.HomeFocusStageSummary(ctx, account, req)
		}()
		wg.Wait()
		if focusErr != nil {
			return nil, focusErr
		}
		if sumErr != nil {
			return nil, sumErr
		}
		resp, err := s.populateWorkItems(ctx, actor, refs, total, req.Page, req.PageSize, displayMap)
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
