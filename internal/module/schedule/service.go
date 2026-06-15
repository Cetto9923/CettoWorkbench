// =============================================================================
// 文件: internal/module/schedule/service.go
// 模块: 排期工作台
// 类型: action
// 职责: 实现排期工作台页面业务逻辑。
// 依赖: internal/model
//       internal/module/schedule/repo.go
// =============================================================================

package schedule

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"workbench/internal/model"
)

var windowCardToneClasses = []string{"red", "blue", "green", "purple"}

// Service 处理排期业务逻辑。
type Service struct {
	repo   *Repo
	logger *zap.Logger
}

// NewService 创建 Service。
func NewService(repo *Repo, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// GetUserTeamgroups 查询用户所属敏捷小组并拼接展示名称。
func (s *Service) GetUserTeamgroups(ctx context.Context, account string) ([]TeamgroupOption, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return []TeamgroupOption{}, nil
	}

	isAdmin, err := s.repo.IsAdmin(ctx, account)
	if err != nil {
		return nil, err
	}

	var groups []ZtTeamgroup
	if isAdmin {
		groups, err = s.repo.ListAllTeamgroups(ctx)
	} else {
		groups, err = s.repo.GetUserTeamgroups(ctx, account)
	}
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return []TeamgroupOption{}, nil
	}

	parentIDs := make([]uint, 0)
	parentSeen := make(map[uint]struct{})
	for _, group := range groups {
		if group.Parent == 0 {
			continue
		}
		if _, ok := parentSeen[group.Parent]; ok {
			continue
		}
		parentSeen[group.Parent] = struct{}{}
		parentIDs = append(parentIDs, group.Parent)
	}

	parentNameByID := make(map[uint]string, len(parentIDs))
	if len(parentIDs) > 0 {
		parents, err := s.repo.FindTeamgroupsByIDs(ctx, parentIDs)
		if err != nil {
			return nil, err
		}
		for _, parent := range parents {
			parentNameByID[parent.ID] = strings.TrimSpace(parent.Name)
		}
	}

	options := make([]TeamgroupOption, 0, len(groups))
	for _, group := range groups {
		displayName := strings.TrimSpace(group.Name)
		if group.Parent > 0 {
			parentName := parentNameByID[group.Parent]
			if parentName != "" {
				displayName = fmt.Sprintf("%s / %s", parentName, displayName)
			}
		}
		options = append(options, TeamgroupOption{
			ID:          group.ID,
			DisplayName: displayName,
		})
	}
	return options, nil
}

func actorAccount(actor *model.User) string {
	if actor == nil {
		return ""
	}
	return strings.TrimSpace(actor.Account)
}

// GetCreateWindowFormData 查询新建版本窗口弹窗所需表单数据。
func (s *Service) GetCreateWindowFormData(ctx context.Context, actor *model.User) (*CreateWindowFormData, error) {
	account := actorAccount(actor)
	teamgroups, err := s.GetUserTeamgroups(ctx, account)
	if err != nil {
		return nil, err
	}
	products, err := s.repo.GetUserProducts(ctx, account)
	if err != nil {
		return nil, err
	}
	if products == nil {
		products = []ZtProduct{}
	}
	return &CreateWindowFormData{
		Teamgroups: teamgroups,
		Products:   products,
	}, nil
}

func computeWindowPermissions(createdBy, account string, demandCount int) (canEdit, canDelete, hasLinkedDemands bool) {
	hasLinkedDemands = demandCount > 0
	canEdit = strings.TrimSpace(createdBy) == strings.TrimSpace(account)
	canDelete = canEdit && !hasLinkedDemands
	return
}

