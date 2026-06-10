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
	"fmt"
	"log"
	"strings"

	"workbench/internal/model"
)

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

// SaveWindowWithPlans 保存版本窗口并按需同步禅道产品计划。
func (s *Service) SaveWindowWithPlans(ctx context.Context, window *model.VersionWindow, products []WindowProductInput, account string) error {
	if window == nil {
		return fmt.Errorf("version window is nil")
	}
	account = strings.TrimSpace(account)
	window.CreatedBy = account

	endDate := window.ReleaseDate.Format("2006-01-02")
	beginDate := endDate
	if window.StartDate != nil {
		beginDate = window.StartDate.Format("2006-01-02")
	}

	return s.repo.Transaction(ctx, func(txRepo *Repo) error {
		if err := txRepo.CreateVersionWindow(ctx, window); err != nil {
			return fmt.Errorf("create version window: %w", err)
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
				WindowID:   window.ID,
				ProductID:  product.ProductID,
				PlanID:     planID,
				PlanSynced: planSynced,
			}
			if err := txRepo.CreateWindowProduct(ctx, wp); err != nil {
				return fmt.Errorf("create window product for product %d: %w", product.ProductID, err)
			}
		}
		return nil
	})
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
