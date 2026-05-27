// =============================================================================
// 文件: internal/module/dictitem/repo.go
// 模块: 字典值
// 类型: crud
// 职责: 封装字典值数据访问。
// 依赖: internal/model
// =============================================================================

package dictitem

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"workbrench/internal/model"
)

// Repo 封装字典值数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// RepoFindAllReq 列表查询参数。
type RepoFindAllReq struct {
	TypeCode string
}

// FindAll 查询字典值列表（count + find）。
func (r *Repo) FindAll(ctx context.Context, req RepoFindAllReq) ([]model.DictItem, int64, error) {
	q := r.db.WithContext(ctx).
		Model(&model.DictItem{}).
		Where("deletedAt IS NULL")
	if strings.TrimSpace(req.TypeCode) != "" {
		q = q.Where("typeCode = ?", strings.TrimSpace(req.TypeCode))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []model.DictItem
	if err := q.Order("sort ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// FindByID 按 ID 查询字典值。
func (r *Repo) FindByID(ctx context.Context, id uint64) (*model.DictItem, error) {
	var row model.DictItem
	if err := r.db.WithContext(ctx).
		Model(&model.DictItem{}).
		Where("id = ? AND deletedAt IS NULL", id).
		First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Create 创建字典值。
func (r *Repo) Create(ctx context.Context, m *model.DictItem) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// Update 更新字典值。
func (r *Repo) Update(ctx context.Context, m *model.DictItem) error {
	return r.db.WithContext(ctx).
		Model(&model.DictItem{}).
		Where("id = ? AND deletedAt IS NULL", m.ID).
		Updates(map[string]any{
			"typeCode":  m.TypeCode,
			"label":     m.Label,
			"value":     m.Value,
			"sort":      m.Sort,
			"updatedAt": m.UpdatedAt,
		}).Error
}

// Delete 删除字典值（软删除）。
func (r *Repo) Delete(ctx context.Context, id uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.DictItem{}).
		Where("id = ? AND deletedAt IS NULL", id).
		Updates(map[string]any{
			"deletedAt": now,
			"updatedAt": now,
		}).Error
}
