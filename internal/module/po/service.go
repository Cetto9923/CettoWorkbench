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

	"go.uber.org/zap"

	"workbench/internal/model"
	"workbench/internal/module/schedule"
	"workbench/internal/module/user"
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
	repo     *Repo
	schedule *schedule.Service
	userSvc  *user.Service
	logger   *zap.Logger
}

// NewService 创建 Service。
func NewService(repo *Repo, scheduleSvc *schedule.Service, userSvc *user.Service, logger *zap.Logger) *Service {
	return &Service{repo: repo, schedule: scheduleSvc, userSvc: userSvc, logger: logger}
}

// Home 加载首页价值流阶段统计。
func (s *Service) Home(ctx context.Context, actor *model.User) (*HomeResp, error) {
	account := ""
	if actor != nil {
		account = actor.Account
	}

	stages := make([]ValueStreamStage, 0, len(valueStreamStages))
	allIdx := -1
	for _, def := range valueStreamStages {
		if def.status == "all" {
			allIdx = len(stages)
			stages = append(stages, ValueStreamStage{Label: def.label, Status: def.status})
			continue
		}

		var demand, story int64
		if filter, ok := mysqlStageFilters[def.status]; ok {
			n, countErr := s.repo.CountRoleDemands(ctx, account, filter)
			if countErr != nil {
				return nil, countErr
			}
			demand = n
			if filter.scheduleIncomplete {
				sn, storyErr := s.repo.CountScheduleStories(ctx, account)
				if storyErr != nil {
					return nil, storyErr
				}
				story = sn
			}
			if filter.deliverStories {
				sn, storyErr := s.repo.CountDeliverStories(ctx, account)
				if storyErr != nil {
					return nil, storyErr
				}
				story = sn
			}
		}
		stages = append(stages, ValueStreamStage{
			Label:       def.label,
			Status:      def.status,
			Count:       demand + story,
			DemandCount: demand,
			StoryCount:  story,
		})
	}

	// 「全部」= 各阶段 kind+id 去重并集；只拉 ID，不拉详情
	if allIdx >= 0 {
		demandSum, storySum, allErr := s.countAllStageUniq(ctx, account)
		if allErr != nil {
			return nil, allErr
		}
		stages[allIdx].DemandCount = demandSum
		stages[allIdx].StoryCount = storySum
		stages[allIdx].Count = demandSum + storySum
	}

	versionWindows := []schedule.HomeVersionWindowCard{}
	if s.schedule == nil {
		if s.logger != nil {
			s.logger.Error("po home schedule service is nil, version windows skipped")
		}
	} else {
		windows, winErr := s.schedule.ListHomeVersionWindows(ctx, actor)
		if winErr != nil {
			if s.logger != nil {
				s.logger.Warn("po home version windows", zap.Error(winErr))
			}
		} else {
			versionWindows = windows
		}
	}

	// 5 个焦点摘要：MyPending 已包含在 all 阶段计数；其余 4 个由 Repo 真实统计。
	kpi := KPICounts{MyPending: stages[allIdx].Count}
	if strings.TrimSpace(account) != "" {
		if err := s.fillKPICounts(ctx, account, &kpi); err != nil {
			return nil, err
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
		}
	}

	return &HomeResp{Stages: stages, VersionWindows: versionWindows, KPI: kpi}, nil
}

// kpiCountFn 适配 CountKPI* 方法的统一签名，便于在 fillKPICounts 中以 map 驱动循环。
type kpiCountFn func(context.Context, string) (int64, error)

// fillKPICounts 顺序调用 4 个 CountKPI* 方法并填充 KPICounts；任一失败即返回（首个错误包 label）。
func (s *Service) fillKPICounts(ctx context.Context, account string, kpi *KPICounts) error {
	fills := []struct {
		label string
		fn    kpiCountFn
		dst   *int64
	}{
		{"today", s.repo.CountKPIToday, &kpi.Today},
		{"overdue", s.repo.CountKPIOverdue, &kpi.Overdue},
		{"suspended", s.repo.CountKPISuspended, &kpi.Suspended},
		{"blocked", s.repo.CountKPIBlocked, &kpi.Blocked},
	}
	for _, f := range fills {
		n, err := f.fn(ctx, account)
		if err != nil {
			return fmt.Errorf("kpi %s: %w", f.label, err)
		}
		*f.dst = n
	}
	return nil
}

// Demands 按价值流状态返回当前用户关联的需求/故事详情。
func (s *Service) Demands(ctx context.Context, actor *model.User, req DemandsReq) (*DemandsResp, error) {
	displayMap, err := s.loadAccountDisplayMap(ctx, actor)
	if err != nil {
		return nil, err
	}
	if req.Status == "all" {
		return s.listAllStageDemands(ctx, actor, displayMap)
	}
	if filter, ok := mysqlStageFilters[req.Status]; ok {
		return s.listMySQLDemands(ctx, actor, req.Status, filter, displayMap)
	}
	return &DemandsResp{Items: []WorkItemDetail{}}, nil
}

