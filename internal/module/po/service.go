// =============================================================================
// 文件: internal/module/po/service.go
// 模块: PO 工作台
// 类型: action
// 职责: 组装 PO 首页价值流统计与需求列表（各阶段读只读备库，「全部」为其余阶段去重总和；排期窗口走主库 schedule）。
// 依赖: internal/model
//       internal/module/schedule
//       internal/pkg/zentao
// =============================================================================

package po

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"workbench/internal/model"
	"workbench/internal/module/schedule"
	"workbench/internal/pkg/zentao"
)

// valueStreamStages 价值流卡片（UI 视角）。
// 业务口径：
//   - 「发布」 = waitdeliver（在禅道「等待发布」语义）
//   - 「评价」 = released && overall=0 && parent≠-1 && overall≠5（PO 待评价,排除已被系统自动五星评价的）
// 「发布」与「评价」业务相邻但 SQL 装载层不同，故拆为 2 段。
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
	{label: "发布", status: "waitdeliver"},
	{label: "评价", status: "released"},
}

// Service PO 工作台业务逻辑。
type Service struct {
	repo     *Repo
	schedule *schedule.Service
	logger   *zap.Logger
}

// NewService 创建 Service。
func NewService(repo *Repo, scheduleSvc *schedule.Service, logger *zap.Logger) *Service {
	return &Service{repo: repo, schedule: scheduleSvc, logger: logger}
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

	// 「全部」= 其余各阶段去重后的 demand/story 数量
	if allIdx >= 0 {
		allResp, allErr := s.listAllStageDemands(ctx, actor)
		if allErr != nil {
			return nil, allErr
		}
		var demandSum, storySum int64
		for _, item := range allResp.Items {
			if item.Kind == "story" {
				storySum++
			} else {
				demandSum++
			}
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

	return &HomeResp{Stages: stages, VersionWindows: versionWindows}, nil
}

// Demands 按价值流状态返回当前用户关联的需求/故事详情。
func (s *Service) Demands(ctx context.Context, actor *model.User, req DemandsReq) (*DemandsResp, error) {
	if req.Status == "all" {
		return s.listAllStageDemands(ctx, actor)
	}
	if filter, ok := mysqlStageFilters[req.Status]; ok {
		return s.listMySQLDemands(ctx, actor, req.Status, filter)
	}
	return &DemandsResp{Items: []WorkItemDetail{}}, nil
}

// listAllStageDemands 「全部」列表 = 其余各阶段列表按阶段顺序拼接，按 kind+id 去重（保留首次出现）。
func (s *Service) listAllStageDemands(ctx context.Context, actor *model.User) (*DemandsResp, error) {
	items := make([]WorkItemDetail, 0)
	seen := make(map[string]struct{})
	for _, def := range valueStreamStages {
		if def.status == "all" {
			continue
		}
		filter, ok := mysqlStageFilters[def.status]
		if !ok {
			continue
		}
		resp, err := s.listMySQLDemands(ctx, actor, def.status, filter)
		if err != nil {
			return nil, err
		}
		for _, item := range resp.Items {
			key := workItemKey(item.Kind, item.ID)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			items = append(items, item)
		}
	}
	return &DemandsResp{Items: items}, nil
}

// listMySQLDemands 从 MySQL 加载指定价值流阶段的业需列表（排期/交付阶段额外合并独立研发需求）。
func (s *Service) listMySQLDemands(ctx context.Context, actor *model.User, stageStatus string, filter mysqlStageFilter) (*DemandsResp, error) {
	account := ""
	if actor != nil {
		account = actor.Account
	}
	rows, err := s.repo.FindRoleDemands(ctx, account, filter)
	if err != nil {
		return nil, err
	}
	label := valueStreamLabelForStatus(stageStatus)
	items := make([]WorkItemDetail, 0, len(rows))
	for _, row := range rows {
		pri := ""
		if row.Pri != "" {
			pri = "P" + row.Pri
		}
		items = append(items, WorkItemDetail{
			Kind:        "demand",
			ID:          fmt.Sprintf("%d", row.ID),
			Pri:         pri,
			Title:       row.Name,
			ZentaoUrl:   zentao.URL("demand", "view", fmt.Sprintf("demandID=%d", row.ID)),
			ValueStream: label,
		})
	}
	if filter.scheduleIncomplete {
		stories, storyErr := s.repo.FindScheduleStories(ctx, account)
		if storyErr != nil {
			return nil, storyErr
		}
		items = append(items, storyWorkItems(stories, label)...)
	}
	if filter.deliverStories {
		stories, storyErr := s.repo.FindDeliverStories(ctx, account)
		if storyErr != nil {
			return nil, storyErr
		}
		items = append(items, storyWorkItems(stories, label)...)
	}
	return &DemandsResp{Items: items}, nil
}

func storyWorkItems(rows []StoryRow, label string) []WorkItemDetail {
	items := make([]WorkItemDetail, 0, len(rows))
	for _, row := range rows {
		items = append(items, WorkItemDetail{
			Kind:        "story",
			ID:          fmt.Sprintf("%d", row.ID),
			Pri:         fmt.Sprintf("P%d", row.Pri),
			Title:       row.Title,
			ZentaoUrl:   zentao.URL("story", "view", fmt.Sprintf("storyID=%d", row.ID)),
			ValueStream: label,
		})
	}
	return items
}

func isValidValueStreamStatus(status string) bool {
	for _, def := range valueStreamStages {
		if def.status == status {
			return true
		}
	}
	return false
}

func valueStreamLabelForStatus(status string) string {
	for _, def := range valueStreamStages {
		if def.status == status {
			return def.label
		}
	}
	return ""
}

func workItemKey(kind, id string) string {
	return kind + ":" + id
}
