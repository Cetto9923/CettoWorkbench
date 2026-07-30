// =============================================================================
// 文件: internal/module/po/repo.go
// 模块: PO 工作台
// 类型: action
// 职责: 价值流阶段业需的 MySQL 统计与列表查询（按 QD/RD/BRA + status）。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

// mysqlStageDemandStatuses 走 MySQL 的价值流阶段 → zt_demand.status 列表。
var mysqlStageDemandStatuses = map[string][]string{
	"accept":  {"draft", "wait", "refuse"},
	"testing": {"testing"},
}

// Repo PO 工作台数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// DemandRow 业需列表投影。
type DemandRow struct {
	ID   int    `gorm:"column:id"`
	Name string `gorm:"column:name"`
	Pri  string `gorm:"column:pri"`
}

// CountRoleDemands 统计当前用户作为 QD/RD/BRA 且 status 在指定集合内的业需数量。
func (r *Repo) CountRoleDemands(ctx context.Context, account string, statuses []string) (int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" || len(statuses) == 0 {
		return 0, nil
	}
	var total int64
	err := r.db.WithContext(ctx).Table("zt_demand").
		Where("deleted = ?", "0").
		Where("(QD = ? OR RD = ? OR BRA = ?)", account, account, account).
		Where("status IN ?", statuses).
		Count(&total).Error
	return total, err
}

// FindRoleDemands 查询当前用户作为 QD/RD/BRA 且 status 在指定集合内的业需列表。
func (r *Repo) FindRoleDemands(ctx context.Context, account string, statuses []string) ([]DemandRow, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" || len(statuses) == 0 {
		return nil, nil
	}
	var rows []DemandRow
	err := r.db.WithContext(ctx).Table("zt_demand").
		Select("id", "name", "pri").
		Where("deleted = ?", "0").
		Where("(QD = ? OR RD = ? OR BRA = ?)", account, account, account).
		Where("status IN ?", statuses).
		Order("id DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