// ListWindowCards 查询版本窗口概览卡片数据。
func (s *Service) ListWindowCards(ctx context.Context, actor *model.User) ([]WindowCard, error) {
	account := actorAccount(actor)
	windows, _, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	if len(windows) == 0 {
		return []WindowCard{}, nil
	}

	teamgroupIDs := make([]uint, 0, len(windows))
	seen := make(map[uint]struct{}, len(windows))
	for _, window := range windows {
		if window.TeamgroupID == 0 {
			continue
		}
		if _, ok := seen[window.TeamgroupID]; ok {
			continue
		}
		seen[window.TeamgroupID] = struct{}{}
		teamgroupIDs = append(teamgroupIDs, window.TeamgroupID)
	}

	teamgroupNameByID := make(map[uint]string, len(teamgroupIDs))
	if len(teamgroupIDs) > 0 {
		groups, err := s.repo.FindTeamgroupsByIDs(ctx, teamgroupIDs)
		if err != nil {
			return nil, err
		}
		parentIDs := make([]uint, 0)
		parentSeen := make(map[uint]struct{})
		for _, group := range groups {
			teamgroupNameByID[group.ID] = strings.TrimSpace(group.Name)
			if group.Parent == 0 {
				continue
			}
			if _, ok := parentSeen[group.Parent]; ok {
				continue
			}
			parentSeen[group.Parent] = struct{}{}
			parentIDs = append(parentIDs, group.Parent)
		}
		if len(parentIDs) > 0 {
			parents, err := s.repo.FindTeamgroupsByIDs(ctx, parentIDs)
			if err != nil {
				return nil, err
			}
			parentNameByID := make(map[uint]string, len(parents))
			for _, parent := range parents {
				parentNameByID[parent.ID] = strings.TrimSpace(parent.Name)
			}
			for _, group := range groups {
				name := strings.TrimSpace(group.Name)
				if group.Parent > 0 {
					if parentName := parentNameByID[group.Parent]; parentName != "" {
						name = fmt.Sprintf("%s / %s", parentName, name)
					}
				}
				teamgroupNameByID[group.ID] = name
			}
		}
	}

	cards := make([]WindowCard, 0, len(windows))
	for i, window := range windows {
		start := window.ReleaseDate
		if window.StartDate != nil {
			start = *window.StartDate
		}
		capacityHours, err := s.CalcCapacity(
			ctx,
			start.Format("2006-01-02"),
			window.ReleaseDate.Format("2006-01-02"),
			int(window.GroupSize),
		)
		if err != nil {
			return nil, err
		}

		consumed, err := s.repo.GetWindowConsumedHours(ctx, window.ID)
		if err != nil {
			return nil, err
		}
		demandCount, err := s.repo.GetWindowDemandCount(ctx, window.ID)
		if err != nil {
			return nil, err
		}

		usedHours := int(math.Round(consumed))
		remainingHours := capacityHours - usedHours
		usedPercent := 0
		if capacityHours > 0 {
			usedPercent = usedHours * 100 / capacityHours
		}

		canEdit, canDelete, hasLinkedDemands := computeWindowPermissions(window.CreatedBy, account, demandCount)

		cards = append(cards, WindowCard{
			ID:               window.ID,
			ShortName:        window.Name,
			Range:            formatWindowDateRange(start, window.ReleaseDate),
			ToneClass:        windowCardToneClasses[i%len(windowCardToneClasses)],
			AgileGroup:       teamgroupNameByID[window.TeamgroupID],
			DemandCount:      demandCount,
			CapacityHours:    capacityHours,
			UsedHours:        usedHours,
			RemainingHours:   remainingHours,
			BlockedCount:     0,
			UsedPercent:      usedPercent,
			CanEdit:          canEdit,
			CanDelete:        canDelete,
			HasLinkedDemands: hasLinkedDemands,
		})
	}
	return cards, nil
}

// HomeVersionWindowCard PO 首页版本窗口卡片展示数据。
type HomeVersionWindowCard struct {
	Name         string
	AgileGroup   string
	Range        string
	DemandCount  int
	DevCount     int
	TestCount    int
	DeliverCount int
}

// ListHomeVersionWindows 查询 PO 首页近期版本窗口（最多 4 条，按用户敏捷小组过滤）。
func (s *Service) ListHomeVersionWindows(ctx context.Context, actor *model.User) ([]HomeVersionWindowCard, error) {
	account := actorAccount(actor)
	if account == "" {
		s.logHomeVersionWindows(account, nil, "account empty", 0, nil)
		return []HomeVersionWindowCard{}, nil
	}

	teamgroups, err := s.GetUserTeamgroups(ctx, account)
	if err != nil {
		return nil, err
	}
	teamgroupIDs := make([]uint, 0, len(teamgroups))
	for _, group := range teamgroups {
		if group.ID == 0 {
			continue
		}
		teamgroupIDs = append(teamgroupIDs, group.ID)
	}
	if len(teamgroupIDs) == 0 {
		s.logHomeVersionWindows(account, teamgroupIDs, "deletedAt IS NULL AND releaseDate>=CURDATE() AND teamgroup IN (...)", 0, nil)
		return []HomeVersionWindowCard{}, nil
	}

	windows, err := s.repo.ListUpcomingVersionWindowsForTeamgroups(ctx, teamgroupIDs, 4)
	if err != nil {
		return nil, err
	}
	s.logHomeVersionWindows(account, teamgroupIDs, "deletedAt IS NULL AND releaseDate>=CURDATE() AND teamgroup IN (...)", len(windows), windows)
	if len(windows) == 0 {
		return []HomeVersionWindowCard{}, nil
	}

	nameByID, err := s.loadTeamgroupDisplayNames(ctx, windows)
	if err != nil {
		return nil, err
	}

	cards := make([]HomeVersionWindowCard, 0, len(windows))
	for _, window := range windows {
		start := window.ReleaseDate
		if window.StartDate != nil {
			start = *window.StartDate
		}
		stats, err := s.repo.GetWindowStageStats(ctx, window.ID)
		if err != nil {
			return nil, err
		}
		cards = append(cards, HomeVersionWindowCard{
			Name:         window.Name,
			AgileGroup:   nameByID[window.TeamgroupID],
			Range:        formatWindowDateRange(start, window.ReleaseDate),
			DemandCount:  stats.DemandCount,
			DevCount:     stats.DevCount,
			TestCount:    stats.TestCount,
			DeliverCount: stats.DeliverCount,
		})
	}
	return cards, nil
}

