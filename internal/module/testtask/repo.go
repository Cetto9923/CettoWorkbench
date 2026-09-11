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
// 多返回 main_system_id 用于把主系统放在 systems 列表首位。
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

// buildMeta 版本所属产品/项目/执行（用于创建测试单前的归属校验）。
type buildMeta struct {
	ID        uint
	ProductID uint
	ProjectID uint
	Execution uint
	Name      string
}

// FindBuildsByIDs 按版本 ID 批量读取未删除版本的产品/项目/执行。
// 重复 ID 自动去重，空输入返回空 map；查询失败原样返回。
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
		Where("id IN ? AND deleted = ?", uniq, "0").
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

// VerifyExecutionBelongsToProject 校验执行是否属于所选项目（且项目类型为 project）。
// 项目=0 视作跳过（禅道历史 build 偶有无 project 字段）；执行=0 或执行.project != projectID 返回 false。
func (r *Repo) VerifyExecutionBelongsToProject(ctx context.Context, projectID, executionID uint) (bool, error) {
	if r == nil || r.db == nil {
		return false, nil
	}
	if executionID == 0 {
		return false, nil
	}
	if projectID == 0 {
		// 项目未知时只校验执行本身存在
		var count int64
		if err := r.db.WithContext(ctx).Table("zt_project").
			Where("id = ? AND deleted = ? AND type IN ('sprint','stage','kanban')", executionID, "0").
			Count(&count).Error; err != nil {
			return false, err
		}
		return count == 1, nil
	}
	var count int64
	if err := r.db.WithContext(ctx).Table("zt_project").
		Where("id = ? AND project = ? AND deleted = ? AND type IN ('sprint','stage','kanban')", executionID, projectID, "0").
		Count(&count).Error; err != nil {
		return false, err
	}
	return count == 1, nil
}
