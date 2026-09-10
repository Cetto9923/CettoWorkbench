// =============================================================================
// 文件: internal/module/po/repokpi.go
// 模块: PO 工作台
// 类型: action
// 职责: 首页 5 个焦点摘要 KPI 真实计数（今日必推/阻塞/超期/挂起；actor role scope）。
//       MyPending 来自价值流「全部」阶段计数，由 Service 拼装。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"strings"
	"time"
)

// KPISummaryResult 4 项 KPI 聚合计数结果。
type KPISummaryResult struct {
	Today     int64 `gorm:"column:today"`
	Overdue   int64 `gorm:"column:overdue"`
	Suspended int64 `gorm:"column:suspended"`
	Blocked   int64 `gorm:"column:blocked"`
}

// CountKPISummary 单次 SQL 聚合统计今日必推/超期/挂起/阻塞，避免 4 次独立大表扫描。
func (r *Repo) CountKPISummary(ctx context.Context, account string) (KPISummaryResult, error) {
	var res KPISummaryResult
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return res, nil
	}
	today := time.Now().Format("2006-01-02")
	err := r.roleDemandBase(ctx, account).Select(`
		COUNT(CASE WHEN deadline IS NOT NULL AND deadline != '0000-00-00' AND deadline <= ? THEN 1 END) AS today,
		COUNT(CASE WHEN deadline IS NOT NULL AND deadline != '0000-00-00' AND deadline < ? THEN 1 END) AS overdue,
		COUNT(CASE WHEN hang = '1' THEN 1 END) AS suspended,
		COUNT(CASE WHEN status = 'refuse' OR (
			developFinish IS NOT NULL AND developFinish != '0000-00-00' AND developFinish <= ?
			AND (managerReviewers IS NOT NULL AND managerReviewers <> '' OR EXISTS (
				SELECT 1 FROM zt_demandmanagerreview mr WHERE mr.demand = zt_demand.id
			))
			AND COALESCE(isManagerReview, '') NOT IN ('pass', 'passed')
		) THEN 1 END) AS blocked
	`, today, today, today).Scan(&res).Error
	return res, err
}