func (s *Service) logHomeVersionWindows(account string, teamgroupIDs []uint, sqlCondition string, resultCount int, windows []model.VersionWindow) {
	if s.logger == nil {
		return
	}
	fields := []zap.Field{
		zap.String("account", account),
		zap.Uint64s("teamgroup_ids", uintsToUint64s(teamgroupIDs)),
		zap.String("sql_condition", sqlCondition),
		zap.Int("result_count", resultCount),
	}
	if len(windows) > 0 {
		ids := make([]uint64, 0, len(windows))
		names := make([]string, 0, len(windows))
		for _, window := range windows {
			ids = append(ids, window.ID)
			names = append(names, window.Name)
		}
		fields = append(fields, zap.Uint64s("window_ids", ids), zap.Strings("window_names", names))
	}
	s.logger.Info("home version windows query", fields...)
}

func uintsToUint64s(values []uint) []uint64 {
	out := make([]uint64, len(values))
	for i, value := range values {
		out[i] = uint64(value)
	}
	return out
}

func (s *Service) loadTeamgroupDisplayNames(ctx context.Context, windows []model.VersionWindow) (map[uint]string, error) {
	teamgroupIDs := make([]uint, 0, len(windows))
	seen := make(map[uint]struct{}, len(windows))
	for _, window := range windows {
		if window.TeamgroupID == 0 {
			continue
		}
		if _, ok := seen[window.TeamgroupID]; ok {
			continue
		}
		seen[window.TeamgroupID] = struct{}{}
		teamgroupIDs = append(teamgroupIDs, window.TeamgroupID)
	}
	if len(teamgroupIDs) == 0 {
		return map[uint]string{}, nil
	}

	groups, err := s.repo.FindTeamgroupsByIDs(ctx, teamgroupIDs)
	if err != nil {
		return nil, err
	}
	nameByID := make(map[uint]string, len(groups))
	parentIDs := make([]uint, 0)
	parentSeen := make(map[uint]struct{})
	for _, group := range groups {
		nameByID[group.ID] = strings.TrimSpace(group.Name)
		if group.Parent == 0 {
			continue
		}
		if _, ok := parentSeen[group.Parent]; ok {
			continue
		}
		parentSeen[group.Parent] = struct{}{}
		parentIDs = append(parentIDs, group.Parent)
	}
	if len(parentIDs) > 0 {
		parents, err := s.repo.FindTeamgroupsByIDs(ctx, parentIDs)
		if err != nil {
			return nil, err
		}
		parentNameByID := make(map[uint]string, len(parents))
		for _, parent := range parents {
			parentNameByID[parent.ID] = strings.TrimSpace(parent.Name)
		}
		for _, group := range groups {
			name := strings.TrimSpace(group.Name)
			if group.Parent > 0 {
				if parentName := parentNameByID[group.Parent]; parentName != "" {
					name = fmt.Sprintf("%s / %s", parentName, name)
				}
			}
			nameByID[group.ID] = name
		}
	}
	return nameByID, nil
}

func dateOnly(value time.Time) time.Time {
	value = value.In(time.Local)
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

// ListWindows 查询版本窗口维护列表。
func (s *Service) ListWindows(ctx context.Context, actor *model.User) (ListWindowsResp, error) {
	account := actorAccount(actor)
	windows, _, err := s.repo.FindAll(ctx)
	if err != nil {
		return ListWindowsResp{}, err
	}
	if len(windows) == 0 {
		return ListWindowsResp{Windows: []WindowListItem{}}, nil
	}

	items := make([]WindowListItem, 0, len(windows))
	for _, window := range windows {
		start := window.ReleaseDate
		if window.StartDate != nil {
			start = *window.StartDate
		}
		capacityHours, err := s.CalcCapacity(
			ctx,
			start.Format("2006-01-02"),
			window.ReleaseDate.Format("2006-01-02"),
			int(window.GroupSize),
		)
		if err != nil {
			return ListWindowsResp{}, err
		}
		demandCount, err := s.repo.GetWindowDemandCount(ctx, window.ID)
		if err != nil {
			return ListWindowsResp{}, err
		}
		canEdit, canDelete, hasLinkedDemands := computeWindowPermissions(window.CreatedBy, account, demandCount)

		items = append(items, WindowListItem{
			ID:               window.ID,
			Name:             window.Name,
			ReleaseDate:      window.ReleaseDate.Format("2006-01-02"),
			Range:            formatWindowDateRange(start, window.ReleaseDate),
			CapacityHours:    capacityHours,
			CanEdit:          canEdit,
			CanDelete:        canDelete,
			HasLinkedDemands: hasLinkedDemands,
		})
	}
	return ListWindowsResp{Windows: items}, nil
}

func buildVersionWindowFromCreateReq(req CreateReq) (*model.VersionWindow, error) {
	releaseDate, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(req.ReleaseDate), time.Local)
	if err != nil {
		return nil, fmt.Errorf("invalid release date")
	}

	window := &model.VersionWindow{
		Name:        strings.TrimSpace(req.Name),
		ReleaseDate: releaseDate,
		TeamgroupID: req.TeamgroupID,
		Status:      "planning",
	}
	if req.GroupSize > 0 {
		window.GroupSize = uint(req.GroupSize)
	} else {
		window.GroupSize = 1
	}

	startDate := strings.TrimSpace(req.StartDate)
	if startDate != "" {
		parsed, err := time.ParseInLocation("2006-01-02", startDate, time.Local)
		if err != nil {
			return nil, fmt.Errorf("invalid start date")
		}
		window.StartDate = &parsed
	}
	return window, nil
}

