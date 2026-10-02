// =============================================================================
// 文件: internal/module/schedule/repo_window_stats.go
// 模块: 排期工作台
// 类型: repo
// 职责: 批量读取窗口关联与工时统计。
// 依赖: internal/model
// =============================================================================

package schedule

import "context"

type windowStats struct {
	WindowStageStats
	Consumed float64
}

func (r *Repo) windowsStats(ctx context.Context, ids []uint64) (map[uint64]windowStats, error) {
	out := make(map[uint64]windowStats, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var items []struct {
		WindowWorkItem
		WindowID uint64 `gorm:"column:window_id"`
	}
	if err := r.db.WithContext(ctx).Raw(findWindowWorkItemsSQL, ids, ids, ids).Scan(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		value := out[item.WindowID]
		value.DemandCount++
		out[item.WindowID] = value
	}
	var consumed []struct {
		WindowID uint64 `gorm:"column:windowID"`
		Total    float64
	}
	const consumedSQL = `SELECT linked.windowID, COALESCE(SUM(t.consumed),0) AS total
 FROM (SELECT DISTINCT vwp.versionWindow AS windowID, ps.story FROM zt_versionwindowproduct vwp
 JOIN zt_planstory ps ON ps.plan=vwp.plan WHERE vwp.versionWindow IN ? AND vwp.deletedAt IS NULL) linked
 JOIN zt_task t ON t.story=linked.story AND t.deleted='0' AND t.status!='closed' GROUP BY linked.windowID`
	if err := r.db.WithContext(ctx).Raw(consumedSQL, ids).Scan(&consumed).Error; err != nil {
		return nil, err
	}
	for _, row := range consumed {
		value := out[row.WindowID]
		value.Consumed = row.Total
		out[row.WindowID] = value
	}
	var stages []struct {
		WindowID     uint64 `gorm:"column:windowID"`
		DevCount     int    `gorm:"column:devCount"`
		TestCount    int    `gorm:"column:testCount"`
		DeliverCount int    `gorm:"column:deliverCount"`
	}
	const stagesSQL = `SELECT linked.windowID,
 SUM(s.stage='developing') AS devCount, SUM(s.stage='testing') AS testCount,
 SUM(s.stage IN ('verified','tested','delivering','delivered')) AS deliverCount
 FROM (SELECT DISTINCT vwp.versionWindow AS windowID, ps.story FROM zt_versionwindowproduct vwp
 JOIN zt_planstory ps ON ps.plan=vwp.plan WHERE vwp.versionWindow IN ? AND vwp.deletedAt IS NULL) linked
 JOIN zt_story s ON s.id=linked.story AND s.deleted='0' GROUP BY linked.windowID`
	if err := r.db.WithContext(ctx).Raw(stagesSQL, ids).Scan(&stages).Error; err != nil {
		return nil, err
	}
	for _, row := range stages {
		value := out[row.WindowID]
		value.DevCount = row.DevCount
		value.TestCount = row.TestCount
		value.DeliverCount = row.DeliverCount
		out[row.WindowID] = value
	}
	return out, nil
}
