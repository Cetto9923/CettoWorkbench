package po

import "testing"

// 零日期（含驱动解析出的 0001-01-01）必须视为未填，不能算出超期。
func TestCalcOverdue_ZeroDatesAreUnset(t *testing.T) {
	for _, in := range []string{"", "0000-00-00", "0001-01-01", "0001-01-01T00:00:00Z"} {
		d, overdue, days := calcOverdue(in, "active")
		if d != "" || overdue || days != 0 {
			t.Fatalf("calcOverdue(%q) = (%q, %v, %d), want unset", in, d, overdue, days)
		}
	}
	if _, overdue, days := calcOverdue("2020-01-01", "active"); !overdue || days <= 0 {
		t.Fatalf("expected past date to be overdue, got overdue=%v days=%d", overdue, days)
	}
}

// 终态（已关闭 / 已驳回 / 已发布）保留截止日展示，但不再算超期，与 SQL 侧口径一致。
func TestCalcOverdue_TerminalStatusNotOverdue(t *testing.T) {
	for _, st := range []string{"closed", "refuse", "released", " refuse "} {
		d, overdue, days := calcOverdue("2020-01-01", st)
		if d != "2020-01-01" || overdue || days != 0 {
			t.Fatalf("calcOverdue(2020-01-01, %q) = (%q, %v, %d), want terminal not overdue", st, d, overdue, days)
		}
	}
}
