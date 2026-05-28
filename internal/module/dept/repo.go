// =============================================================================
// 文件: internal/module/dept/repo.go
// 模块: 部门管理
// 类型: crud
// 职责: 封装部门数据访问。
// 依赖: internal/model
// =============================================================================

package dept

import (
	"context"
	"time"

	"gorm.io/gorm"

	"workbench/internal/model"
)

// Repo 封装部门数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// RepoFindAllReq 列表查询参数。
type RepoFindAllReq struct {
	Page     int
	PageSize int
}

// FindAll 查询全部未删除部门（count + find）。
func (r *Repo) FindAll(ctx context.Context, req RepoFindAllReq) ([]model.Dept, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&model.Dept{}).
		Where("deletedAt IS NULL")

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []model.Dept
	if err := query.
		Order("sort ASC, id ASC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// FindByID 按 ID 查询部门。
func (r *Repo) FindByID(ctx context.Context, id uint64) (*model.Dept, error) {
	var row model.Dept
	if err := r.db.WithContext(ctx).
		Model(&model.Dept{}).
		Where("id = ? AND deletedAt IS NULL", id).
		First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// Create 创建部门。
func (r *Repo) Create(ctx context.Context, m *model.Dept) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// Update 更新部门。
func (r *Repo) Update(ctx context.Context, m *model.Dept) error {
	return r.db.WithContext(ctx).
		Model(&model.Dept{}).
		Where("id = ? AND deletedAt IS NULL", m.ID).
		Updates(map[string]any{
			"parentId":  m.ParentID,
			"name":      m.Name,
			"leader":    m.Leader,
			"phone":     m.Phone,
			"email":     m.Email,
			"status":    m.Status,
			"sort":      m.Sort,
			"updatedAt": m.UpdatedAt,
		}).Error
}

// Delete 软删除部门。
func (r *Repo) Delete(ctx context.Context, id uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.Dept{}).
		Where("id = ? AND deletedAt IS NULL", id).
		Updates(map[string]any{
			"deletedAt": now,
			"updatedAt": now,
		}).Error
}

// CountChildren 统计直接子部门数量。
func (r *Repo) CountChildren(ctx context.Context, id uint64) (int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.Dept{}).
		Where("parentId = ? AND deletedAt IS NULL", id).
		Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// CountUsers 统计部门下关联用户数量。
func (r *Repo) CountUsers(ctx context.Context, id uint64) (int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where(&model.User{DeptID: id}).
		Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// UpdateStatus 仅更新部门状态。
func (r *Repo) UpdateStatus(ctx context.Context, id uint64, status uint8) error {
	return r.db.WithContext(ctx).
		Model(&model.Dept{}).
		Where("id = ? AND deletedAt IS NULL", id).
		Updates(map[string]any{
			"status": status,
		}).Error
}

// UpdateStatusBatch 批量更新部门状态。
func (r *Repo) UpdateStatusBatch(ctx context.Context, ids []uint64, status uint8) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&model.Dept{}).
		Where("id IN ? AND deletedAt IS NULL", ids).
		Updates(map[string]any{
			"status": status,
		}).Error
}
