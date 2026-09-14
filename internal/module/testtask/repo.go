// =============================================================================
// 文件: internal/module/testtask/repo.go
// 模块: 提测办理
// 类型: action
// 职责: 从 zt_demand 读取提测上下文；涉及产品来自 zt_demandclarify。
// 依赖: 无
// =============================================================================

package testtask

import (
	"context"
	"errors"
	"strings"

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
			CAST(NULLIF(d.mainSystem, '') AS UNSIGNED) AS main_system_id,
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

// productRow 涉及产品查询行。
type productRow struct {
	ID   uint   `gorm:"column:id"`
	Name string `gorm:"column:name"`
}

// FindDemandInvolvedProducts 查询业需涉及产品（zt_demandclarify），按 id 升序。
func (r *Repo) FindDemandInvolvedProducts(ctx context.Context, demandID uint) ([]productRow, error) {
	if r == nil || r.db == nil || demandID == 0 {
		return []productRow{}, nil
	}
	const query = `
SELECT DISTINCT p.id, p.name
FROM zt_demandclarify dc
JOIN zt_product p ON p.id = dc.product AND p.deleted = '0'
WHERE dc.demand = ?
ORDER BY p.id ASC`
	var rows []productRow
	if err := r.db.WithContext(ctx).Raw(query, demandID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]productRow, 0, len(rows))
	for _, row := range rows {
		if row.ID == 0 {
			continue
		}
		out = append(out, productRow{
			ID:   row.ID,
			Name: strings.TrimSpace(row.Name),
		})
	}
	return out, nil
}

// buildMeta 版本所属项目/执行（用于创建测试单）。
type buildMeta struct {
	ID        uint
	ProductID uint
	ProjectID uint
	Execution uint
	Name      string
}

// FindBuildsByIDs 按版本 ID 批量读取所属项目与执行。
func (r *Repo) FindBuildsByIDs(ctx context.Context, ids []uint) (map[uint]buildMeta, error) {
	out := make(map[uint]buildMeta)
	if r == nil || r.db == nil || len(ids) == 0 {
		return out, nil
	}
	uniq := make([]uint, 0, len(ids))
	seen := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return out, nil
	}

	type row struct {
		ID        uint   `gorm:"column:id"`
		ProductID uint   `gorm:"column:product"`
		ProjectID uint   `gorm:"column:project"`
		Execution uint   `gorm:"column:execution"`
		Name      string `gorm:"column:name"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Table("zt_build").
		Select("id, product, project, execution, name").
		Where("id IN ? AND deleted = '0'", uniq).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = buildMeta{
			ID:        row.ID,
			ProductID: row.ProductID,
			ProjectID: row.ProjectID,
			Execution: row.Execution,
			Name:      strings.TrimSpace(row.Name),
		}
	}
	return out, nil
}
