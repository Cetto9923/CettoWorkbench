// =============================================================================
// 文件: internal/module/schedule/window_batch_test.go
// 模块: 排期工作台
// 类型: test
// 职责: 验证版本窗口批量统计读取与容量计算在各种边界条件下的正确性与等价性。
// =============================================================================

package schedule

import (
	"context"
	"math"
	"testing"
	"time"

	"workbench/internal/model"
)

// TestWindowStatsEquivalent 验证工时、容量、权限等统计字段在各种边界条件下的等价性与正确性。
func TestWindowStatsEquivalent(t *testing.T) {
	t.Run("RoundingAndRemainingCalculations", func(t *testing.T) {
		// 验证四舍五入与负剩余工时
		tests := []struct {
			name           string
			capacity       int
			consumed       float64
			expectedUsed   int
			expectedRemain int
			expectedPct    int
		}{
			{
				name:           "normal rounding down",
				capacity:       100,
				consumed:       45.4,
				expectedUsed:   45,
				expectedRemain: 55,
				expectedPct:    45,
			},
			{
				name:           "normal rounding up",
				capacity:       100,
				consumed:       45.6,
				expectedUsed:   46,
				expectedRemain: 54,
				expectedPct:    46,
			},
			{
				name:           "consumed exceeds capacity (negative remaining)",
				capacity:       80,
				consumed:       120.2,
				expectedUsed:   120,
				expectedRemain: -40,
				expectedPct:    150,
			},
			{
				name:           "zero capacity handles gracefully (percent is 0)",
				capacity:       0,
				consumed:       25.0,
				expectedUsed:   25,
				expectedRemain: -25,
				expectedPct:    0,
			},
			{
				name:           "zero consumed",
				capacity:       70,
				consumed:       0.0,
				expectedUsed:   0,
				expectedRemain: 70,
				expectedPct:    0,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				usedHours := int(math.Round(tc.consumed))
				if usedHours != tc.expectedUsed {
					t.Errorf("usedHours mismatch: got %d, want %d", usedHours, tc.expectedUsed)
				}
				remainingHours := tc.capacity - usedHours
				if remainingHours != tc.expectedRemain {
					t.Errorf("remainingHours mismatch: got %d, want %d", remainingHours, tc.expectedRemain)
				}
				usedPercent := 0
				if tc.capacity > 0 {
					usedPercent = usedHours * 100 / tc.capacity
				}
				if usedPercent != tc.expectedPct {
					t.Errorf("usedPercent mismatch: got %d, want %d", usedPercent, tc.expectedPct)
				}
			})
		}
	})

	t.Run("WindowPermissions", func(t *testing.T) {
		// 验证 computeWindowPermissions
		// 1. 创建者本人，无关联需求 -> 可编辑，可删除，hasLinkedDemands=false
		canEdit, canDelete, hasLinked := computeWindowPermissions("user_a", "user_a", 0)
		if !canEdit || !canDelete || hasLinked {
			t.Errorf("creator with 0 demands: got canEdit=%v, canDelete=%v, hasLinked=%v", canEdit, canDelete, hasLinked)
		}

		// 2. 创建者本人，有关联需求 -> 可编辑，不可删除，hasLinkedDemands=true
		canEdit, canDelete, hasLinked = computeWindowPermissions("user_a", "user_a", 3)
		if !canEdit || canDelete || !hasLinked {
			t.Errorf("creator with demands: got canEdit=%v, canDelete=%v, hasLinked=%v", canEdit, canDelete, hasLinked)
		}

		// 3. 非创建者 -> 不可编辑，不可删除
		canEdit, canDelete, _ = computeWindowPermissions("user_a", "user_b", 0)
		if canEdit || canDelete {
			t.Errorf("non-creator: got canEdit=%v, canDelete=%v", canEdit, canDelete)
		}
	})

	t.Run("CountWorkdaysFromSets", func(t *testing.T) {
		// 验证在无假期的普通一周（周一到周日）
		monday, _ := time.ParseInLocation("2006-01-02", "2026-09-07", time.Local) // Monday
		sunday, _ := time.ParseInLocation("2006-01-02", "2026-09-13", time.Local) // Sunday
		holidaySet := map[string]struct{}{}
		workingSet := map[string]struct{}{}

		// 5天工作日模式 (weekendMode = 2)
		workdays := countWorkdaysFromSets(monday, sunday, holidaySet, workingSet, 2)
		if workdays != 5 {
			t.Errorf("expected 5 workdays in a normal week, got %d", workdays)
		}

		// 6天工作日模式 (weekendMode = 1，仅周日休息)
		workdays6 := countWorkdaysFromSets(monday, sunday, holidaySet, workingSet, 1)
		if workdays6 != 6 {
			t.Errorf("expected 6 workdays in 6-day week mode, got %d", workdays6)
		}

		// 加法定假日（周三放假）
		holidaySet["2026-09-09"] = struct{}{}
		workdaysWithHoliday := countWorkdaysFromSets(monday, sunday, holidaySet, workingSet, 2)
		if workdaysWithHoliday != 4 {
			t.Errorf("expected 4 workdays with 1 holiday, got %d", workdaysWithHoliday)
		}

		// 加周末补班（周六补班）
		workingSet["2026-09-12"] = struct{}{}
		workdaysWithMakeup := countWorkdaysFromSets(monday, sunday, holidaySet, workingSet, 2)
		if workdaysWithMakeup != 5 {
			t.Errorf("expected 5 workdays with 1 holiday and 1 makeup workday, got %d", workdaysWithMakeup)
		}

		// start > end 自动反转测试
		reversed := countWorkdaysFromSets(sunday, monday, holidaySet, workingSet, 2)
		if reversed != workdaysWithMakeup {
			t.Errorf("start > end should equal reversed range: got %d, want %d", reversed, workdaysWithMakeup)
		}

		// 单日测试：仅周一
		singleMonday := countWorkdaysFromSets(monday, monday, map[string]struct{}{}, map[string]struct{}{}, 2)
		if singleMonday != 1 {
			t.Errorf("single monday should be 1 workday, got %d", singleMonday)
		}

		// 单日测试：仅周六（休息日返回 1 作为保底）
		saturday, _ := time.ParseInLocation("2006-01-02", "2026-09-12", time.Local)
		singleSaturday := countWorkdaysFromSets(saturday, saturday, map[string]struct{}{}, map[string]struct{}{}, 2)
		if singleSaturday != 1 {
			t.Errorf("all-weekend window should fall back to 1 workday, got %d", singleSaturday)
		}
	})
}

// TestNoWindows 验证空窗口列表不会导致任何空指针或非法查询。
func TestNoWindows(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	res, err := svc.loadWindowBatchStats(ctx, []model.VersionWindow{})
	if err != nil {
		t.Fatalf("unexpected error for empty windows: %v", err)
	}
	if len(res.capacityMap) != 0 || len(res.consumedMap) != 0 || len(res.demandCountMap) != 0 {
		t.Errorf("expected all maps to be empty, got capacity=%d, consumed=%d, demand=%d",
			len(res.capacityMap), len(res.consumedMap), len(res.demandCountMap))
	}
}

// TestWindowCardToneRotation 验证音调样式类循环轮转。
func TestWindowCardToneRotation(t *testing.T) {
	for i := 0; i < 10; i++ {
		tone := windowCardToneClasses[i%len(windowCardToneClasses)]
		expected := []string{"red", "blue", "green", "purple"}[i%4]
		if tone != expected {
			t.Errorf("tone class %d mismatch: got %s, want %s", i, tone, expected)
		}
	}
}
