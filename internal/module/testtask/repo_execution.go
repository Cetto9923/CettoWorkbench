// =============================================================================
// 文件: internal/module/testtask/repo_execution.go
// 模块: 提测办理
// 类型: action
// 职责: 对齐禅道 product::getExecutionPairsByProduct 的执行查询。
// 依赖: internal/model/zentao
// =============================================================================

package testtask

import (
	"context"
	"strings"
	"time"

	zentaomodel "workbench/internal/model/zentao"
)

// executionRow 产品关联执行行（对齐 getExecutionPairsByProduct 选出字段）。
type executionRow struct {
	ID          uint       `gorm:"column:id"`
	Name        string     `gorm:"column:name"`
	ProjectID   uint       `gorm:"column:project"`
	Grade       int        `gorm:"column:grade"`
	Path        string     `gorm:"column:path"`
	Parent      uint       `gorm:"column:parent"`
	Attribute   string     `gorm:"column:attribute"`
	Multiple    string     `gorm:"column:multiple"`
	Type        string     `gorm:"column:type"`
	ProjectName string     `gorm:"column:projectName"`
	Begin       *time.Time `gorm:"column:begin"`
}

// stageRow 项目下阶段（供 buildExecutionPairs 拼 path 名、判父阶段）。
type stageRow struct {
	ID        uint   `gorm:"column:id"`
	Name      string `gorm:"column:name"`
	Attribute string `gorm:"column:attribute"`
	Parent    uint   `gorm:"column:parent"`
	Path      string `gorm:"column:path"`
	ProjectID uint   `gorm:"column:project"`
}

// FindExecutionsByProduct 对齐禅道 product::getExecutionPairsByProduct（projectID=0）。
// SQL：zt_projectproduct ⋈ 执行 ⋈ 项目；type∈sprint,kanban,stage；deleted=0；
// 排序 begin DESC, id DESC（无指定项目时不作瀑布升降序切换）。
func (r *Repo) FindExecutionsByProduct(ctx context.Context, productID uint) ([]executionRow, error) {
	if r == nil || r.db == nil || productID == 0 {
		return []executionRow{}, nil
	}
	query := `
SELECT
  t2.id,
  t2.name,
  t2.project,
  t2.grade,
  COALESCE(t2.path, '') AS path,
  t2.parent,
  COALESCE(t2.attribute, '') AS attribute,
  COALESCE(t2.multiple, '') AS multiple,
  t2.type,
  COALESCE(t3.name, '') AS projectName,
  t2.begin
FROM ` + zentaomodel.ZtProjectproduct{}.TableName() + ` AS t1
LEFT JOIN zt_project AS t2 ON t1.project = t2.id
LEFT JOIN zt_project AS t3 ON t2.project = t3.id
WHERE t1.product = ?
  AND t2.type IN ('sprint', 'kanban', 'stage')
  AND t2.deleted = '0'
ORDER BY t2.begin DESC, t2.id DESC`

	var rows []executionRow
	if err := r.db.WithContext(ctx).Raw(query, productID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]executionRow, 0, len(rows))
	for _, row := range rows {
		if row.ID == 0 {
			continue
		}
		row.Name = strings.TrimSpace(row.Name)
		row.ProjectName = strings.TrimSpace(row.ProjectName)
		row.Attribute = strings.TrimSpace(row.Attribute)
		row.Multiple = strings.TrimSpace(row.Multiple)
		row.Path = strings.TrimSpace(row.Path)
		row.Type = strings.TrimSpace(row.Type)
		out = append(out, row)
	}
	return out, nil
}

// FindStagesByProjectIDs 查询项目下未删阶段（对齐 buildExecutionPairs 内 stages 查询）。
func (r *Repo) FindStagesByProjectIDs(ctx context.Context, projectIDs []uint) ([]stageRow, error) {
	if r == nil || r.db == nil || len(projectIDs) == 0 {
		return []stageRow{}, nil
	}
	const query = `
SELECT
  id,
  name,
  COALESCE(attribute, '') AS attribute,
  parent,
  COALESCE(path, '') AS path,
  project
FROM zt_project
WHERE type = 'stage'
  AND deleted = '0'
  AND project IN ?`

	var rows []stageRow
	if err := r.db.WithContext(ctx).Raw(query, projectIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]stageRow, 0, len(rows))
	for _, row := range rows {
		if row.ID == 0 {
			continue
		}
		row.Name = strings.TrimSpace(row.Name)
		row.Attribute = strings.TrimSpace(row.Attribute)
		row.Path = strings.TrimSpace(row.Path)
		out = append(out, row)
	}
	return out, nil
}