func applyUpdateReqToVersionWindow(window *model.VersionWindow, req UpdateReq) error {
	if window == nil {
		return fmt.Errorf("version window is nil")
	}
	releaseDate, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(req.ReleaseDate), time.Local)
	if err != nil {
		return fmt.Errorf("invalid release date")
	}
	window.Name = strings.TrimSpace(req.Name)
	window.ReleaseDate = releaseDate
	window.TeamgroupID = req.TeamgroupID
	if req.GroupSize > 0 {
		window.GroupSize = uint(req.GroupSize)
	} else {
		window.GroupSize = 1
	}
	startDate := strings.TrimSpace(req.StartDate)
	if startDate != "" {
		parsed, err := time.ParseInLocation("2006-01-02", startDate, time.Local)
		if err != nil {
			return fmt.Errorf("invalid start date")
		}
		window.StartDate = &parsed
	} else {
		window.StartDate = nil
	}
	return nil
}

// GetByID 查询版本窗口详情。
func (s *Service) GetByID(ctx context.Context, actor *model.User, id uint64) (*WindowDetailResp, error) {
	_ = actor
	window, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if window == nil {
		return nil, errors.New("窗口不存在")
	}

	products, err := s.repo.GetWindowProducts(ctx, id)
	if err != nil {
		return nil, err
	}

	startDate := ""
	if window.StartDate != nil {
		startDate = window.StartDate.Format("2006-01-02")
	}

	detail := &WindowDetailResp{
		ID:          window.ID,
		ReleaseDate: window.ReleaseDate.Format("2006-01-02"),
		Name:        window.Name,
		StartDate:   startDate,
		TeamgroupID: window.TeamgroupID,
		GroupSize:   window.GroupSize,
		Products:    make([]WindowProductDetail, 0, len(products)),
	}

	for _, row := range products {
		item := WindowProductDetail{
			ProductID:   row.ProductID,
			ProductName: row.ProductName,
			PlanID:      row.PlanID,
		}
		if row.PlanID != nil && strings.TrimSpace(row.PlanTitle) != "" {
			item.HasMatch = true
			item.Plans = []MatchingPlanItem{{
				ID:    *row.PlanID,
				Title: row.PlanTitle,
				Begin: row.PlanBegin,
				End:   row.PlanEnd,
			}}
		} else {
			item.HasMatch = false
			item.SyncPlan = row.PlanSynced == 1
			item.PlanTitle = strings.TrimSpace(row.PlanTitle)
			if item.PlanTitle == "" {
				item.PlanTitle = window.Name
			}
		}
		detail.Products = append(detail.Products, item)
	}
	return detail, nil
}

func formatWindowDateRange(start, end time.Time) string {
	return start.Format("01-02") + " ~ " + end.Format("01-02")
}

// Create 保存版本窗口并按需同步禅道产品计划。
func (s *Service) Create(ctx context.Context, actor *model.User, req CreateReq) error {
	window, err := buildVersionWindowFromCreateReq(req)
	if err != nil {
		return err
	}
	account := actorAccount(actor)
	window.CreatedBy = account
	window.UpdatedBy = account

	return s.repo.Transaction(ctx, func(txRepo *Repo) error {
		if err := txRepo.Create(ctx, window); err != nil {
			return fmt.Errorf("create version window: %w", err)
		}
		return s.saveWindowProducts(ctx, txRepo, window.ID, window, req.Products, account)
	})
}

