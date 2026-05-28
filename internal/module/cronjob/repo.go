// =============================================================================
// 文件: internal/module/cronjob/repo.go
// 模块: 定时任务
// 类型: action
// 职责: 封装定时任务与执行日志数据访问。
// 依赖: internal/model
//       internal/pkg/pagination
// =============================================================================

package cronjob

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/pagination"
)

// Repo 定时任务仓储。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// FindAll 查询任务列表。
func (r *Repo) FindAll(ctx context.Context, req RepoFindAllReq) ([]model.CronJob, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.CronJob{})
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	pager := pagination.New(total, req.Page, req.PageSize)
	rows := make([]model.CronJob, 0, pager.Limit())
	if err := query.
		Order("id ASC").
		Offset(pager.Offset()).
		Limit(pager.Limit()).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// FindByID 按 ID 查询任务。
func (r *Repo) FindByID(ctx context.Context, id uint64) (*model.CronJob, error) {
	var row model.CronJob
	if err := r.db.WithContext(ctx).
		Model(&model.CronJob{}).
		Where("id = ?", id).
		First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// UpdateEnabled 更新任务启用状态。
func (r *Repo) UpdateEnabled(ctx context.Context, id uint64, isEnabled bool) error {
	return r.db.WithContext(ctx).
		Model(&model.CronJob{}).
		Where("id = ?", id).
		Update("isEnabled", isEnabled).Error
}

// FindLogs 查询执行日志列表。
func (r *Repo) FindLogs(ctx context.Context, req RepoFindLogsReq) ([]model.CronLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.CronLog{})
	if strings.TrimSpace(req.JobName) != "" {
		query = query.Where("jobName = ?", strings.TrimSpace(req.JobName))
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	pager := pagination.New(total, req.Page, req.PageSize)
	rows := make([]model.CronLog, 0, pager.Limit())
	if err := query.
		Order("id DESC").
		Offset(pager.Offset()).
		Limit(pager.Limit()).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
