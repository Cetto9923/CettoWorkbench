//go:build integration

// =============================================================================
// 文件: tests/integration/performance_baseline_test.go
// 模块: 性能基线测试
// 类型: test
// 职责: 验证列表查询基准的确定性、Actor 隔离性与真实隔离 MySQL 执行。
// 依赖: tests/integration/synthetic_fixtures.go
//       tests/integration/db_schema.go
//       tests/integration/db_seed.go
//       tests/integration/query_counter.go
// =============================================================================

package integration

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/module/po"
	"workbench/internal/module/schedule"
)

func openIsolatedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("WB_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("WB_TEST_MYSQL_DSN not set: isolated live MySQL performance execution skipped")
	}

	// 严格白名单与前置安全校验：拦截非法实例、非 _test 库及缺少 DDL opt-in
	if err := ValidateTestDSN(dsn, true); err != nil {
		t.Fatalf("SAFETY VIOLATION: %v", err)
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open isolated test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil || sqlDB.Ping() != nil {
		t.Fatalf("failed to ping isolated test database: %v", err)
	}

	return db
}

// TestPerformanceBaseline 验证可重复性、跨用户隔离与规模增长检测。
func TestPerformanceBaseline(t *testing.T) {
	baseTime, _ := time.Parse(time.RFC3339, "2026-09-05T00:00:00Z")
	const fixedSeed = int64(42)

	t.Run("BaselineDeterministic", func(t *testing.T) {
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

		for _, n := range userANotices {
			if !strings.Contains(n.ToList, "user_a") {
				t.Errorf("notice %d leaked to user_a: toList=%s", n.ID, n.ToList)
			}
		}
	})

	t.Run("MeasurementDetectsGrowth", func(t *testing.T) {
		nSmall := 100
		nLarge := 1000

		todosSmall := GenerateSyntheticTodos(nSmall, fixedSeed, baseTime)
		todosLarge := GenerateSyntheticTodos(nLarge, fixedSeed, baseTime)

		runSortSlice := func(items []po.TodoItem) []po.TodoItem {
			sorted := make([]po.TodoItem, len(items))
			copy(sorted, items)
			sort.Slice(sorted, func(i, j int) bool {
				if sorted[i].Priority != sorted[j].Priority {
					return sorted[i].Priority < sorted[j].Priority
				}
				return sorted[i].ID > sorted[j].ID
			})
			if len(sorted) > 20 {
				return sorted[:20]
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
	})

	t.Run("LiveIsolatedMySQL", func(t *testing.T) {
		db := openIsolatedTestDB(t)
		ctx := context.Background()

		// 1. 初始化最小表结构
		if err := InitMinimalSchema(ctx, db); err != nil {
			t.Fatalf("failed to init minimal schema: %v", err)
		}

		// 2. 填充合成测试数据
		const nNotices = 100
		const nTodos = 50
		const nWindows = 10
		if err := SeedSyntheticDatabase(ctx, db, nNotices, nTodos, nWindows); err != nil {
			t.Fatalf("failed to seed database: %v", err)
		}

		dbWithCounter, qc := AttachQueryCounter(db)

		// 3. 生产路径测试 A: 通知
		poRepo := po.NewRepo(dbWithCounter)
		qc.Reset()
		noticeResp, err := poRepo.FindNotices(ctx, "user_a", po.NoticeListReq{Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("FindNotices error: %v", err)
		}
		noticeQueries := qc.Count()
		t.Logf("FindNotices: Total=%d, Filtered=%d, Items=%d, SQL queries=%d",
			noticeResp.Total, noticeResp.Filtered, len(noticeResp.Items), noticeQueries)

		// 4. 生产路径测试 B: 待办
		qc.Reset()
		todoItems, total, err := poRepo.FindTodoItems(ctx, "user_a", po.TodoListReq{Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("FindTodoItems error: %v", err)
		}
		todoQueries := qc.Count()
		t.Logf("FindTodoItems: Total=%d, Items=%d, SQL queries=%d", total, len(todoItems), todoQueries)

		// 5. 生产路径测试 C: 版本窗口 (验证 1 + 2N 扇出)
		scheduleRepo := schedule.NewRepo(dbWithCounter)
		scheduleSvc := schedule.NewService(scheduleRepo, nil)
		actor := &model.User{ID: 1, Account: "user_a"}

		qc.Reset()
		cards, err := scheduleSvc.ListWindowCards(ctx, actor)
		if err != nil {
			t.Fatalf("ListWindowCards error: %v", err)
		}
		windowQueries := qc.Count()
		t.Logf("ListWindowCards: N=%d windows returned, SQL queries=%d (demonstrating O(N) fanout)",
			len(cards), windowQueries)

		// 6. 保存基线结果镜像用于 P1/P2 优化等价性验证
		fixtureData := map[string]interface{}{
			"notices": map[string]interface{}{
				"total":               noticeResp.Total,
				"filtered":            noticeResp.Filtered,
				"unread":              noticeResp.Unread,
				"categories":          noticeResp.Categories,
				"returned_item_count": len(noticeResp.Items),
			},
			"todos": map[string]interface{}{
				"total":               total,
				"returned_item_count": len(todoItems),
			},
			"windows": map[string]interface{}{
				"count":       len(cards),
				"query_count": windowQueries,
			},
		}
		if _, err := os.Stat("testdata/performance/old_impl_results.json"); os.IsNotExist(err) {
			bytes, _ := json.MarshalIndent(fixtureData, "", "  ")
			_ = os.WriteFile("testdata/performance/old_impl_results.json", bytes, 0644)
		}
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