// Update 更新版本窗口并重建关联产品及计划。
func (s *Service) Update(ctx context.Context, actor *model.User, req UpdateReq) error {
	window, err := s.repo.FindByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if window == nil {
		return errors.New("窗口不存在")
	}
	if err := applyUpdateReqToVersionWindow(window, req); err != nil {
		return err
	}
	account := actorAccount(actor)
	window.UpdatedBy = account

	return s.repo.Transaction(ctx, func(txRepo *Repo) error {
		if err := txRepo.Update(ctx, window); err != nil {
			return fmt.Errorf("update version window: %w", err)
		}
		if err := txRepo.DeleteWindowProducts(ctx, window.ID); err != nil {
			return fmt.Errorf("delete window products: %w", err)
		}
		return s.saveWindowProducts(ctx, txRepo, window.ID, window, req.Products, account)
	})
}

// Delete 软删除版本窗口。
func (s *Service) Delete(ctx context.Context, actor *model.User, req DeleteReq) error {
	window, err := s.repo.FindByID(ctx, req.ID)
	if err != nil {
		return err
	}
	if window == nil {
		return errors.New("窗口不存在")
	}
	account := actorAccount(actor)
	if window.CreatedBy != account {
		return errors.New("只有创建人可以删除")
	}
	// TODO: 如果窗口已关联需求，不允许删除
	return s.repo.Delete(ctx, req.ID)
}

func (s *Service) saveWindowProducts(ctx context.Context, txRepo *Repo, windowID uint64, window *model.VersionWindow, products []WindowProductInput, account string) error {
	endDate := window.ReleaseDate.Format("2006-01-02")
	beginDate := endDate
	if window.StartDate != nil {
		beginDate = window.StartDate.Format("2006-01-02")
	}

	for _, product := range products {
		if product.ProductID == 0 {
			continue
		}

		plans, err := txRepo.GetMatchingPlans(ctx, product.ProductID, endDate)
		if err != nil {
			return fmt.Errorf("get matching plans for product %d: %w", product.ProductID, err)
		}

		var planID *uint
		planSynced := uint8(0)

		if len(plans) > 0 {
			id := plans[0].ID
			planID = &id
			planSynced = 1
		} else if product.SyncPlan {
			title := strings.TrimSpace(product.PlanTitle)
			newID, err := txRepo.CreateProductPlan(ctx, product.ProductID, title, beginDate, endDate, account)
			if err != nil {
				return fmt.Errorf("create product plan for product %d: %w", product.ProductID, err)
			}
			planID = &newID
			planSynced = 1
		}

		wp := &model.VersionWindowProduct{
			WindowID:   windowID,
			ProductID:  product.ProductID,
			PlanID:     planID,
			PlanSynced: planSynced,
			CreatedBy:  account,
			UpdatedBy:  account,
		}
		if err := txRepo.CreateWindowProduct(ctx, wp); err != nil {
			return fmt.Errorf("create window product for product %d: %w", product.ProductID, err)
		}
	}
	return nil
}

// GetMatchingPlans 根据产品 ID 和结束日期查询匹配计划。
func (s *Service) GetMatchingPlans(ctx context.Context, actor *model.User, req MatchingPlansReq) (*MatchingPlansResp, error) {
	_ = actor
	plans, err := s.repo.GetMatchingPlans(ctx, req.ProductID, strings.TrimSpace(req.EndDate))
	if err != nil {
		return nil, err
	}
	items := make([]MatchingPlanItem, 0, len(plans))
	for _, plan := range plans {
		items = append(items, MatchingPlanItem{
			ID:    plan.ID,
			Title: plan.Title,
			Begin: plan.Begin,
			End:   plan.End,
		})
	}
	return &MatchingPlansResp{
		Plans:    items,
		HasMatch: len(items) > 0,
	}, nil
}

