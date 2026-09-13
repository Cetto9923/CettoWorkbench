// =============================================================================
// 文件: internal/module/testtask/repo_execution.go
// 模块: 提测办理
// 类型: action
// 职责: 产品关联项目与执行查询；读取 CRExecution 配置；澄清涉及产品与内部用户。
// 依赖: internal/model/zentao
// =============================================================================

package testtask

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"

	zentaomodel "workbench/internal/model/zentao"
)

// executionRow 产品下执行查询行（含所属项目信息，供 stagefilter / 展示用）。
type executionRow struct {
	ID           uint   `gorm:"column:id"`
	Name         string `gorm:"column:name"`
	ProjectID    uint   `gorm:"column:project"`
	Parent       uint   `gorm:"column:parent"`
	Grade        int    `gorm:"column:grade"`
	Attribute    string `gorm:"column:attribute"`
	Status       string `gorm:"column:status"`
	Type         string `gorm:"column:type"`
	ProjectName  string `gorm:"column:project_name"`
	ProjectModel string `gorm:"column:project_model"`
}

// productRow 涉及产品查询行。
type productRow struct {
	ID   uint   `gorm:"column:id"`
	Name string `gorm:"column:name"`
}

// insideUserRow 测试负责人检索下拉查询行。
type insideUserRow struct {
	Account  string `gorm:"column:account"`
	Realname string `gorm:"column:realname"`
	Pinyin   string `gorm:"column:pinyin"`
}

// FindProductProjectIDs 通过 zt_projectproduct 解析产品关联的项目 ID。
// project 列可能是项目或执行：执行取其 project 字段。
func (r *Repo) FindProductProjectIDs(ctx context.Context, productID uint) ([]uint, error) {
	if r == nil || r.db == nil || productID == 0 {
		return []uint{}, nil
	}
	query := fmt.Sprintf(`
SELECT DISTINCT
  CASE WHEN p.type = 'project' THEN p.id ELSE p.project END AS project_id
FROM zt_projectproduct AS pp
JOIN zt_project p ON p.id = pp.project AND p.deleted = '0'
WHERE pp.product = ?
  AND (
    (p.type = 'project' AND p.id > 0)
    OR (p.type IN ('sprint', 'stage', 'kanban') AND p.project > 0)
  )`)

	type idRow struct {
		ProjectID uint `gorm:"column:project_id"`
	}
	var rows []idRow
	if err := r.db.WithContext(ctx).Raw(query, productID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]uint, 0, len(rows))
	seen := make(map[uint]struct{}, len(rows))
	for _, row := range rows {
		if row.ProjectID == 0 {
			continue
		}
		if _, ok := seen[row.ProjectID]; ok {
			continue
		}
		seen[row.ProjectID] = struct{}{}
		out = append(out, row.ProjectID)
	}
	return out, nil
}

// FindExecutionsByProjectIDs 查询若干项目下全部执行/阶段（未删）。
func (r *Repo) FindExecutionsByProjectIDs(ctx context.Context, projectIDs []uint) ([]executionRow, error) {
	if r == nil || r.db == nil || len(projectIDs) == 0 {
		return []executionRow{}, nil
	}
	const query = `
SELECT
  e.id,
  e.name,
  e.project,
  e.parent,
  e.grade,
  COALESCE(e.attribute, '') AS attribute,
  e.status,
  e.type,
  COALESCE(proj.name, '') AS project_name,
  COALESCE(proj.model, '') AS project_model
FROM zt_project e
JOIN zt_project proj ON proj.id = e.project AND proj.deleted = '0' AND proj.type = 'project'
WHERE e.deleted = '0'
  AND e.type IN ('sprint', 'stage', 'kanban')
  AND e.project IN ?
ORDER BY e.project ASC, e.id ASC`

	var rows []executionRow
	if err := r.db.WithContext(ctx).Raw(query, projectIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]executionRow, 0, len(rows))
	for _, row := range rows {
		if row.ID == 0 {
			continue
		}
		row.Name = strings.TrimSpace(row.Name)
		row.ProjectName = strings.TrimSpace(row.ProjectName)
		row.ProjectModel = strings.TrimSpace(row.ProjectModel)
		row.Attribute = strings.TrimSpace(row.Attribute)
		row.Status = strings.TrimSpace(row.Status)
		out = append(out, row)
	}
	return out, nil
}

// FindCRExecution 读取「已关闭执行的变更」开关；未配置视为 0（禁止，需 noclosed）。
func (r *Repo) FindCRExecution(ctx context.Context) (int, error) {
	if r == nil || r.db == nil {
		return 0, nil
	}
	var cfg zentaomodel.ZtConfig
	err := r.db.WithContext(ctx).
		Model(&zentaomodel.ZtConfig{}).
		Select("value").
		Where("module = ? AND `key` = ?", "common", "CRExecution").
		Order("id DESC").
		Limit(1).
		Take(&cfg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	raw := strings.TrimSpace(cfg.Value)
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, nil
	}
	return n, nil
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

// storyRow 业务需求实际转出的研发需求查询行。
type storyRow struct {
	ID          uint   `gorm:"column:id"`
	Title       string `gorm:"column:title"`
	Pri         int    `gorm:"column:pri"`
	Status      string `gorm:"column:status"`
	Stage       string `gorm:"column:stage"`
	ProductID   uint   `gorm:"column:product"`
	ProductName string `gorm:"column:product_name"`
}

// FindDemandConvertedStories 查询当前业务需求实际转出的研发需求及所属系统。
func (r *Repo) FindDemandConvertedStories(ctx context.Context, demandID uint) ([]storyRow, error) {
	if r == nil || r.db == nil || demandID == 0 {
		return []storyRow{}, nil
	}
	const query = `
SELECT
  s.id, s.title, s.pri, s.status, s.stage, s.product,
  COALESCE(p.name, '') AS product_name
FROM zt_story s
JOIN zt_product p ON p.id = s.product AND p.deleted = '0'
WHERE s.fromDemand = ? AND s.deleted = '0'
ORDER BY s.id ASC`
	var rows []storyRow
	if err := r.db.WithContext(ctx).Raw(query, demandID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].Title = strings.TrimSpace(rows[i].Title)
		rows[i].Status = strings.TrimSpace(rows[i].Status)
		rows[i].Stage = strings.TrimSpace(rows[i].Stage)
		rows[i].ProductName = strings.TrimSpace(rows[i].ProductName)
	}
	return rows, nil
}

// ListInsideUsers 查询内部用户列表（测试负责人检索下拉，对齐排期）。
func (r *Repo) ListInsideUsers(ctx context.Context) ([]UserOption, error) {
	if r == nil || r.db == nil {
		return []UserOption{}, nil
	}
	const query = `
SELECT account, realname, pinyin
FROM zt_user
WHERE deleted = '0'
  AND type = 'inside'
ORDER BY account ASC`
	var rows []insideUserRow
	if err := r.db.WithContext(ctx).Raw(query).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]UserOption, 0, len(rows))
	for _, row := range rows {
		account := strings.TrimSpace(row.Account)
		if account == "" {
			continue
		}
		realname := strings.TrimSpace(row.Realname)
		if realname == "" {
			realname = account
		}
		out = append(out, UserOption{Account: account, Realname: realname, Pinyin: strings.TrimSpace(row.Pinyin)})
	}
	return out, nil
}
