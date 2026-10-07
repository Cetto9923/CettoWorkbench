// =============================================================================
// 文件: internal/module/po/deliverydeadline_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证评审截止边界、无效日期和统一覆盖配置。
// =============================================================================
package po

import (
	"testing"
	"time"
)

func TestDeliveryDeadlineBoundary(t *testing.T) {
	cfg := deliveryDeadlineConfig{Windows: "2026-09-30,2026-10-08|2026-10-09,2026-10-15", Clock: "12:30"}
	for _, tc := range []struct {
		launch, now, date string
		overdue           bool
	}{
		{"2026-10-08", "2026-09-30 12:29", "2026-09-30", false},
		{"2026-10-08", "2026-09-30 12:30", "2026-09-30", false},
		{"2026-10-08", "2026-09-30 12:31", "2026-09-30", true},
		{"2026-10-09", "2026-10-07 23:00", "2026-10-09", false},
		{"", "2026-10-07 23:00", "", false},
		{"0000-00-00", "2026-10-07 23:00", "", false},
		{"2099-12-31", "2026-10-07 23:00", "", false},
		{"2026-10-20", "2026-10-07 23:00", "", false},
	} {
		now, err := time.ParseInLocation("2006-01-02 15:04", tc.now, time.Local)
		if err != nil {
			t.Fatal(err)
		}
		date, overdue, _ := cfg.overdue(tc.launch, now)
		if date != tc.date || overdue != tc.overdue {
			t.Fatalf("%+v: date=%s overdue=%v", tc, date, overdue)
		}
	}
	cfg.Date = "2026-10-06"
	if got := cfg.deadline("2026-10-09"); got != cfg.Date {
		t.Fatal(got)
	}
	cfg.Date = "2099-12-31"
	if _, overdue, _ := cfg.overdue("2026-10-08", time.Now()); overdue {
		t.Fatal("sentinel date must not be overdue")
	}
	if sql, _ := cfg.overdueSQL("estimateLaunch", time.Now()); sql != "1 = 0" {
		t.Fatal(sql)
	}
}
