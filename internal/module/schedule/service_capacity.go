// =============================================================================
// 文件: internal/module/schedule/service_capacity.go
// 模块: 排期工作台
// 类型: action
// 职责: 版本窗口容量与工作日计算逻辑。
// 依赖: internal/module/schedule/repo.go
// =============================================================================

package schedule

import (
	"context"
	"fmt"
	"strings"
	"time"

	"workbench/internal/model"
)

// CountActualWorkdays 按禅道规则统计实际工作日（含节假日与补班）。
func (s *Service) CountActualWorkdays(ctx context.Context, startDate, endDate string) (int, error) {
	start, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(startDate), time.Local)
	if err != nil {
		return 0, fmt.Errorf("invalid start date")
	}
	end, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(endDate), time.Local)
	if err != nil {
		return 0, fmt.Errorf("invalid end date")
	}
	if end.Before(start) {
		start, end = end, start
	}

	holidays, err := s.repo.GetHolidays(ctx, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		return 0, err
	}
	workingDays, err := s.repo.GetWorkingDays(ctx, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		return 0, err
	}
	_, weekendMode, err := s.repo.GetWorkhoursConfig(ctx)
	if err != nil {
		weekendMode = 2
	}

	holidaySet := holidayDatesSet(holidays)
	workingSet := holidayDatesSet(workingDays)

	return countWorkdaysFromSets(start, end, holidaySet, workingSet, weekendMode), nil
}

// CalcCapacityBatch 批量计算版本窗口容量工时。
// 聚合全部窗口的日期跨度 [minStart, maxEnd]，单次拉取节假日、补班日与工时配置，并在内存中完成各窗口容量计算。
func (s *Service) CalcCapacityBatch(ctx context.Context, windows []model.VersionWindow) (map[uint64]int, error) {
	result := make(map[uint64]int, len(windows))
	if len(windows) == 0 {
		return result, nil
	}

	var (
		minStart time.Time
		maxEnd   time.Time
		inited   bool
	)

	// 计算全局日期范围 [minStart, maxEnd]
	for _, w := range windows {
		startStr := w.ReleaseDate.Format("2006-01-02")
		if w.StartDate != nil {
			startStr = w.StartDate.Format("2006-01-02")
		}
		endStr := w.ReleaseDate.Format("2006-01-02")

		start, _ := time.ParseInLocation("2006-01-02", startStr, time.Local)
		end, _ := time.ParseInLocation("2006-01-02", endStr, time.Local)
		if end.Before(start) {
			start, end = end, start
		}

		if !inited {
			minStart = start
			maxEnd = end
			inited = true
		} else {
			if start.Before(minStart) {
				minStart = start
			}
			if end.After(maxEnd) {
				maxEnd = end
			}
		}
	}

	holidays, err := s.repo.GetHolidays(ctx, minStart.Format("2006-01-02"), maxEnd.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	workingDays, err := s.repo.GetWorkingDays(ctx, minStart.Format("2006-01-02"), maxEnd.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	hoursPerDay, weekendMode, err := s.repo.GetWorkhoursConfig(ctx)
	if err != nil {
		hoursPerDay = 7
		weekendMode = 2
	}
	if hoursPerDay == 0 {
		hoursPerDay = 7
	}

	holidaySet := holidayDatesSet(holidays)
	workingSet := holidayDatesSet(workingDays)

	for _, w := range windows {
		startStr := w.ReleaseDate.Format("2006-01-02")
		if w.StartDate != nil {
			startStr = w.StartDate.Format("2006-01-02")
		}
		endStr := w.ReleaseDate.Format("2006-01-02")

		start, _ := time.ParseInLocation("2006-01-02", startStr, time.Local)
		end, _ := time.ParseInLocation("2006-01-02", endStr, time.Local)
		if end.Before(start) {
			start, end = end, start
		}

		workdays := countWorkdaysFromSets(start, end, holidaySet, workingSet, weekendMode)
		groupSize := int(w.GroupSize)
		if groupSize == 0 {
			groupSize = 1
		}
		result[w.ID] = workdays * hoursPerDay * groupSize
	}

	return result, nil
}

func countWorkdaysFromSets(start, end time.Time, holidaySet, workingSet map[string]struct{}, weekendMode int) int {
	if end.Before(start) {
		start, end = end, start
	}
	count := 0
	for current := start; !current.After(end); current = current.AddDate(0, 0, 1) {
		dateKey := current.Format("2006-01-02")
		if _, ok := workingSet[dateKey]; ok {
			count++
			continue
		}
		if _, ok := holidaySet[dateKey]; ok {
			continue
		}
		if isConfiguredWeekend(current.Weekday(), weekendMode) {
			continue
		}
		count++
	}
	if count == 0 {
		return 1
	}
	return count
}

// CalcCapacity 计算版本窗口容量工时（工作日 × 每日工时 × 小组人数）。
func (s *Service) CalcCapacity(ctx context.Context, startDate, endDate string, groupSize int) (int, error) {
	workdays, err := s.CountActualWorkdays(ctx, startDate, endDate)
	if err != nil {
		return 0, err
	}
	hoursPerDay, _, err := s.repo.GetWorkhoursConfig(ctx)
	if err != nil {
		hoursPerDay = 7
	}
	if hoursPerDay == 0 {
		hoursPerDay = 7
	}
	if groupSize == 0 {
		groupSize = 1
	}
	return workdays * hoursPerDay * groupSize, nil
}

func holidayDatesSet(rows []ZtHoliday) map[string]struct{} {
	set := make(map[string]struct{})
	for _, row := range rows {
		begin, ok1 := parseHolidayDate(row.Begin)
		end, ok2 := parseHolidayDate(row.End)
		if !ok1 || !ok2 {
			continue
		}
		begin = dateOnly(begin)
		end = dateOnly(end)
		if end.Before(begin) {
			begin, end = end, begin
		}
		for current := begin; !current.After(end); current = current.AddDate(0, 0, 1) {
			set[current.Format("2006-01-02")] = struct{}{}
		}
	}
	return set
}

func parseHolidayDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	if parsed, err := time.ParseInLocation("2006-01-02", value, time.Local); err == nil {
		return parsed, true
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.In(time.Local), true
	}
	return time.Time{}, false
}

func isConfiguredWeekend(weekday time.Weekday, weekendMode int) bool {
	if weekendMode == 1 {
		return weekday == time.Sunday
	}
	return weekday == time.Saturday || weekday == time.Sunday
}
