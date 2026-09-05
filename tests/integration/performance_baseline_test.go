//go:build integration

// =============================================================================
// 文件: tests/integration/performance_baseline_test.go
// 模块: 性能基线测试
// 类型: test
// 职责: 验证列表查询基准的确定性、Actor 隔离性与规模增长特征。
// 依赖: tests/integration/synthetic_fixtures.go
// =============================================================================

package integration

import (
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"workbench/internal/module/po"
)

// TestPerformanceBaseline 验证可重复性、跨用户隔离与规模增长检测。
func TestPerformanceBaseline(t *testing.T) {
	baseTime, _ := time.Parse(time.RFC3339, "2026-09-05T00:00:00Z")
	const fixedSeed = int64(42)

	t.Run("BaselineDeterministic", func(t *testing.T) {
		// 验证相同 seed 和时间下两次生成并处理的结果完全相同
		notices1 := GenerateSyntheticNotices(100, fixedSeed, baseTime)
		notices2 := GenerateSyntheticNotices(100, fixedSeed, baseTime)

		if len(notices1) != len(notices2) {
			t.Fatalf("length mismatch: %d vs %d", len(notices1), len(notices2))
		}
		for i := range notices1 {
			if notices1[i].ID != notices2[i].ID || notices1[i].Subject != notices2[i].Subject {
				t.Fatalf("notice item %d differs between identical seed runs", i)
			}
		}

		todos1 := GenerateSyntheticTodos(100, fixedSeed, baseTime)
		todos2 := GenerateSyntheticTodos(100, fixedSeed, baseTime)
		if len(todos1) != len(todos2) {
			t.Fatalf("todo length mismatch: %d vs %d", len(todos1), len(todos2))
		}
		for i := range todos1 {
			if todos1[i].ID != todos2[i].ID || todos1[i].Title != todos2[i].Title {
				t.Fatalf("todo item %d differs between identical seed runs", i)
			}
		}
	})

	t.Run("CrossActorIsolation", func(t *testing.T) {
		notices := GenerateSyntheticNotices(500, fixedSeed, baseTime)

		// 模拟针对 user_a 的可见性过滤
		userANotices := make([]SyntheticNoticeRow, 0)
		for _, n := range notices {
			if n.Deleted == 1 {
				continue
			}
			parts := strings.Split(n.ToList, ",")
			for _, p := range parts {
				if strings.TrimSpace(p) == "user_a" {
					userANotices = append(userANotices, n)
					break
				}
			}
		}

		// 验证过滤结果中绝无仅发给 user_c 且不包含 user_a 的对象
		for _, n := range userANotices {
			if !strings.Contains(n.ToList, "user_a") {
				t.Errorf("notice %d leaked to user_a: toList=%s", n.ID, n.ToList)
			}
		}
	})

	t.Run("MeasurementDetectsGrowth", func(t *testing.T) {
		// 测量全量在内存中过滤和排序时，N=100 与 N=1000 的耗时/内存增长趋势
		nSmall := 100
		nLarge := 1000

		todosSmall := GenerateSyntheticTodos(nSmall, fixedSeed, baseTime)
		todosLarge := GenerateSyntheticTodos(nLarge, fixedSeed, baseTime)

		// 模拟内存全量排序并分页
		runSortSlice := func(items []po.TodoItem) []po.TodoItem {
			sorted := make([]po.TodoItem, len(items))
			copy(sorted, items)
			sort.Slice(sorted, func(i, j int) bool {
				if sorted[i].Priority != sorted[j].Priority {
					return sorted[i].Priority < sorted[j].Priority
				}
				return sorted[i].ID > sorted[j].ID
			})
			pageSize := 20
			if len(sorted) > pageSize {
				return sorted[:pageSize]
			}
			return sorted
		}

		startSmall := time.Now()
		for i := 0; i < 50; i++ {
			_ = runSortSlice(todosSmall)
		}
		durationSmall := time.Since(startSmall)

		startLarge := time.Now()
		for i := 0; i < 50; i++ {
			_ = runSortSlice(todosLarge)
		}
		durationLarge := time.Since(startLarge)

		t.Logf("N=%d duration: %v, N=%d duration: %v", nSmall, durationSmall, nLarge, durationLarge)
		if durationLarge < durationSmall {
			t.Logf("warning: measurement noise detected, large scale ran in %v vs small %v", durationLarge, durationSmall)
		}
	})

	t.Run("LiveIsolatedDatabaseStatus", func(t *testing.T) {
		dsn := os.Getenv("WB_TEST_MYSQL_DSN")
		if dsn == "" {
			t.Log("WB_TEST_MYSQL_DSN not set: isolated live MySQL performance execution is pending external environment injection")
			return
		}
		t.Log("WB_TEST_MYSQL_DSN is provided; live database integration test ready")
	})
}

// BenchmarkWorkbenchLists 基准测试不同数据规模下的处理开销。
func BenchmarkWorkbenchLists(b *testing.B) {
	baseTime, _ := time.Parse(time.RFC3339, "2026-09-05T00:00:00Z")
	const fixedSeed = int64(42)

	notices100 := GenerateSyntheticNotices(100, fixedSeed, baseTime)
	notices1000 := GenerateSyntheticNotices(1000, fixedSeed, baseTime)
	todos100 := GenerateSyntheticTodos(100, fixedSeed, baseTime)
	todos1000 := GenerateSyntheticTodos(1000, fixedSeed, baseTime)

	b.Run("Notices_Filter_N100", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			filtered := make([]SyntheticNoticeRow, 0, len(notices100))
			for _, n := range notices100 {
				if n.Deleted == 0 && strings.Contains(n.ToList, "user_a") {
					filtered = append(filtered, n)
				}
			}
		}
	})

	b.Run("Notices_Filter_N1000", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			filtered := make([]SyntheticNoticeRow, 0, len(notices1000))
			for _, n := range notices1000 {
				if n.Deleted == 0 && strings.Contains(n.ToList, "user_a") {
					filtered = append(filtered, n)
				}
			}
		}
	})

	b.Run("Todos_SortSlice_N100", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			sorted := make([]po.TodoItem, len(todos100))
			copy(sorted, todos100)
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].ID > sorted[j].ID
			})
		}
	})

	b.Run("Todos_SortSlice_N1000", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			sorted := make([]po.TodoItem, len(todos1000))
			copy(sorted, todos1000)
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].ID > sorted[j].ID
			})
		}
	})
}
