package schedule

import (
	"context"
	"strings"
)

// GetDemandClarifyStoryDefaults 查询每个涉及系统的澄清需求分析人员。
// 同一系统存在多条澄清记录时按澄清记录 ID 保留最早一条，若其 PM 为空则用后续非空 PM 补齐。
func (r *Repo) GetDemandClarifyStoryDefaults(ctx context.Context, demandID uint) ([]DemandSchedulingClarifyDefault, error) {
	if demandID == 0 {
		return []DemandSchedulingClarifyDefault{}, nil
	}

	const query = `
SELECT
  CAST(dc.product AS UNSIGNED) AS productID,
  p.name AS productName,
  TRIM(dc.PM) AS analyst
FROM zt_demandclarify dc
JOIN zt_product p ON p.id = dc.product AND p.deleted = '0'
WHERE dc.demand = ?
  AND TRIM(dc.product) <> ''
ORDER BY dc.id ASC`

	var rows []DemandSchedulingClarifyDefault
	if err := r.db.WithContext(ctx).Raw(query, demandID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	indexByProduct := make(map[uint]int, len(rows))
	out := make([]DemandSchedulingClarifyDefault, 0, len(rows))
	for _, row := range rows {
		if row.ProductID == 0 {
			continue
		}
		row.ProductName = strings.TrimSpace(row.ProductName)
		row.Analyst = strings.TrimSpace(row.Analyst)
		if index, exists := indexByProduct[row.ProductID]; exists {
			if out[index].Analyst == "" && row.Analyst != "" {
				out[index].Analyst = row.Analyst
			}
			continue
		}
		indexByProduct[row.ProductID] = len(out)
		out = append(out, row)
	}
	return out, nil
}

// ListSchedulingWindowProductPlans 查询窗口-系统关联的禅道产品计划，用于研发需求默认值回显。
func (r *Repo) ListSchedulingWindowProductPlans(ctx context.Context, windowIDs, productIDs []uint) ([]SchedulingWindowProductPlan, error) {
	windowIDs = uniqueUints(windowIDs)
	productIDs = uniqueUints(productIDs)
	if len(windowIDs) == 0 || len(productIDs) == 0 {
		return []SchedulingWindowProductPlan{}, nil
	}

	const query = `
SELECT
  vwp.versionWindow AS windowID,
  vwp.product AS productID,
  COALESCE(vwp.plan, 0) AS planID,
  COALESCE(pp.title, '') AS planName
FROM zt_versionwindowproduct vwp
LEFT JOIN zt_productplan pp ON pp.id = vwp.plan AND pp.deleted = '0'
WHERE vwp.versionWindow IN ?
  AND vwp.product IN ?
  AND vwp.deletedAt IS NULL
ORDER BY vwp.versionWindow ASC, vwp.product ASC`

	var rows []SchedulingWindowProductPlan
	if err := r.db.WithContext(ctx).Raw(query, windowIDs, productIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].PlanName = strings.TrimSpace(rows[i].PlanName)
	}
	return rows, nil
}

// ListSchedulingProductPlans 查询同产品下可选的禅道产品计划，用于转研发需求时允许改计划。
func (r *Repo) ListSchedulingProductPlans(ctx context.Context, productIDs []uint) (map[string][]SchedulingProductPlanOption, error) {
	productIDs = uniqueUints(productIDs)
	out := make(map[string][]SchedulingProductPlanOption, len(productIDs))
	for _, productID := range productIDs {
		out[formatUintKey(productID)] = []SchedulingProductPlanOption{}
	}
	if len(productIDs) == 0 {
		return out, nil
	}

	const query = `
SELECT id, product AS productID, title
FROM zt_productplan
WHERE product IN ?
  AND deleted = '0'
ORDER BY product ASC, id DESC`

	var rows []SchedulingProductPlanOption
	if err := r.db.WithContext(ctx).Raw(query, productIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		row.Title = strings.TrimSpace(row.Title)
		key := formatUintKey(row.ProductID)
		out[key] = append(out[key], row)
	}
	return out, nil
}
