package po

import (
	"testing"
	"time"
)

// 终态集合是逾期口径的唯一事实源；引号定界，closed 不得误配到 closedX。
func TestIsTerminal(t *testing.T) {
	for _, st := range []string{"closed", "refuse", "released", " refuse ", "Closed"} {
		if !isTerminal(st) {
			t.Errorf("isTerminal(%q) = false, want true", st)
		}
	}
	for _, st := range []string{"", "active", "closedX", "refuse2", "hang", "done"} {
		if isTerminal(st) {
			t.Errorf("isTerminal(%q) = true, want false", st)
		}
	}
}

// dayDiff 按自然日而非 24 小时时长：当天到期为 0，昨天为 1。
// 08:00 之后 UTC 截断会把当天到期算成 -1，必须挡住。
func TestDayDiff_CalendarDayNotDuration(t *testing.T) {
	base := time.Date(2026, 10, 2, 8, 30, 0, 0, time.Local)
	if got := dayDiff(base, base); got != 0 {
		t.Errorf("dayDiff(同刻) = %d, want 0", got)
	}
	if got := dayDiff(base, base.Add(6*time.Hour)); got != 0 {
		t.Errorf("dayDiff(+6h 跨零点前) = %d, want 0", got)
	}
	if got := dayDiff(base.AddDate(0, 0, -1), base); got != 1 {
		t.Errorf("dayDiff(昨天→今天) = %d, want 1", got)
	}
	if got := dayDiff(base, base.AddDate(0, 0, 3)); got != 3 {
		t.Errorf("dayDiff(今天→3 天后) = %d, want 3", got)
	}
}

// todayStr 必须给本地自然日；东八区 00:30 时 UTC 仍是前一天。
func TestTodayStr_IsLocalCalendarDay(t *testing.T) {
	want := time.Now().Format("2006-01-02")
	if got := todayStr(); got != want {
		t.Errorf("todayStr() = %q, want %q", got, want)
	}
}
