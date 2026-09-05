// =============================================================================
// 文件: internal/module/schedule/repo_capacity.go
// 模块: 排期工作台
// 类型: action
// 职责: 禅道节假日与工时配置只读查询（容量计算用）。
// 依赖: internal/module/schedule/repo.go
// =============================================================================

package schedule

import (
	"context"
	"strconv"
	"strings"
)

// ZtHoliday 表示禅道 zt_holiday 表只读字段。
type ZtHoliday struct {
	ID    uint   `gorm:"column:id"`
	Name  string `gorm:"column:name"`
	Type  string `gorm:"column:type"`
	Begin string `gorm:"column:begin"`
	End   string `gorm:"column:end"`
}

// TableName 指定 zt_holiday 表。
func (ZtHoliday) TableName() string {
	return "zt_holiday"
}

// GetHolidays 获取与指定日期范围重叠的法定节假日。
func (r *Repo) GetHolidays(ctx context.Context, begin, end string) ([]ZtHoliday, error) {
	const query = `
SELECT id, name, type, ` + "`begin`" + `, ` + "`end`" + `
FROM zt_holiday
WHERE type = 'holiday' AND ` + "`begin`" + ` <= ? AND ` + "`end`" + ` >= ?`

	var rows []ZtHoliday
	if err := r.db.WithContext(ctx).Raw(query, end, begin).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetWorkingDays 获取与指定日期范围重叠的补班日。
func (r *Repo) GetWorkingDays(ctx context.Context, begin, end string) ([]ZtHoliday, error) {
	const query = `
SELECT id, name, type, ` + "`begin`" + `, ` + "`end`" + `
FROM zt_holiday
WHERE type = 'working' AND ` + "`begin`" + ` <= ? AND ` + "`end`" + ` >= ?`

	var rows []ZtHoliday
	if err := r.db.WithContext(ctx).Raw(query, end, begin).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

type workhoursConfigRow struct {
	Key   string `gorm:"column:key"`
	Value string `gorm:"column:value"`
}

// GetWorkhoursConfig 读取 execution 模块的每日工时与周末规则配置。
func (r *Repo) GetWorkhoursConfig(ctx context.Context) (int, int, error) {
	const query = `
SELECT ` + "`key`" + `, value
FROM zt_config
WHERE module = 'execution' AND ` + "`key`" + ` IN ('defaultWorkhours', 'weekend')`

	var rows []workhoursConfigRow
	if err := r.db.WithContext(ctx).Raw(query).Scan(&rows).Error; err != nil {
		return 7, 2, err
	}

	defaultWorkhours := 7
	weekend := 2
	for _, row := range rows {
		value, err := strconv.Atoi(strings.TrimSpace(row.Value))
		if err != nil {
			continue
		}
		switch row.Key {
		case "defaultWorkhours":
			defaultWorkhours = value
		case "weekend":
			weekend = value
		}
	}
	return defaultWorkhours, weekend, nil
}

const windowBatchChunkSize = 200

// GetWindowConsumedHoursBatch 批量查询窗口关联任务的已消耗工时总和。
// 链路: zt_versionwindowproduct.plan → zt_planstory.story → zt_task.consumed
// 按 windowBatchChunkSize (B=200) 分块执行，消除 N+1 扇出。
func (r *Repo) GetWindowConsumedHoursBatch(ctx context.Context, windowIDs []uint64) (map[uint64]float64, error) {
	result := make(map[uint64]float64, len(windowIDs))
	if len(windowIDs) == 0 {
		return result, nil
	}

	const query = `
SELECT vwp.versionWindow AS window_id, COALESCE(SUM(t.consumed), 0) AS total
FROM zt_versionwindowproduct vwp
JOIN zt_planstory ps ON ps.plan = vwp.plan
JOIN zt_task t ON t.story = ps.story AND t.deleted = '0' AND t.status != 'closed'
WHERE vwp.versionWindow IN ? AND vwp.deletedAt IS NULL AND vwp.plan IS NOT NULL
GROUP BY vwp.versionWindow`

	for i := 0; i < len(windowIDs); i += windowBatchChunkSize {
		end := i + windowBatchChunkSize
		if end > len(windowIDs) {
			end = len(windowIDs)
		}
		chunk := windowIDs[i:end]

		var rows []struct {
			WindowID uint64  `gorm:"column:window_id"`
			Total    float64 `gorm:"column:total"`
		}
		if err := r.db.WithContext(ctx).Raw(query, chunk).Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			result[row.WindowID] = row.Total
		}
	}
	return result, nil
}

// GetWindowDemandCountBatch 批量查询窗口关联的需求数量（业需去重 + 独立软需）。
// 链路: zt_versionwindowproduct.plan → zt_planstory.story → zt_story
// 按 windowBatchChunkSize (B=200) 分块执行，消除 N+1 扇出。
func (r *Repo) GetWindowDemandCountBatch(ctx context.Context, windowIDs []uint64) (map[uint64]int, error) {
	result := make(map[uint64]int, len(windowIDs))
	if len(windowIDs) == 0 {
		return result, nil
	}

	const query = `
SELECT
  vwp.versionWindow AS window_id,
  COUNT(DISTINCT CASE WHEN s.sourceType = 'demandpool' AND s.fromDemand > 0 THEN s.fromDemand ELSE NULL END)
  + COUNT(CASE WHEN IFNULL(s.sourceType, '') != 'demandpool' THEN 1 ELSE NULL END) AS demand_count
FROM zt_versionwindowproduct vwp
JOIN zt_planstory ps ON ps.plan = vwp.plan
JOIN zt_story s ON s.id = ps.story AND s.deleted = '0'
WHERE vwp.versionWindow IN ? AND vwp.deletedAt IS NULL AND vwp.plan IS NOT NULL
GROUP BY vwp.versionWindow`

	for i := 0; i < len(windowIDs); i += windowBatchChunkSize {
		end := i + windowBatchChunkSize
		if end > len(windowIDs) {
			end = len(windowIDs)
		}
		chunk := windowIDs[i:end]

		var rows []struct {
			WindowID    uint64 `gorm:"column:window_id"`
			DemandCount int64  `gorm:"column:demand_count"`
		}
		if err := r.db.WithContext(ctx).Raw(query, chunk).Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			result[row.WindowID] = int(row.DemandCount)
		}
	}
	return result, nil
}
