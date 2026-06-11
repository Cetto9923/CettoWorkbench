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
	"log"
	"strings"
	"time"

	"workbench/internal/model"
)

var windowCardToneClasses = []string{"red", "blue", "green", "purple"}

// Service 处理排期业务逻辑。
type Service struct {
	repo *Repo
}

// NewService 创建 Service。
func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

// GetUserTeamgroups 查询用户所属敏捷小组并拼接展示名称。
func (s *Service) GetUserTeamgroups(ctx context.Context, account string) ([]TeamgroupOption, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return []TeamgroupOption{}, nil
	}

	groups, err := s.repo.GetUserTeamgroups(ctx, account)
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

// GetCreateWindowFormData 查询新建版本窗口弹窗所需表单数据。
func (s *Service) GetCreateWindowFormData(ctx context.Context, account string) (*CreateWindowFormData, error) {
	teamgroups, err := s.GetUserTeamgroups(ctx, account)
	if err != nil {
		return nil, err
	}
	account = strings.TrimSpace(account)
	products, err := s.repo.GetUserProducts(ctx, account)
	if err != nil {
		return nil, err
	}
	if products == nil {
		products = []ZtProduct{}
	}
	// TODO: 临时调试日志，确认产品匹配逻辑后删除
	for _, p := range products {
		matchedBy := []string{}
		if p.PO == account {
			matchedBy = append(matchedBy, "PO")
		}
		if p.QD == account {
			matchedBy = append(matchedBy, "QD")
		}
		if p.RD == account {
			matchedBy = append(matchedBy, "RD")
		}
		if p.CreatedBy == account {
			matchedBy = append(matchedBy, "createdBy")
		}
		if strings.Contains(","+p.Whitelist+",", ","+account+",") {
			matchedBy = append(matchedBy, "whitelist")
		}
		if strings.Contains(","+p.PMT+",", ","+account+",") {
			matchedBy = append(matchedBy, "PMT")
		}
		log.Printf("[DEBUG] 产品 %d %s 匹配方式: %v", p.ID, p.Name, matchedBy)
	}
	log.Printf("[DEBUG] GetUserProducts account=%s 共 %d 个产品", account, len(products))
	return &CreateWindowFormData{
		Teamgroups: teamgroups,
		Products:   products,
	}, nil
}

func computeWindowPermissions(createdBy, account string) (canEdit, canDelete, hasLinkedDemands bool) {
	hasLinkedDemands = false // TODO: 接需求数据后改为真实查询
	canEdit = strings.TrimSpace(createdBy) == strings.TrimSpace(account)
	canDelete = canEdit && !hasLinkedDemands
	return
}

// ListWindowCards 查询版本窗口概览卡片数据。
func (s *Service) ListWindowCards(ctx context.Context, account string) ([]WindowCard, error) {
	windows, err := s.repo.ListVersionWindows(ctx)
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
		workdays := countWorkdays(start, window.ReleaseDate)
		groupSize := window.GroupSize
		if groupSize == 0 {
			groupSize = 1
		}
		capacityHours := workdays * 7 * int(groupSize)

		canEdit, canDelete, hasLinkedDemands := computeWindowPermissions(window.CreatedBy, account)

		cards = append(cards, WindowCard{
			ID:               window.ID,
			ShortName:        window.Name,
			Range:            formatWindowDateRange(start, window.ReleaseDate),
			Status:           windowStatusLabel(window.Status),
			ToneClass:        windowCardToneClasses[i%len(windowCardToneClasses)],
			AgileGroup:       teamgroupNameByID[window.TeamgroupID],
			DemandCount:      0,
			CapacityHours:    capacityHours,
			UsedHours:        0,
			RemainingHours:   capacityHours,
			BlockedCount:     0,
			UsedPercent:      0,
			CanEdit:          canEdit,
			CanDelete:        canDelete,
			HasLinkedDemands: hasLinkedDemands,
		})
	}
	return cards, nil
}

// ListWindows 查询版本窗口维护列表。
func (s *Service) ListWindows(ctx context.Context, account string) ([]WindowListItem, error) {
	windows, err := s.repo.ListVersionWindows(ctx)
	if err != nil {
		return nil, err
	}
	if len(windows) == 0 {
		return []WindowListItem{}, nil
	}

	items := make([]WindowListItem, 0, len(windows))
	for _, window := range windows {
		start := window.ReleaseDate
		if window.StartDate != nil {
			start = *window.StartDate
		}
		workdays := countWorkdays(start, window.ReleaseDate)
		groupSize := window.GroupSize
		if groupSize == 0 {
			groupSize = 1
		}
		capacityHours := workdays * 7 * int(groupSize)
		canEdit, canDelete, hasLinkedDemands := computeWindowPermissions(window.CreatedBy, account)

		items = append(items, WindowListItem{
			ID:               window.ID,
			Name:             window.Name,
			ReleaseDate:      window.ReleaseDate.Format("2006-01-02"),
			Range:            formatWindowDateRange(start, window.ReleaseDate),
			Status:           windowStatusLabel(window.Status),
			CapacityHours:    capacityHours,
			CanEdit:          canEdit,
			CanDelete:        canDelete,
			HasLinkedDemands: hasLinkedDemands,
		})
	}
	return items, nil
}

