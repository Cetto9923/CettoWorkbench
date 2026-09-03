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
		var kpiErr error
		if kpi.Today, kpiErr = s.repo.CountKPIToday(ctx, account); kpiErr != nil {
			return nil, kpiErr
		}
		if kpi.Overdue, kpiErr = s.repo.CountKPIOverdue(ctx, account); kpiErr != nil {
			return nil, kpiErr
		}
		if kpi.Suspended, kpiErr = s.repo.CountKPISuspended(ctx, account); kpiErr != nil {
			return nil, kpiErr
		}
		if kpi.Blocked, kpiErr = s.repo.CountKPIBlocked(ctx, account); kpiErr != nil {
			return nil, kpiErr
		}
	}

	return &HomeResp{Stages: stages, VersionWindows: versionWindows, KPI: kpi}, nil
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

func (s *Service) loadAccountDisplayMap(ctx context.Context, actor *model.User) (map[string]string, error) {
	if s.userSvc == nil {
		return map[string]string{}, nil
	}
	return s.userSvc.AccountDisplayMap(ctx, actor)
}

// countAllStageUniq 各阶段只查 ID，按 kind+id 去重后返回业需/研需数量（与 listAllStageDemands 并集语义一致）。
func (s *Service) countAllStageUniq(ctx context.Context, account string) (demandSum, storySum int64, err error) {
	seenDemand := make(map[int]struct{})
	seenStory := make(map[int]struct{})
	for _, def := range valueStreamStages {
		if def.status == "all" {
			continue
		}
		filter, ok := mysqlStageFilters[def.status]
		if !ok {
			continue
		}
		ids, idErr := s.repo.FindRoleDemandIDs(ctx, account, filter)
		if idErr != nil {
			return 0, 0, idErr
		}
		for _, id := range ids {
			seenDemand[id] = struct{}{}
		}
		if filter.scheduleIncomplete {
			storyIDs, storyErr := s.repo.FindScheduleStoryIDs(ctx, account)
			if storyErr != nil {
				return 0, 0, storyErr
			}
			for _, id := range storyIDs {
				seenStory[id] = struct{}{}
			}
		}
		if filter.deliverStories {
			storyIDs, storyErr := s.repo.FindDeliverStoryIDs(ctx, account)
			if storyErr != nil {
				return 0, 0, storyErr
			}
			for _, id := range storyIDs {
				seenStory[id] = struct{}{}
			}
		}
	}
	return int64(len(seenDemand)), int64(len(seenStory)), nil
}

// listAllStageDemands 「全部」列表 = 其余各阶段列表按阶段顺序拼接，按 kind+id 去重（保留首次出现）。
func (s *Service) listAllStageDemands(ctx context.Context, actor *model.User, displayMap map[string]string) (*DemandsResp, error) {
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
		resp, err := s.listMySQLDemands(ctx, actor, def.status, filter, displayMap)
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
func (s *Service) listMySQLDemands(ctx context.Context, actor *model.User, stageStatus string, filter mysqlStageFilter, displayMap map[string]string) (*DemandsResp, error) {
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
		ownerDisp := resolveNextOwnerDisplay(row, displayMap)
		items = append(items, WorkItemDetail{
			Kind:         "demand",
			ID:           fmt.Sprintf("US%d", row.ID),
			Pri:          pri,
			Title:        row.Name,
			Owner:        ownerDisp,
			NextOwner:    ownerDisp,
			ZentaoUrl:    zentao.URL("demand", "view", fmt.Sprintf("demandID=%d", row.ID)),
			ValueStream:  label,
			ZentaoStatus: row.Status,
		})
	}
	if filter.scheduleIncomplete {
		stories, storyErr := s.repo.FindScheduleStories(ctx, account)
		if storyErr != nil {
			return nil, storyErr
		}
		items = append(items, storyWorkItems(stories, label, actor, displayMap)...)
	}
	if filter.deliverStories {
		stories, storyErr := s.repo.FindDeliverStories(ctx, account)
		if storyErr != nil {
			return nil, storyErr
		}
		items = append(items, storyWorkItems(stories, label, actor, displayMap)...)
	}
	return &DemandsResp{Items: items}, nil
}

func resolveNextOwnerDisplay(row DemandRow, displayMap map[string]string) string {
	_, disp := DeriveCurrentHandler(row.Status, row.AssignedTo, row.QD, row.RD, row.BRA, row.PM,
		lookupAccountsDisplay(displayMap, row.PM),
		lookupAccountDisplay(displayMap, row.AssignedTo),
		lookupAccountDisplay(displayMap, row.QD),
		lookupAccountDisplay(displayMap, row.RD),
		lookupAccountDisplay(displayMap, row.BRA))
	return disp
}

func lookupAccountDisplay(displayMap map[string]string, account string) string {
	acc := strings.TrimSpace(account)
	if acc == "" {
		return ""
	}
	if displayMap != nil {
		if d := strings.TrimSpace(displayMap[acc]); d != "" {
			return d
		}
	}
	return acc
}

// lookupAccountsDisplay 支持 GROUP_CONCAT 多账号，逐个映射后用 ", " 拼接。
func lookupAccountsDisplay(displayMap map[string]string, accountsCSV string) string {
	raw := strings.TrimSpace(accountsCSV)
	if raw == "" {
		return ""
	}
	parts := strings.Split(raw, ",")
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		names = append(names, lookupAccountDisplay(displayMap, p))
	}
	return strings.Join(names, ", ")
}

func storyWorkItems(rows []StoryRow, label string, actor *model.User, displayMap map[string]string) []WorkItemDetail {
	account := ""
	if actor != nil {
		account = actor.Account
	}
	owner := lookupAccountDisplay(displayMap, account)
	if owner == "" && actor != nil {
		owner = FormatAccountName(actor.Account, actor.DisplayName)
	}
	items := make([]WorkItemDetail, 0, len(rows))
	for _, row := range rows {
		items = append(items, WorkItemDetail{
			Kind:         "story",
			ID:           fmt.Sprintf("U%d", row.ID),
			Pri:          fmt.Sprintf("P%d", row.Pri),
			Title:        row.Title,
			Owner:        owner,
			NextOwner:    owner,
			ZentaoUrl:    zentao.URL("story", "view", fmt.Sprintf("storyID=%d", row.ID)),
			ValueStream:  label,
			ZentaoStatus: row.Status,
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
