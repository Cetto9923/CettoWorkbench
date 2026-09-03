// =============================================================================
// 文件: internal/module/schedule/repo_window.go
// 模块: 排期工作台
// 类型: action
// 职责: 窗口附属:统计查询(工时/阶段/需求/产品) + 写入辅助(CreateProductPlan/CreateWindowProduct/GetMatchingPlans)。
// 依赖: internal/model
// =============================================================================

package schedule

import (
	"context"
	"time"

	"workbench/internal/model"
)

// WindowStageStats 窗口需求阶段统计。
type WindowStageStats struct {
	DemandCount  int // 业需数 + 非需求池软需数
	DevCount     int // 开发中
	TestCount    int // 测试中
	DeliverCount int // 待交付
}

// GetWindowConsumedHours 查询窗口关联任务的已消耗工时总和。
// 链路: zt_versionwindowproduct.plan → zt_planstory.story → zt_task.consumed
func (r *Repo) GetWindowConsumedHours(ctx context.Context, windowID uint64) (float64, error) {
	const query = `
SELECT COALESCE(SUM(t.consumed), 0) AS total
FROM zt_versionwindowproduct vwp
JOIN zt_planstory ps ON ps.plan = vwp.plan
JOIN zt_task t ON t.story = ps.story AND t.deleted = '0' AND t.status != 'closed'
WHERE vwp.versionWindow = ? AND vwp.deletedAt IS NULL AND vwp.plan IS NOT NULL`

	var row struct {
		Total float64 `gorm:"column:total"`
	}
	if err := r.db.WithContext(ctx).Raw(query, windowID).Scan(&row).Error; err != nil {
		return 0, err
	}
	return row.Total, nil
}

// GetWindowStageStats 查询窗口关联 story 的需求/开发/测试/待交付统计。
// 链路: zt_versionwindowproduct.plan → zt_planstory.story → zt_story
func (r *Repo) GetWindowStageStats(ctx context.Context, windowID uint64) (*WindowStageStats, error) {
	const query = `
SELECT
  COUNT(DISTINCT CASE WHEN s.sourceType = 'demandpool' AND s.fromDemand > 0 THEN s.fromDemand ELSE NULL END)
  + COUNT(CASE WHEN IFNULL(s.sourceType, '') != 'demandpool' THEN 1 ELSE NULL END) AS demandCount,
  SUM(CASE WHEN s.stage = 'developing' THEN 1 ELSE 0 END) AS devCount,
  SUM(CASE WHEN s.stage = 'testing' THEN 1 ELSE 0 END) AS testCount,
  SUM(CASE WHEN s.stage IN ('verified','tested','delivering','delivered') THEN 1 ELSE 0 END) AS deliverCount
FROM zt_versionwindowproduct vwp
JOIN zt_planstory ps ON ps.plan = vwp.plan
JOIN zt_story s ON s.id = ps.story AND s.deleted = '0'
WHERE vwp.versionWindow = ? AND vwp.deletedAt IS NULL AND vwp.plan IS NOT NULL`

	var row struct {
		DemandCount  int64 `gorm:"column:demandCount"`
		DevCount     int64 `gorm:"column:devCount"`
		TestCount    int64 `gorm:"column:testCount"`
		DeliverCount int64 `gorm:"column:deliverCount"`
	}
	if err := r.db.WithContext(ctx).Raw(query, windowID).Scan(&row).Error; err != nil {
		return nil, err
	}
	return &WindowStageStats{
		DemandCount:  int(row.DemandCount),
		DevCount:     int(row.DevCount),
		TestCount:    int(row.TestCount),
		DeliverCount: int(row.DeliverCount),
	}, nil
}

// GetWindowDemandCount 查询窗口关联的需求数量（业需去重 + 独立软需）。
func (r *Repo) GetWindowDemandCount(ctx context.Context, windowID uint64) (int, error) {
	const query = `
SELECT
  COUNT(DISTINCT CASE WHEN s.sourceType = 'demandpool' AND s.fromDemand > 0 THEN s.fromDemand ELSE NULL END)
  + COUNT(CASE WHEN IFNULL(s.sourceType, '') != 'demandpool' THEN 1 ELSE NULL END) AS demandCount
FROM zt_versionwindowproduct vwp
JOIN zt_planstory ps ON ps.plan = vwp.plan
JOIN zt_story s ON s.id = ps.story AND s.deleted = '0'
WHERE vwp.versionWindow = ? AND vwp.deletedAt IS NULL AND vwp.plan IS NOT NULL`

	var row struct {
		DemandCount int64 `gorm:"column:demandCount"`
	}
	if err := r.db.WithContext(ctx).Raw(query, windowID).Scan(&row).Error; err != nil {
		return 0, err
	}
	return int(row.DemandCount), nil
}

// GetWindowProducts 查询窗口关联产品及计划信息。
func (r *Repo) GetWindowProducts(ctx context.Context, windowID uint64) ([]WindowProductRow, error) {
	const query = `
SELECT
  vwp.product,
  p.name AS product_name,
  vwp.plan,
  vwp.planSynced,
  pp.title AS plan_title,
  DATE_FORMAT(pp.` + "`begin`" + `, '%Y-%m-%d') AS plan_begin,
  DATE_FORMAT(pp.` + "`end`" + `, '%Y-%m-%d') AS plan_end
FROM zt_versionwindowproduct vwp
INNER JOIN zt_product p ON p.id = vwp.product
LEFT JOIN zt_productplan pp ON pp.id = vwp.plan AND pp.deleted = '0'
WHERE vwp.versionWindow = ? AND vwp.deletedAt IS NULL
ORDER BY vwp.id ASC`

	var rows []WindowProductRow
	if err := r.db.WithContext(ctx).Raw(query, windowID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CreateProductPlan 在禅道创建产品计划，返回新计划 ID。
func (r *Repo) CreateProductPlan(ctx context.Context, productID uint, title, begin, end, account string) (uint, error) {
	row := ztProductplanCreate{
		Product:      productID,
		Branch:       "0",
		Parent:       0,
		Title:        title,
		Status:       "wait",
		Begin:        begin,
		End:          end,
		Order:        "0",
		ClosedReason: "",
		CreatedBy:    account,
		CreatedDate:  time.Now(),
		Deleted:      "0",
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

// CreateWindowProduct 写入 zt_versionwindowproduct 关联记录。
func (r *Repo) CreateWindowProduct(ctx context.Context, wp *model.VersionWindowProduct) error {
	return r.db.WithContext(ctx).Create(wp).Error
}

// GetMatchingPlans 根据产品 ID 和结束日期查询匹配的计划。
func (r *Repo) GetMatchingPlans(ctx context.Context, productID uint, endDate string) ([]ZtProductplan, error) {
	const query = `
SELECT id, product, title, begin, end, status
FROM zt_productplan
WHERE product = ? AND end = ? AND deleted = '0'
ORDER BY id DESC`

	var rows []ZtProductplan
	if err := r.db.WithContext(ctx).Raw(query, productID, endDate).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