// ListBizDemands 查询排期工作台业务需求 Tab 列表。
func (s *Service) ListBizDemands(ctx context.Context, actor *model.User, req ListBizDemandsReq) (*ListBizDemandsResp, error) {
	account := actorAccount(actor)
	if account == "" {
		return &ListBizDemandsResp{Total: 0, Items: []BizDemandItem{}}, nil
	}

	normalizeBizDemandPage(&req)

	poolIDs, err := s.repo.GetUserDemandPools(ctx, account)
	if err != nil {
		return nil, err
	}
	if len(poolIDs) == 0 {
		return &ListBizDemandsResp{Total: 0, Items: []BizDemandItem{}}, nil
	}

	topDemands, total, err := s.repo.ListBizDemands(ctx, req, poolIDs)
	if err != nil {
		return nil, err
	}
	if len(topDemands) == 0 {
		return &ListBizDemandsResp{Total: total, Items: []BizDemandItem{}}, nil
	}

	topIDs := pluckDemandIDs(topDemands)

	childDemands, err := s.repo.FindChildDemandsByParents(ctx, topIDs)
	if err != nil {
		return nil, err
	}
	childByParent := groupChildDemandsByParent(childDemands)

	allDemandIDs := mergeDemandIDs(topIDs, pluckDemandIDs(childDemands))

	stories, err := s.repo.FindStoriesByDemands(ctx, allDemandIDs)
	if err != nil {
		return nil, err
	}
	storiesByDemand := groupStoriesByFromDemand(stories)

	productCountByDemand, err := s.repo.CountClarifyProductsByDemands(ctx, allDemandIDs)
	if err != nil {
		return nil, err
	}
	clarifyPMsByDemand, err := s.repo.FindClarifyPMsByDemands(ctx, allDemandIDs)
	if err != nil {
		return nil, err
	}

	productIDs := collectBizDemandProductIDs(topDemands, childDemands, stories)
	storyIDs := pluckStoryIDs(stories)
	teamgroupIDs := collectBizDemandTeamgroupIDs(topDemands)
	accounts := collectBizDemandAccounts(stories, clarifyPMsByDemand)

	productNameByID, err := s.repo.FindProductsByIDs(ctx, productIDs)
	if err != nil {
		return nil, err
	}
	windowByStory, err := s.repo.FindStoryWindowMappings(ctx, storyIDs)
	if err != nil {
		return nil, err
	}
	taskStatByStory, err := s.repo.CountStoryTasks(ctx, storyIDs)
	if err != nil {
		return nil, err
	}
	teamgroupNameByID, err := s.loadTeamgroupDisplayNamesByIDs(ctx, teamgroupIDs)
	if err != nil {
		return nil, err
	}
	realnameByAccount, err := s.repo.FindUsersByAccounts(ctx, accounts)
	if err != nil {
		return nil, err
	}

	assembleCtx := bizDemandAssembleContext{
		childByParent:        childByParent,
		storiesByDemand:      storiesByDemand,
		productCountByDemand: productCountByDemand,
		clarifyPMsByDemand:   clarifyPMsByDemand,
		productNameByID:      productNameByID,
		windowByStory:        windowByStory,
		taskStatByStory:      taskStatByStory,
		teamgroupNameByID:    teamgroupNameByID,
		realnameByAccount:    realnameByAccount,
	}

	items := make([]BizDemandItem, 0, len(topDemands))
	for _, top := range topDemands {
		items = append(items, assembleCtx.buildBizDemandItem(top))
	}

	// TODO: req.Stage / req.Scope 等 Service 层过滤

	return &ListBizDemandsResp{Total: total, Items: items}, nil
}

func normalizeBizDemandPage(req *ListBizDemandsReq) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
}

type bizDemandAssembleContext struct {
	childByParent        map[uint][]ZtDemand
	storiesByDemand      map[uint][]ZtStory
	productCountByDemand map[uint]int
	clarifyPMsByDemand   map[uint][]ClarifyPM
	productNameByID      map[uint]string
	windowByStory        map[uint]StoryWindowRef
	taskStatByStory      map[uint]StoryTaskStat
	teamgroupNameByID    map[uint]string
	realnameByAccount    map[string]string
}

func (c bizDemandAssembleContext) buildBizDemandItem(top ZtDemand) BizDemandItem {
	children := c.childByParent[top.ID]
	subtreeStories := collectSubtreeStories(top.ID, children, c.storiesByDemand)
	mainSystemStories := filterMainSystemStories(subtreeStories)
	teamgroupName := c.teamgroupName(top.TeamGroup)

	return BizDemandItem{
		ID:               top.ID,
		Name:             strings.TrimSpace(top.Name),
		Pri:              parseDemandPri(top.Pri),
		Status:           strings.TrimSpace(top.Status),
		MainSystemName:   c.productNameByID[parseUintString(top.MainSystem)],
		ExtraSystemCount: extraSystemCount(c.productCountByDemand[top.ID]),
		TeamgroupName:    teamgroupName,
		PMs:              resolvePMNames(c.clarifyPMsByDemand[top.ID], c.realnameByAccount),
		Stage:            calcBizDemandStage(subtreeStories, mainSystemStories, c.windowByStory, c.taskStatByStory),
		WindowName:       pickBizWindowName(subtreeStories, c.windowByStory),
		Children:         c.buildSubDemandItems(top, children),
		Stories:          c.buildStoryItems(top.TeamGroup, teamgroupName, c.storiesByDemand[top.ID]),
	}
}

func (c bizDemandAssembleContext) buildSubDemandItems(parent ZtDemand, children []ZtDemand) []SubDemandItem {
	if len(children) == 0 {
		return []SubDemandItem{}
	}
	parentTeamgroupName := c.teamgroupName(parent.TeamGroup)
	items := make([]SubDemandItem, 0, len(children))
	for _, child := range children {
		childStories := c.storiesByDemand[child.ID]
		subtreeStories := append([]ZtStory(nil), childStories...)
		items = append(items, SubDemandItem{
			ID:               child.ID,
			Name:             strings.TrimSpace(child.Name),
			Pri:              parseDemandPri(child.Pri),
			Status:           strings.TrimSpace(child.Status),
			MainSystemName:   c.productNameByID[parseUintString(child.MainSystem)],
			ExtraSystemCount: extraSystemCount(c.productCountByDemand[child.ID]),
			TeamgroupName:    parentTeamgroupName,
			PMs:              resolvePMNames(c.clarifyPMsByDemand[child.ID], c.realnameByAccount),
			WindowName:       pickBizWindowName(subtreeStories, c.windowByStory),
			Stories:          c.buildStoryItems(parent.TeamGroup, parentTeamgroupName, childStories),
		})
	}
	return items
}

