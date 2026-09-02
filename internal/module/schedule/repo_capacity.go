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
