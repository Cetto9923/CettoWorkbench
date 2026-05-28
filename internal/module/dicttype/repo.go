// =============================================================================
// 文件: internal/module/dicttype/repo.go
// 模块: 字典类型
// 类型: crud
// 职责: 封装字典类型数据访问。
// 依赖: internal/model
// =============================================================================

package dicttype

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"workbench/internal/model"
)

// Repo 封装字典类型数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// RepoFindAllReq 列表查询参数。
type RepoFindAllReq struct {
	Keyword string
}

// FindAll 查询字典类型列表（count + find）。
func (r *Repo) FindAll(ctx context.Context, req RepoFindAllReq) ([]model.DictType, int64, error) {
	q := r.db.WithContext(ctx).
		Model(&model.DictType{}).
		Where("deletedAt IS NULL")
	if strings.TrimSpace(req.Keyword) != "" {
		like := "%" + strings.TrimSpace(req.Keyword) + "%"
		q = q.Where("(code LIKE ? OR name LIKE ?)", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []model.DictType
	if err := q.Order("id DESC").Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// FindByID 按 ID 查询字典类型。
func (r *Repo) FindByID(ctx context.Context, id uint64) (*model.DictType, error) {
	var row model.DictType
	if err := r.db.WithContext(ctx).
		Model(&model.DictType{}).
		Where("id = ? AND deletedAt IS NULL", id).
		First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Create 创建字典类型。
func (r *Repo) Create(ctx context.Context, m *model.DictType) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// Update 更新字典类型。
func (r *Repo) Update(ctx context.Context, m *model.DictType) error {
	return r.db.WithContext(ctx).
		Model(&model.DictType{}).
		Where("id = ? AND deletedAt IS NULL", m.ID).
		Updates(map[string]any{
			"code":      m.Code,
			"name":      m.Name,
			"remark":    m.Remark,
			"updatedAt": m.UpdatedAt,
		}).Error
}

// Delete 删除字典类型（软删除）。
func (r *Repo) Delete(ctx context.Context, id uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.DictType{}).
		Where("id = ? AND deletedAt IS NULL", id).
		Updates(map[string]any{
			"deletedAt": now,
			"updatedAt": now,
		}).Error
}

// ExistsByCode 检查编码是否已存在。
func (r *Repo) ExistsByCode(ctx context.Context, code string, excludeID uint64) (bool, error) {
	q := r.db.WithContext(ctx).
		Model(&model.DictType{}).
		Where("code = ? AND deletedAt IS NULL", strings.TrimSpace(code))
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}

// HasItems 检查字典类型下是否存在字典值。
func (r *Repo) HasItems(ctx context.Context, typeCode string) (bool, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.DictItem{}).
		Where("typeCode = ? AND deletedAt IS NULL", strings.TrimSpace(typeCode)).
		Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}
