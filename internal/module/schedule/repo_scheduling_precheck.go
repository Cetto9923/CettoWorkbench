// =============================================================================
// 文件: internal/module/schedule/repo_scheduling_precheck.go
// 模块: 排期工作台
// 类型: action
// 职责: 排期一体化「确认并同步」写前查询/校验(写前预检,不是写)。
// 依赖: internal/model
// =============================================================================

package schedule

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"workbench/internal/model"
)

// FindWindowProductPlan 查窗口下某产品的关联计划。
func (r *Repo) FindWindowProductPlan(ctx context.Context, windowID uint, productID uint) (*model.VersionWindowProduct, error) {
	if windowID == 0 || productID == 0 {
		return nil, nil
	}
	var row model.VersionWindowProduct
	err := r.db.WithContext(ctx).
		Where("versionWindow = ? AND product = ?", uint64(windowID), productID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// GetProjectIDByExecution 从 executionID 沿 parent 链推导 projectID。
func (r *Repo) GetProjectIDByExecution(ctx context.Context, executionID uint) (uint, error) {
	if executionID == 0 {
		return 0, errors.New("执行 ID 无效")
	}
	current := executionID
	for i := 0; i < 20; i++ {
		var row struct {
			ID     uint   `gorm:"column:id"`
			Parent uint   `gorm:"column:parent"`
			Type   string `gorm:"column:type"`
		}
		err := r.db.WithContext(ctx).
			Raw(`SELECT id, parent, type FROM zt_project WHERE id = ? AND deleted = '0' LIMIT 1`, current).
			Scan(&row).Error
		if err != nil {
			return 0, err
		}
		if row.ID == 0 {
			return 0, errors.New("执行不存在")
		}
		if strings.TrimSpace(row.Type) == "project" {
			return row.ID, nil
		}
		if row.Parent == 0 {
			return 0, errors.New("未找到所属项目")
		}
		current = row.Parent
	}
	return 0, errors.New("项目层级过深")
}

// UserCanAccessProduct 判断用户是否有权操作指定产品/系统。
func (r *Repo) UserCanAccessProduct(ctx context.Context, account string, productID uint) (bool, error) {
	account = strings.TrimSpace(account)
	if account == "" || productID == 0 {
		return false, nil
	}
	isAdmin, err := r.IsAdmin(ctx, account)
	if err != nil {
		return false, err
	}
	if isAdmin {
		return true, nil
	}
	products, err := r.GetUserProducts(ctx, account)
	if err != nil {
		return false, err
	}
	for _, product := range products {
		if product.ID == productID {
			return true, nil
		}
	}
	return false, nil
}

// GetDemandMainSystem 查询业需主系统 ID。
func (r *Repo) GetDemandMainSystem(ctx context.Context, demandID uint) (uint, error) {
	if demandID == 0 {
		return 0, errors.New("业需 ID 无效")
	}
	const query = `SELECT mainSystem FROM zt_demand WHERE id = ? AND deleted = '0' LIMIT 1`
	var mainSystem string
	if err := r.db.WithContext(ctx).Raw(query, demandID).Scan(&mainSystem).Error; err != nil {
		return 0, err
	}
	return parseUintString(mainSystem), nil
}

// GetStoryProductID 查询独立研发需求的主系统 ID（zt_story.product）。
func (r *Repo) GetStoryProductID(ctx context.Context, storyID uint) (uint, error) {
	if storyID == 0 {
		return 0, errors.New("研发需求 ID 无效")
	}
	const query = `SELECT product FROM zt_story WHERE id = ? AND deleted = '0' LIMIT 1`
	var product uint
	if err := r.db.WithContext(ctx).Raw(query, storyID).Scan(&product).Error; err != nil {
		return 0, err
	}
	return product, nil
}