func (c bizDemandAssembleContext) buildStoryItems(teamGroup, teamgroupName string, stories []ZtStory) []StoryItem {
	if len(stories) == 0 {
		return []StoryItem{}
	}
	_ = teamGroup
	items := make([]StoryItem, 0, len(stories))
	for _, story := range stories {
		windowRef := c.windowByStory[story.ID]
		taskStat := c.taskStatByStory[story.ID]
		assignedTo := strings.TrimSpace(story.AssignedTo)
		items = append(items, StoryItem{
			ID:                      story.ID,
			Title:                   strings.TrimSpace(story.Title),
			Pri:                     story.Pri,
			ProductName:             c.productNameByID[story.Product],
			Stage:                   calcStoryStage(story.ID, c.windowByStory),
			WindowName:              windowRef.WindowName,
			TeamgroupName:           teamgroupName,
			AssignedTo:              assignedTo,
			AssignedToName:          resolveRealname(assignedTo, c.realnameByAccount),
			TaskCount:               taskStat.Total,
			IsMainSystemAssociation: story.IsMainSystemAssociation,
		})
	}
	return items
}

func (c bizDemandAssembleContext) teamgroupName(teamGroup string) string {
	return c.teamgroupNameByID[parseUintString(teamGroup)]
}

func (s *Service) loadTeamgroupDisplayNamesByIDs(ctx context.Context, teamgroupIDs []uint) (map[uint]string, error) {
	if len(teamgroupIDs) == 0 {
		return map[uint]string{}, nil
	}

	groups, err := s.repo.FindTeamgroupsByIDs(ctx, teamgroupIDs)
	if err != nil {
		return nil, err
	}
	nameByID := make(map[uint]string, len(groups))
	parentIDs := make([]uint, 0)
	parentSeen := make(map[uint]struct{})
	for _, group := range groups {
		nameByID[group.ID] = strings.TrimSpace(group.Name)
		if group.Parent == 0 {
			continue
		}
		if _, ok := parentSeen[group.Parent]; ok {
			continue
		}
		parentSeen[group.Parent] = struct{}{}
		parentIDs = append(parentIDs, group.Parent)
	}
	if len(parentIDs) > 0 {
		parents, err := s.repo.FindTeamgroupsByIDs(ctx, parentIDs)
		if err != nil {
			return nil, err
		}
		parentNameByID := make(map[uint]string, len(parents))
		for _, parent := range parents {
			parentNameByID[parent.ID] = strings.TrimSpace(parent.Name)
		}
		for _, group := range groups {
			name := strings.TrimSpace(group.Name)
			if group.Parent > 0 {
				if parentName := parentNameByID[group.Parent]; parentName != "" {
					name = fmt.Sprintf("%s / %s", parentName, name)
				}
			}
			nameByID[group.ID] = name
		}
	}
	return nameByID, nil
}

func pluckDemandIDs(demands []ZtDemand) []uint {
	ids := make([]uint, 0, len(demands))
	for _, demand := range demands {
		if demand.ID == 0 {
			continue
		}
		ids = append(ids, demand.ID)
	}
	return ids
}

