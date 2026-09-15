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
	"strconv"
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
	ztAPI    *zentao.Client
	logger   *zap.Logger
}

// NewService 创建 Service。ztAPI 可为空，评审写禅道时回退 zentao.API()。
func NewService(repo *Repo, scheduleSvc *schedule.Service, userSvc *user.Service, ztAPI *zentao.Client, logger *zap.Logger) *Service {
	return &Service{repo: repo, schedule: scheduleSvc, userSvc: userSvc, ztAPI: ztAPI, logger: logger}
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

	launchWindows := []LaunchWindowOption{}
	vwRows, vwErr := s.repo.ListVersionWindows(ctx)
	if vwErr != nil {
		if s.logger != nil {
			s.logger.Warn("po home launch windows", zap.Error(vwErr))
		}
	} else {
		launchWindows = make([]LaunchWindowOption, 0, len(vwRows))
		for _, w := range vwRows {
			launchWindows = append(launchWindows, LaunchWindowOption{
				ID:          w.ID,
				Name:        strings.TrimSpace(w.Name),
				ReleaseDate: w.ReleaseDate.Format("2006-01-02"),
			})
		}
	}

	return &HomeResp{
		Stages:         stages,
		VersionWindows: versionWindows,
		LaunchWindows:  launchWindows,
		Users:          s.listVerifierUsers(ctx, actor),
	}, nil
}

// Demands 按价值流状态返回当前用户关联的需求/故事详情（后端分页）。
func (s *Service) Demands(ctx context.Context, actor *model.User, req DemandsReq) (*DemandsResp, error) {
	displayMap, err := s.loadAccountDisplayMap(ctx, actor)
	if err != nil {
		return nil, err
	}
	var resp *DemandsResp
	if req.Status == "all" {
		resp, err = s.listAllStageDemands(ctx, actor, displayMap)
	} else if filter, ok := mysqlStageFilters[req.Status]; ok {
		resp, err = s.listMySQLDemands(ctx, actor, req.Status, filter, displayMap, req.Page, req.PageSize)
	} else {
		resp = &DemandsResp{Items: []WorkItemDetail{}}
	}
	if err != nil {
		return nil, err
	}
	if resp == nil {
		resp = &DemandsResp{Items: []WorkItemDetail{}}
	}
	// 「全部」在 list 内未切页时在此统一切页；单阶段已带分页的保留 Total
	if req.Status == "all" {
		pageItems, total := paginateWorkItems(resp.Items, req.Page, req.PageSize)
		resp.Items = pageItems
		resp.Total = total
	}
	resp.Page = req.Page
	resp.PageSize = req.PageSize
	if resp.Items == nil {
		resp.Items = []WorkItemDetail{}
	}
	return resp, nil
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
// 返回全量 Items，由 Demands 统一切页。
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
		// 拉全量再并集；分页在 Demands 出口统一切
		resp, err := s.listMySQLDemands(ctx, actor, def.status, filter, displayMap, 0, 0)
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
	return &DemandsResp{Items: items, Total: int64(len(items))}, nil
}

// listMySQLDemands 从 MySQL 加载指定价值流阶段的业需列表（排期/交付阶段额外合并独立研发需求）。
// pageSize<=0 表示不分页（供「全部」并集）；否则后端分页并填充 Total。
func (s *Service) listMySQLDemands(ctx context.Context, actor *model.User, stageStatus string, filter mysqlStageFilter, displayMap map[string]string, page, pageSize int) (*DemandsResp, error) {
	account := ""
	if actor != nil {
		account = actor.Account
	}
	needMerge := filter.scheduleIncomplete || filter.deliverStories
	unpaged := pageSize <= 0

	if !needMerge && !unpaged {
		total, countErr := s.repo.CountRoleDemands(ctx, account, filter)
		if countErr != nil {
			return nil, countErr
		}
		offset := (page - 1) * pageSize
		rows, err := s.repo.FindRoleDemands(ctx, account, filter, pageSize, offset)
		if err != nil {
			return nil, err
		}
		items, buildErr := s.buildDemandWorkItems(ctx, account, stageStatus, rows, displayMap)
		if buildErr != nil {
			return nil, buildErr
		}
		return &DemandsResp{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
	}

	rows, err := s.repo.FindRoleDemands(ctx, account, filter, 0, 0)
	if err != nil {
		return nil, err
	}
	items, buildErr := s.buildDemandWorkItems(ctx, account, stageStatus, rows, displayMap)
	if buildErr != nil {
		return nil, buildErr
	}
	label := valueStreamLabelForStatus(stageStatus)
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
	if unpaged {
		return &DemandsResp{Items: items, Total: int64(len(items))}, nil
	}
	pageItems, total := paginateWorkItems(items, page, pageSize)
	return &DemandsResp{Items: pageItems, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) buildDemandWorkItems(ctx context.Context, account, stageStatus string, rows []DemandRow, displayMap map[string]string) ([]WorkItemDetail, error) {
	label := valueStreamLabelForStatus(stageStatus)
	waitIDs := make([]int, 0, len(rows))
	productIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Status) == "wait" {
			waitIDs = append(waitIDs, row.ID)
		}
		if pid := parseMainSystemID(row.MainSystem); pid > 0 {
			productIDs = append(productIDs, pid)
		}
	}
	pendingReview, pendingErr := s.repo.FindPendingReviewDemandIDs(ctx, account, waitIDs)
	if pendingErr != nil {
		return nil, pendingErr
	}
	testtaskByProduct, ttErr := s.repo.FindMaxTesttaskIDByProducts(ctx, productIDs)
	if ttErr != nil {
		return nil, ttErr
	}
	items := make([]WorkItemDetail, 0, len(rows))
	for _, row := range rows {
		pri := ""
		if row.Pri != "" {
			pri = "P" + row.Pri
		}
		ownerDisp := resolveNextOwnerDisplay(row, displayMap)
		_, canReview := pendingReview[row.ID]
		status := strings.TrimSpace(row.Status)
		isCreator := account != "" && strings.TrimSpace(row.CreatedBy) == account
		canOwnerDraft := isCreator && (status == "draft" || status == "refuse")
		testtaskURL := ""
		if pid := parseMainSystemID(row.MainSystem); pid > 0 {
			if taskID := testtaskByProduct[pid]; taskID > 0 {
				testtaskURL = zentao.URL("testtask", "cases", fmt.Sprintf("taskID=%d", taskID))
			}
		}
		items = append(items, WorkItemDetail{
			Kind:            "demand",
			ID:              fmt.Sprintf("US%d", row.ID),
			Pri:             pri,
			Title:           row.Name,
			Owner:           ownerDisp,
			NextOwner:       ownerDisp,
			ZentaoUrl:       zentao.URL("demand", "view", fmt.Sprintf("demandID=%d", row.ID)),
			ClarifyUrl:      zentao.URL("demand", "clarify", fmt.Sprintf("demandID=%d", row.ID)),
			AppraiseUrl:     zentao.URL("demand", "appraise", fmt.Sprintf("demandID=%d", row.ID)),
			TesttaskUrl:     testtaskURL,
			ValueStream:     label,
			ZentaoStatus:    row.Status,
			CanReview:       canReview,
			CanCancelReview: isCreator && status == "wait",
			CanSubmitReview: canOwnerDraft,
			CanEdit:         canOwnerDraft,
		})
	}
	return items, nil
}

// parseMainSystemID 将业需 mainSystem 字符串解析为产品 ID。
func parseMainSystemID(raw string) uint {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return uint(n)
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
