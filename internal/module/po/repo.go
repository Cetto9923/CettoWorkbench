// =============================================================================
// 文件: internal/module/po/repo.go
// 模块: PO 工作台
// 类型: action
// 职责: 受理阶段业需的 MySQL 统计与列表查询。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

// acceptDemandStatuses 受理阶段对应的 zt_demand.status。
var acceptDemandStatuses = []string{"draft", "wait", "refuse"}

// Repo PO 工作台数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// AcceptDemandRow 受理阶段业需列表投影。
type AcceptDemandRow struct {
	ID   int    `gorm:"column:id"`
	Name string `gorm:"column:name"`
	Pri  string `gorm:"column:pri"`
}

// CountAcceptDemands 统计当前用户作为 QD/RD/BRA 且处于受理状态的业需数量。
func (r *Repo) CountAcceptDemands(ctx context.Context, account string) (int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return 0, nil
	}
	var total int64
	err := r.db.WithContext(ctx).Table("zt_demand").
		Where("deleted = ?", "0").
		Where("(QD = ? OR RD = ? OR BRA = ?)", account, account, account).
		Where("status IN ?", acceptDemandStatuses).
		Count(&total).Error
	return total, err
}

// FindAcceptDemands 查询当前用户作为 QD/RD/BRA 且处于受理状态的业需列表。
func (r *Repo) FindAcceptDemands(ctx context.Context, account string) ([]AcceptDemandRow, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, nil
	}
	var rows []AcceptDemandRow
	err := r.db.WithContext(ctx).Table("zt_demand").
		Select("id", "name", "pri").
		Where("deleted = ?", "0").
		Where("(QD = ? OR RD = ? OR BRA = ?)", account, account, account).
		Where("status IN ?", acceptDemandStatuses).
		Order("id DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
