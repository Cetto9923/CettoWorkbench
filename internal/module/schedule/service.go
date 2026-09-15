// =============================================================================
// 文件: internal/module/schedule/service.go
// 模块: 排期工作台
// 类型: action
// 职责: 版本窗口 CRUD 与表单数据、匹配计划查询。
// 依赖: internal/model
//       internal/module/schedule/repo.go
//       internal/module/schedule/service_window.go
// =============================================================================

package schedule

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"workbench/internal/model"
)

// Service 处理排期业务逻辑。
type Service struct {
	repo   *Repo
	logger *zap.Logger
}

// NewService 创建 Service。
func NewService(repo *Repo, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger}
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

// ListFilterProducts 查询筛选区全部产品/系统列表，当前用户参与的系统排序在前。
func (s *Service) ListFilterProducts(ctx context.Context, actor *model.User) ([]ZtProduct, error) {
	products, err := s.repo.ListAllProducts(ctx)
	if err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return []ZtProduct{}, nil
	}
	account := actorAccount(actor)
	if account != "" {
		userProducts, err := s.repo.GetUserProducts(ctx, account)
		if err == nil && len(userProducts) > 0 {
			userMap := make(map[uint]bool, len(userProducts))
			for _, up := range userProducts {
				userMap[up.ID] = true
			}
			for i := range products {
				if userMap[products[i].ID] {
					products[i].IsMyProduct = true
				}
			}
			sort.SliceStable(products, func(i, j int) bool {
				if products[i].IsMyProduct != products[j].IsMyProduct {
					return products[i].IsMyProduct
				}
				return false
			})
		}
	}
	return products, nil
}

// ListFilterWindows 查询筛选区全部版本窗口列表。
func (s *Service) ListFilterWindows(ctx context.Context) ([]WindowFilterOption, error) {
	windows, _, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	if len(windows) == 0 {
		return []WindowFilterOption{}, nil
	}
	out := make([]WindowFilterOption, 0, len(windows))
	for _, window := range windows {
		out = append(out, WindowFilterOption{
			ID:   uint(window.ID),
			Name: strings.TrimSpace(window.Name),
		})
	}
	return out, nil
}

// ListScheduleUsers 查询筛选区负责人下拉用户。
func (s *Service) ListScheduleUsers(ctx context.Context, actor *model.User) ([]SchedulingUserOption, error) {
	users, err := s.repo.ListInsideUsersForScheduling(ctx, actorAccount(actor))
	if err != nil {
		return nil, err
	}
	if users == nil {
		return []SchedulingUserOption{}, nil
	}
	return users, nil
}

func dateOnly(value time.Time) time.Time {
	value = value.In(time.Local)
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
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

func parseOptionalWindowMilestoneDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return nil, err
	}
	d := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, parsed.Location())
	return &d, nil
}

func buildVersionWindowMilestone(windowID uint64, account, planTestDone, testDone, acceptDone string) (*model.VersionWindowMilestone, error) {
	planTest, err := parseOptionalWindowMilestoneDate(planTestDone)
	if err != nil {
		return nil, fmt.Errorf("invalid plan test done date")
	}
	test, err := parseOptionalWindowMilestoneDate(testDone)
	if err != nil {
		return nil, fmt.Errorf("invalid test done date")
	}
	accept, err := parseOptionalWindowMilestoneDate(acceptDone)
	if err != nil {
		return nil, fmt.Errorf("invalid accept done date")
	}
	return &model.VersionWindowMilestone{
		WindowID:     windowID,
		PlanTestDone: planTest,
		TestDone:     test,
		AcceptDone:   accept,
		CreatedBy:    account,
		UpdatedBy:    account,
	}, nil
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

	milestone, err := s.repo.GetWindowMilestone(ctx, id)
	if err != nil {
		return nil, err
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
	if milestone != nil {
		detail.PlanTestDone = milestone.PlanTestDone
		detail.TestDone = milestone.TestDone
		detail.AcceptDone = milestone.AcceptDone
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

func collectWindowProductIDs(products []WindowProductInput) []uint {
	raw := make([]uint, 0, len(products))
	for _, p := range products {
		if p.ProductID != 0 {
			raw = append(raw, p.ProductID)
		}
	}
	return uniqueUints(raw)
}

// Create 保存版本窗口并按需同步禅道产品计划。
func (s *Service) Create(ctx context.Context, actor *model.User, req CreateReq) (*model.VersionWindow, error) {
	window, err := buildVersionWindowFromCreateReq(req)
	if err != nil {
		return nil, err
	}
	account := actorAccount(actor)
	window.CreatedBy = account
	window.UpdatedBy = account

	exists, err := s.repo.ExistsWindowName(ctx, window.Name, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("版本窗口名称已存在，请更换名称")
	}

	productIDs := collectWindowProductIDs(req.Products)
	notice, err := s.validateProductsAccess(ctx, productIDs, account)
	if err != nil {
		return nil, err
	}
	if notice != nil {
		return nil, notice
	}

	err = s.repo.Transaction(ctx, func(txRepo *Repo) error {
		if err := txRepo.Create(ctx, window); err != nil {
			return fmt.Errorf("create version window: %w", err)
		}
		milestone, err := buildVersionWindowMilestone(window.ID, account, req.PlanTestDone, req.TestDone, req.AcceptDone)
		if err != nil {
			return err
		}
		if err := txRepo.UpsertWindowMilestone(ctx, milestone); err != nil {
			return fmt.Errorf("save version window milestone: %w", err)
		}
		return s.saveWindowProducts(ctx, txRepo, window.ID, window, req.Products, account)
	})
	if err != nil {
		return nil, err
	}
	return window, nil
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
	account := actorAccount(actor)
	if window.CreatedBy != account {
		return errors.New("只有创建人可以修改")
	}
	if err := applyUpdateReqToVersionWindow(window, req); err != nil {
		return err
	}
	window.UpdatedBy = account

	exists, err := s.repo.ExistsWindowName(ctx, window.Name, req.ID)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("版本窗口名称已存在，请更换名称")
	}

	productIDs := collectWindowProductIDs(req.Products)
	notice, err := s.validateProductsAccess(ctx, productIDs, account)
	if err != nil {
		return err
	}
	if notice != nil {
		return notice
	}

	return s.repo.Transaction(ctx, func(txRepo *Repo) error {
		if err := txRepo.Update(ctx, window); err != nil {
			return fmt.Errorf("update version window: %w", err)
		}
		milestone, err := buildVersionWindowMilestone(window.ID, account, req.PlanTestDone, req.TestDone, req.AcceptDone)
		if err != nil {
			return err
		}
		if err := txRepo.UpsertWindowMilestone(ctx, milestone); err != nil {
			return fmt.Errorf("save version window milestone: %w", err)
		}
		if err := txRepo.DeleteWindowProducts(ctx, window.ID); err != nil {
			return fmt.Errorf("delete window products: %w", err)
		}
		return s.saveWindowProducts(ctx, txRepo, window.ID, window, req.Products, account)
	})
}

// Delete 软删除版本窗口。
// 业务不变量：窗口若已被任何需求（zt_demandwindow）或产品/计划（zt_versionwindowproduct）
// 关联，则拒绝删除，避免产生孤儿关联记录。
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
	assoc, err := s.repo.CountWindowAssociations(ctx, req.ID)
	if err != nil {
		return err
	}
	if assoc > 0 {
		return fmt.Errorf("版本窗口已关联 %d 条需求/产品记录，无法删除", assoc)
	}
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
