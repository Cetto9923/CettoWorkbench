package schedule

import (
	"context"
	"fmt"
	"time"
)

// WorkdayOffset uses the same holiday, make-up day and weekend rules as capacity.
func (s *Service) WorkdayOffset(ctx context.Context, start time.Time, count int) (time.Time, error) {
	end := start.AddDate(0, 0, 90)
	holidays, err := s.repo.GetHolidays(ctx, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		return time.Time{}, err
	}
	working, err := s.repo.GetWorkingDays(ctx, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		return time.Time{}, err
	}
	_, weekend, err := s.repo.GetWorkhoursConfig(ctx)
	if err != nil {
		return time.Time{}, err
	}
	hs, ws := holidayDatesSet(holidays), holidayDatesSet(working)
	for day := start.AddDate(0, 0, 1); !day.After(end); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		_, work := ws[key]
		_, holiday := hs[key]
		if work || (!holiday && !isConfiguredWeekend(day.Weekday(), weekend)) {
			count--
			if count == 0 {
				return day, nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("工作日范围不足")
}
