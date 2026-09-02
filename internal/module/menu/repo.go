// =============================================================================
// 文件: internal/module/menu/repo.go
// 模块: 菜单管理
// 类型: crud
// 职责: 封装菜单数据访问。
// 依赖: internal/model
// =============================================================================

package menu

import (
	"context"
	"time"

	"gorm.io/gorm"

	"workbench/internal/model"
)

// Repo 封装菜单数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// FindAll 查询全部未删除菜单（分页三元组返回）。
func (r *Repo) FindAll(ctx context.Context, req RepoFindAllReq) ([]model.Menu, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&model.Menu{}).
		Where("deletedAt IS NULL")

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []model.Menu
	if err := query.
		Order("sort ASC, id ASC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// FindByID 按 ID 查询菜单。
func (r *Repo) FindByID(ctx context.Context, id uint64) (*model.Menu, error) {
	var row model.Menu
	if err := r.db.WithContext(ctx).
		Model(&model.Menu{}).
		Where("id = ? AND deletedAt IS NULL", id).
		First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Create 创建菜单。
func (r *Repo) Create(ctx context.Context, m *model.Menu) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// Update 更新菜单。
func (r *Repo) Update(ctx context.Context, m *model.Menu) error {
	return r.db.WithContext(ctx).
		Model(&model.Menu{}).
		Where("id = ? AND deletedAt IS NULL", m.ID).
		Updates(map[string]any{
			"parentId":  m.ParentID,
			"type":      m.Type,
			"title":     m.Title,
			"icon":      m.Icon,
			"path":      m.Path,
			"perm":      m.Perm,
			"sort":      m.Sort,
			"updatedAt": m.UpdatedAt,
		}).Error
}

// Delete 软删除菜单。
func (r *Repo) Delete(ctx context.Context, id uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.Menu{}).
		Where("id = ? AND deletedAt IS NULL", id).
		Updates(map[string]any{
			"deletedAt": now,
			"updatedAt": now,
		}).Error
}

// FindChildren 查询某父菜单下的直接子菜单。
func (r *Repo) FindChildren(ctx context.Context, parentID uint64) ([]*model.Menu, error) {
	var rows []*model.Menu
	err := r.db.WithContext(ctx).
		Model(&model.Menu{}).
		Where("parentId = ? AND deletedAt IS NULL", parentID).
		Order("sort ASC, id ASC").
		Find(&rows).Error
	return rows, err
}
