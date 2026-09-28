package po

import "testing"

// 零日期（含驱动解析出的 0001-01-01）必须视为未填，不能算出超期。
func TestCalcOverdue_ZeroDatesAreUnset(t *testing.T) {
	for _, in := range []string{"", "0000-00-00", "0001-01-01", "0001-01-01T00:00:00Z"} {
		d, overdue, days := calcOverdue(in)
		if d != "" || overdue || days != 0 {
			t.Fatalf("calcOverdue(%q) = (%q, %v, %d), want unset", in, d, overdue, days)
		}
	}
	if _, overdue, days := calcOverdue("2020-01-01"); !overdue || days <= 0 {
		t.Fatalf("expected past date to be overdue, got overdue=%v days=%d", overdue, days)
	}
}
