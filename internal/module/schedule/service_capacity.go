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
		return 1, nil
	}
	return count, nil
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