func mergeDemandIDs(a, b []uint) []uint {
	if len(b) == 0 {
		return append([]uint(nil), a...)
	}
	out := make([]uint, 0, len(a)+len(b))
	seen := make(map[uint]struct{}, len(a)+len(b))
	for _, id := range append(a, b...) {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func groupChildDemandsByParent(children []ZtDemand) map[uint][]ZtDemand {
	out := make(map[uint][]ZtDemand, len(children))
	for _, child := range children {
		out[child.Parent] = append(out[child.Parent], child)
	}
	return out
}

func groupStoriesByFromDemand(stories []ZtStory) map[uint][]ZtStory {
	out := make(map[uint][]ZtStory, len(stories))
	for _, story := range stories {
		out[story.FromDemand] = append(out[story.FromDemand], story)
	}
	return out
}

func collectSubtreeStories(topID uint, children []ZtDemand, storiesByDemand map[uint][]ZtStory) []ZtStory {
	out := append([]ZtStory(nil), storiesByDemand[topID]...)
	for _, child := range children {
		out = append(out, storiesByDemand[child.ID]...)
	}
	return out
}

func filterMainSystemStories(stories []ZtStory) []ZtStory {
	out := make([]ZtStory, 0, len(stories))
	for _, story := range stories {
		if story.IsMainSystemAssociation == 1 {
			out = append(out, story)
		}
	}
	return out
}

func calcBizDemandStage(
	allStories []ZtStory,
	mainStories []ZtStory,
	windowByStory map[uint]StoryWindowRef,
	taskStatByStory map[uint]StoryTaskStat,
) string {
	if len(allStories) == 0 {
		return StageNoStory
	}
	if allStoriesHaveNoWindow(allStories, windowByStory) {
		return StageNoWindow
	}

	taskTotal, unassignedTotal := sumMainSystemTasks(mainStories, taskStatByStory)
	if taskTotal == 0 {
		return StageNoTask
	}
	if unassignedTotal > 0 {
		return StageTaskUnassigned
	}
	return StageTaskAssigned
}

func allStoriesHaveNoWindow(stories []ZtStory, windowByStory map[uint]StoryWindowRef) bool {
	for _, story := range stories {
		if ref, ok := windowByStory[story.ID]; ok && ref.WindowID > 0 {
			return false
		}
	}
	return true
}

func sumMainSystemTasks(stories []ZtStory, taskStatByStory map[uint]StoryTaskStat) (int, int) {
	taskTotal := 0
	unassignedTotal := 0
	for _, story := range stories {
		stat := taskStatByStory[story.ID]
		taskTotal += stat.Total
		unassignedTotal += stat.Unassigned
	}
	return taskTotal, unassignedTotal
}

func calcStoryStage(storyID uint, windowByStory map[uint]StoryWindowRef) string {
	if ref, ok := windowByStory[storyID]; ok && ref.WindowID > 0 {
		return StoryStageHasWindow
	}
	return StoryStageNoWindow
}

func pickBizWindowName(stories []ZtStory, windowByStory map[uint]StoryWindowRef) string {
	for _, story := range stories {
		ref, ok := windowByStory[story.ID]
		if !ok || ref.WindowID == 0 {
			continue
		}
		name := strings.TrimSpace(ref.WindowName)
		if name != "" {
			return name
		}
	}
	return ""
}

func extraSystemCount(distinctProductCount int) int {
	if distinctProductCount <= 1 {
		return 0
	}
	return distinctProductCount - 1
}

func resolvePMNames(clarifyPMs []ClarifyPM, realnameByAccount map[string]string) []string {
	if len(clarifyPMs) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(clarifyPMs))
	names := make([]string, 0, len(clarifyPMs))
	for _, item := range clarifyPMs {
		account := strings.TrimSpace(item.PM)
		if account == "" {
			continue
		}
		if _, ok := seen[account]; ok {
			continue
		}
		seen[account] = struct{}{}
		names = append(names, resolveRealname(account, realnameByAccount))
	}
	return names
}

func resolveRealname(account string, realnameByAccount map[string]string) string {
	account = strings.TrimSpace(account)
	if account == "" {
		return ""
	}
	if name := strings.TrimSpace(realnameByAccount[account]); name != "" {
		return name
	}
	return account
}

func collectBizDemandProductIDs(topDemands, childDemands []ZtDemand, stories []ZtStory) []uint {
	seen := make(map[uint]struct{})
	ids := make([]uint, 0)
	add := func(id uint) {
		if id == 0 {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for _, demand := range append(topDemands, childDemands...) {
		add(parseUintString(demand.MainSystem))
	}
	for _, story := range stories {
		add(story.Product)
	}
	return ids
}

func collectBizDemandTeamgroupIDs(topDemands []ZtDemand) []uint {
	seen := make(map[uint]struct{})
	ids := make([]uint, 0, len(topDemands))
	for _, demand := range topDemands {
		id := parseUintString(demand.TeamGroup)
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func collectBizDemandAccounts(stories []ZtStory, clarifyPMsByDemand map[uint][]ClarifyPM) []string {
	seen := make(map[string]struct{})
	accounts := make([]string, 0)
	add := func(account string) {
		account = strings.TrimSpace(account)
		if account == "" {
			return
		}
		if _, ok := seen[account]; ok {
			return
		}
		seen[account] = struct{}{}
		accounts = append(accounts, account)
	}
	for _, story := range stories {
		add(story.AssignedTo)
	}
	for _, clarifyPMs := range clarifyPMsByDemand {
		for _, item := range clarifyPMs {
			add(item.PM)
		}
	}
	return accounts
}

func pluckStoryIDs(stories []ZtStory) []uint {
	ids := make([]uint, 0, len(stories))
	for _, story := range stories {
		if story.ID == 0 {
			continue
		}
		ids = append(ids, story.ID)
	}
	return ids
}

func parseDemandPri(pri string) int {
	value, err := strconv.Atoi(strings.TrimSpace(pri))
	if err != nil {
		return 0
	}
	return value
}

func parseUintString(raw string) uint {
	value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return uint(value)
}