// BuildVersionWindowFromForm 将保存请求转换为版本窗口模型。
func BuildVersionWindowFromForm(form CreateWindowForm) (*model.VersionWindow, error) {
	releaseDate, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(form.ReleaseDate), time.Local)
	if err != nil {
		return nil, fmt.Errorf("invalid release date")
	}

	window := &model.VersionWindow{
		Name:        strings.TrimSpace(form.Name),
		ReleaseDate: releaseDate,
		TeamgroupID: form.TeamgroupID,
		Status:      "planning",
	}
	if form.GroupSize > 0 {
		window.GroupSize = uint(form.GroupSize)
	} else {
		window.GroupSize = 1
	}

	startDate := strings.TrimSpace(form.StartDate)
	if startDate != "" {
		parsed, err := time.ParseInLocation("2006-01-02", startDate, time.Local)
		if err != nil {
			return nil, fmt.Errorf("invalid start date")
		}
		window.StartDate = &parsed
	}
	return window, nil
}

// ApplyFormToVersionWindow 将表单数据应用到已有版本窗口。
func ApplyFormToVersionWindow(window *model.VersionWindow, form CreateWindowForm) error {
	if window == nil {
		return fmt.Errorf("version window is nil")
	}
	updated, err := BuildVersionWindowFromForm(form)
	if err != nil {
		return err
	}
	window.Name = updated.Name
	window.ReleaseDate = updated.ReleaseDate
	window.StartDate = updated.StartDate
	window.TeamgroupID = updated.TeamgroupID
	window.GroupSize = updated.GroupSize
	return nil
}

// GetVersionWindow 按 ID 查询未删除的版本窗口。
func (s *Service) GetVersionWindow(ctx context.Context, id uint64) (*model.VersionWindow, error) {
	return s.repo.GetVersionWindowByID(ctx, id)
}

// GetWindowDetail 查询版本窗口详情。
func (s *Service) GetWindowDetail(ctx context.Context, id uint64) (*WindowDetailResp, error) {
	window, err := s.repo.GetVersionWindowByID(ctx, id)
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

func countWorkdays(start, end time.Time) int {
	if end.Before(start) {
		start, end = end, start
	}
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())

	count := 0
	for current := start; !current.After(end); current = current.AddDate(0, 0, 1) {
		weekday := current.Weekday()
		if weekday != time.Saturday && weekday != time.Sunday {
			count++
		}
	}
	if count == 0 {
		return 1
	}
	return count
}

func formatWindowDateRange(start, end time.Time) string {
	return start.Format("01-02") + " ~ " + end.Format("01-02")
}

func windowStatusLabel(status string) string {
	switch strings.TrimSpace(status) {
	case "current":
		return "当前"
	case "next":
		return "下一"
	case "released":
		return "已发布"
	default:
		return "规划中"
	}
}

// SaveWindowWithPlans 保存版本窗口并按需同步禅道产品计划。
func (s *Service) SaveWindowWithPlans(ctx context.Context, window *model.VersionWindow, products []WindowProductInput, account string) error {
	if window == nil {
		return fmt.Errorf("version window is nil")
	}
	account = strings.TrimSpace(account)
	window.CreatedBy = account

	return s.repo.Transaction(ctx, func(txRepo *Repo) error {
		if err := txRepo.CreateVersionWindow(ctx, window); err != nil {
			return fmt.Errorf("create version window: %w", err)
		}
		return s.saveWindowProducts(ctx, txRepo, window.ID, window, products, account)
	})
}

// UpdateWindowWithPlans 更新版本窗口并重建关联产品及计划。
func (s *Service) UpdateWindowWithPlans(ctx context.Context, window *model.VersionWindow, products []WindowProductInput, account string) error {
	if window == nil || window.ID == 0 {
		return fmt.Errorf("version window is invalid")
	}
	account = strings.TrimSpace(account)

	return s.repo.Transaction(ctx, func(txRepo *Repo) error {
		if err := txRepo.UpdateVersionWindow(ctx, window); err != nil {
			return fmt.Errorf("update version window: %w", err)
		}
		if err := txRepo.DeleteWindowProducts(ctx, window.ID); err != nil {
			return fmt.Errorf("delete window products: %w", err)
		}
		return s.saveWindowProducts(ctx, txRepo, window.ID, window, products, account)
	})
}

// SoftDeleteWindow 软删除版本窗口。
func (s *Service) SoftDeleteWindow(ctx context.Context, id uint64, account string) error {
	window, err := s.repo.GetVersionWindowByID(ctx, id)
	if err != nil {
		return err
	}
	if window == nil {
		return errors.New("窗口不存在")
	}
	account = strings.TrimSpace(account)
	if window.CreatedBy != account {
		return errors.New("只有创建人可以删除")
	}
	// TODO: 如果窗口已关联需求，不允许删除
	return s.repo.SoftDeleteVersionWindow(ctx, id)
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
		}
		if err := txRepo.CreateWindowProduct(ctx, wp); err != nil {
			return fmt.Errorf("create window product for product %d: %w", product.ProductID, err)
		}
	}
	return nil
}

// GetMatchingPlans 根据产品 ID 和结束日期查询匹配计划。
func (s *Service) GetMatchingPlans(ctx context.Context, req MatchingPlansReq) (*MatchingPlansResp, error) {
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
