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
	start, end = orderedDates(start, end)

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

	return countCalendarDays(start, end, holidaySet, workingSet, weekendMode), nil
}

func countCalendarDays(start, end time.Time, holidaySet, workingSet map[string]struct{}, weekendMode int) int {
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

func (s *Service) windowCapacities(ctx context.Context, windows []model.VersionWindow) (map[uint64]int, error) {
	out := make(map[uint64]int, len(windows))
	if len(windows) == 0 {
		return out, nil
	}
	begin, end := windowStart(windows[0]), windows[0].ReleaseDate
	for _, w := range windows {
		start, finish := orderedDates(windowStart(w), w.ReleaseDate)
		if start.Before(begin) {
			begin = start
		}
		if finish.After(end) {
			end = finish
		}
	}
	holidays, err := s.repo.GetHolidays(ctx, begin.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	working, err := s.repo.GetWorkingDays(ctx, begin.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	hours, weekend, err := s.repo.GetWorkhoursConfig(ctx)
	if err != nil {
		hours, weekend = 7, 2
	}
	if hours == 0 {
		hours = 7
	}
	holidaySet, workingSet := holidayDatesSet(holidays), holidayDatesSet(working)
	for _, w := range windows {
		start, finish := orderedDates(windowStart(w), w.ReleaseDate)
		size := int(w.GroupSize)
		if size == 0 {
			size = 1
		}
		out[w.ID] = countCalendarDays(dateOnly(start), dateOnly(finish), holidaySet, workingSet, weekend) * hours * size
	}
	return out, nil
}

func windowStart(w model.VersionWindow) time.Time {
	if w.StartDate != nil {
		return *w.StartDate
	}
	return w.ReleaseDate
}

func windowIDs(windows []model.VersionWindow) []uint64 {
	ids := make([]uint64, 0, len(windows))
	for _, w := range windows {
		ids = append(ids, w.ID)
	}
	return ids
}

func orderedDates(start, end time.Time) (time.Time, time.Time) {
	if end.Before(start) {
		return end, start
	}
	return start, end
}
