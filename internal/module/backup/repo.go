// =============================================================================
// 文件: internal/module/backup/repo.go
// 模块: 数据库备份
// 类型: action
// 职责: 封装备份记录查询数据访问。
// 依赖: internal/model
//       internal/pkg/pagination
// =============================================================================

package backup

import (
	"context"

	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/pagination"
)

// Repo 备份记录仓储。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// FindAll 查询备份记录列表。
func (r *Repo) FindAll(ctx context.Context, req RepoFindAllReq) ([]model.BackupRecord, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.BackupRecord{}).Where("deleted = 0")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	pager := pagination.New(total, req.Page, req.PageSize)
	rows := make([]model.BackupRecord, 0, pager.Limit())
	if err := query.
		Order("id DESC").
		Offset(pager.Offset()).
		Limit(pager.Limit()).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// FindByID 按 ID 查询备份记录。
func (r *Repo) FindByID(ctx context.Context, id uint64) (*model.BackupRecord, error) {
	var row model.BackupRecord
	if err := r.db.WithContext(ctx).
		Model(&model.BackupRecord{}).
		Where("id = ? AND deleted = 0", id).
		First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}
