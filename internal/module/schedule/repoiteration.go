// =============================================================================
// 文件: internal/module/schedule/repoiteration.go
// 模块: 排期工作台
// 类型: repo
// 职责: 读取可关联的原生产品计划。
// 依赖: 无
// =============================================================================
package schedule

import "context"

func (r *Repo) IterationPlans(ctx context.Context, productID uint) ([]MatchingPlanItem, error) {
	var plans []MatchingPlanItem
	err := r.db.WithContext(ctx).Raw(`SELECT id, title, DATE_FORMAT(begin, '%Y-%m-%d') AS begin, DATE_FORMAT(end, '%Y-%m-%d') AS end FROM zt_productplan
 WHERE product = ? AND deleted = '0' AND status <> 'closed' ORDER BY end DESC, id DESC`, productID).Scan(&plans).Error
	return plans, err
}
