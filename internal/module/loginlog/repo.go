// =============================================================================
// 文件: internal/module/loginlog/repo.go
// 模块: 登录日志
// 类型: readonly
// 职责: 封装登录日志列表查询。
// 依赖: internal/model
// =============================================================================

package loginlog

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"workbench/internal/model"
)

// Repo 登录日志数据访问层。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// RepoFindAllReq Repo 列表查询入参。
type RepoFindAllReq struct {
	Account  string
	IP       string
	StartAt  *time.Time
	EndAt    *time.Time
	Page     int
	PageSize int
}

// FindAll 查询登录日志列表与总数。
func (r *Repo) FindAll(ctx context.Context, req RepoFindAllReq) ([]model.LoginLog, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.LoginLog{})
	if req.Account != "" {
		like := "%" + strings.TrimSpace(req.Account) + "%"
		q = q.Where("account LIKE ?", like)
	}
	if req.IP != "" {
		like := "%" + strings.TrimSpace(req.IP) + "%"
		q = q.Where("ip LIKE ?", like)
	}
	if req.StartAt != nil {
		q = q.Where("createdDate >= ?", *req.StartAt)
	}
	if req.EndAt != nil {
		q = q.Where("createdDate <= ?", *req.EndAt)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := 0
	if req.Page > 1 {
		offset = (req.Page - 1) * req.PageSize
	}
	var items []model.LoginLog
	if err := q.Order("id DESC").Offset(offset).Limit(req.PageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
