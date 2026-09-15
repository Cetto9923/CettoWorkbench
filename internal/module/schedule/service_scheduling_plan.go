package schedule

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"workbench/internal/model"
)

func (s *Service) resolvePlanForProduct(ctx context.Context, txRepo *Repo, account string, windowID, productID uint) (uint, error) {
	vwp, err := txRepo.FindWindowProductPlan(ctx, windowID, productID)
	if err != nil {
		return 0, err
	}
	if vwp != nil && vwp.PlanID != nil && *vwp.PlanID > 0 {
		return *vwp.PlanID, nil
	}

	window, err := txRepo.FindByID(ctx, uint64(windowID))
	if err != nil {
		return 0, err
	}
	if window == nil {
		return 0, errors.New("版本窗口不存在")
	}
	endDate := window.ReleaseDate.Format("2006-01-02")
	beginDate := endDate
	if window.StartDate != nil {
		beginDate = window.StartDate.Format("2006-01-02")
	}
	title := strings.TrimSpace(window.Name)
	if title == "" {
		title = endDate
	}

	if vwp != nil {
		planID, err := txRepo.CreateProductPlan(ctx, productID, title, beginDate, endDate, account)
		if err != nil {
			return 0, fmt.Errorf("create product plan: %w", err)
		}
		if err := txRepo.UpdateWindowProductPlanID(ctx, vwp.ID, planID, account); err != nil {
			return 0, err
		}
		return planID, nil
	}

	plans, err := txRepo.GetMatchingPlans(ctx, productID, endDate)
	if err != nil {
		return 0, fmt.Errorf("get matching plans for product %d: %w", productID, err)
	}
	var planID uint
	if len(plans) > 0 {
		planID = plans[0].ID
	} else if planID, err = txRepo.CreateProductPlan(ctx, productID, title, beginDate, endDate, account); err != nil {
		return 0, fmt.Errorf("create product plan: %w", err)
	}
	wp := &model.VersionWindowProduct{
		WindowID:   uint64(windowID),
		ProductID:  productID,
		PlanID:     &planID,
		PlanSynced: 1,
		CreatedBy:  account,
		UpdatedBy:  account,
	}
	if err := txRepo.CreateWindowProduct(ctx, wp); err != nil {
		return 0, fmt.Errorf("create window product for product %d: %w", productID, err)
	}
	return planID, nil
}
