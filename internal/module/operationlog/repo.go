// =============================================================================
// 文件: internal/module/operationlog/repo.go
// 模块: 操作日志
// 类型: readonly
// 职责: 封装操作日志列表查询。
// 依赖: internal/model
//       internal/pkg/pagination
// =============================================================================

package operationlog

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/pagination"
)

// Repo 操作日志数据访问层。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// FindAll 查询操作日志列表。
func (r *Repo) FindAll(ctx context.Context, req RepoFindAllReq) ([]*model.OperationLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.OperationLog{})

	if req.Account != "" {
		query = query.Where("account LIKE ?", "%"+strings.TrimSpace(req.Account)+"%")
	}
	if req.Method != "" {
		query = query.Where("method = ?", strings.TrimSpace(req.Method))
	}
	if req.Path != "" {
		query = query.Where("path LIKE ?", "%"+strings.TrimSpace(req.Path)+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	pager := pagination.New(total, req.Page, req.PageSize)
	items := make([]*model.OperationLog, 0, pager.Limit())
	if err := query.
		Order("id DESC").
		Offset(pager.Offset()).
		Limit(pager.Limit()).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}
