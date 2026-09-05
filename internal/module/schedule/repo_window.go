// =============================================================================
// 文件: internal/module/schedule/repo_window.go
// 模块: 排期工作台
// 类型: repo
// 职责: 版本窗口批量阶段统计聚合查询 (P2: 消除循环 N+1)。
// 依赖: gorm.io/gorm
// =============================================================================

package schedule

import (
	"context"
)

// GetWindowStageStatsBatch 批量查询窗口关联 story 的阶段统计（单次 SQL 聚合，消除 N+1）。
func (r *Repo) GetWindowStageStatsBatch(ctx context.Context, windowIDs []uint64) (map[uint64]WindowStageStats, error) {
	out := make(map[uint64]WindowStageStats, len(windowIDs))
	if len(windowIDs) == 0 {
		return out, nil
	}

	const query = `
SELECT
  vwp.versionWindow AS windowID,
  COUNT(DISTINCT CASE WHEN s.sourceType = 'demandpool' AND s.fromDemand > 0 THEN s.fromDemand ELSE NULL END)
  + COUNT(CASE WHEN IFNULL(s.sourceType, '') != 'demandpool' THEN 1 ELSE NULL END) AS demandCount,
  SUM(CASE WHEN s.stage = 'developing' THEN 1 ELSE 0 END) AS devCount,
  SUM(CASE WHEN s.stage = 'testing' THEN 1 ELSE 0 END) AS testCount,
  SUM(CASE WHEN s.stage IN ('verified','tested','delivering','delivered') THEN 1 ELSE 0 END) AS deliverCount
FROM zt_versionwindowproduct vwp
JOIN zt_planstory ps ON ps.plan = vwp.plan
JOIN zt_story s ON s.id = ps.story AND s.deleted = '0'
WHERE vwp.versionWindow IN ? AND vwp.deletedAt IS NULL AND vwp.plan IS NOT NULL
GROUP BY vwp.versionWindow`

	type row struct {
		WindowID     uint64 `gorm:"column:windowID"`
		DemandCount  int64  `gorm:"column:demandCount"`
		DevCount     int64  `gorm:"column:devCount"`
		TestCount    int64  `gorm:"column:testCount"`
		DeliverCount int64  `gorm:"column:deliverCount"`
	}

	var rows []row
	if err := r.db.WithContext(ctx).Raw(query, windowIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, r := range rows {
		out[r.WindowID] = WindowStageStats{
			DemandCount:  int(r.DemandCount),
			DevCount:     int(r.DevCount),
			TestCount:    int(r.TestCount),
			DeliverCount: int(r.DeliverCount),
		}
	}
	return out, nil
}
