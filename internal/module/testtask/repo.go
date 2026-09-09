// =============================================================================
// 文件: internal/module/testtask/repo.go
// 模块: 提测办理
// 类型: action
// 职责: 从 zt_demand 读取提测上下文（联表产品名；人员展示名由 user 模块解析）。
// 依赖: 无
// =============================================================================

package testtask

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var errDemandNotFound = errors.New("demand not found")

// Repo 提测数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo（只读备库优先，可与主库相同）。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// FindDemandContext 按业需 ID 读取上下文行（账号字段；展示名由 Service 用用户 map 解析）。
func (r *Repo) FindDemandContext(ctx context.Context, demandID uint) (*DemandContextRow, error) {
	if r == nil || r.db == nil || demandID == 0 {
		return nil, errDemandNotFound
	}
	var row DemandContextRow
	err := r.db.WithContext(ctx).Table("zt_demand AS d").
		Select(`d.id, d.name, d.status, d.stage, d.BRA, d.RD, d.QD,
			COALESCE(p.name, d.mainSystem, '') AS product_name,
			COALESCE(DATE_FORMAT(d.estimateLaunch, '%Y-%m-%d'), '') AS estimate_launch`).
		Joins("LEFT JOIN zt_product p ON p.id = CAST(NULLIF(d.mainSystem, '') AS UNSIGNED) AND p.deleted = '0'").
		Where("d.id = ? AND d.deleted = '0'", demandID).
		Limit(1).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, errDemandNotFound
	}
	return &row, nil
}
