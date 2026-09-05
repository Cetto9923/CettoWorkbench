//go:build integration

// =============================================================================
// 文件: tests/integration/window_queries_test.go
// 模块: 性能与版本窗口查询集成测试 (P3)
// 类型: test
// 职责: 验证版本窗口列表批量统计读取、SQL 查询次数恒定（不随窗口数增长）以及业务等价性。
// 依赖: tests/integration/db_schema.go, tests/integration/db_seed.go, tests/integration/query_counter.go
// =============================================================================

package integration

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/module/schedule"
)

func seedTestDatabase(ctx context.Context, db *gorm.DB, nNotices, nTodos, nWindows int) error {
	_ = db.WithContext(ctx).Exec("DELETE FROM zt_teamgroup").Error
	_ = db.WithContext(ctx).Exec("DELETE FROM zt_product").Error
	return SeedSyntheticDatabase(ctx, db, nNotices, nTodos, nWindows)
}

func TestWindowQueries(t *testing.T) {
	db := openIsolatedTestDB(t)
	ctx := context.Background()

	// 初始化隔离数据库表结构
	if err := InitMinimalSchema(ctx, db); err != nil {
		t.Fatalf("failed to init minimal schema: %v", err)
	}

	actor := &model.User{ID: 1, Account: "user_a"}
	dbWithCounter, qc := AttachQueryCounter(db)
	repo := schedule.NewRepo(dbWithCounter)
	svc := schedule.NewService(repo, nil)

	t.Run("NoWindows", func(t *testing.T) {
		// 空窗口列表验证：数据库无窗口时，查询次数为固定 2 次（count + find），无多余 SQL
		if err := seedTestDatabase(ctx, db, 10, 10, 0); err != nil {
			t.Fatalf("failed to seed empty database: %v", err)
		}

		qc.Reset()
		cards, err := svc.ListWindowCards(ctx, actor)
		if err != nil {
			t.Fatalf("ListWindowCards error: %v", err)
		}
		if len(cards) != 0 {
			t.Errorf("expected 0 cards, got %d", len(cards))
		}
		cardQueries := qc.Count()
		if cardQueries != 2 {
			t.Errorf("expected exactly 2 queries (count + find) for empty windows, got %d", cardQueries)
		}

		qc.Reset()
		resp, err := svc.ListWindows(ctx, actor)
		if err != nil {
			t.Fatalf("ListWindows error: %v", err)
		}
		if len(resp.Windows) != 0 {
			t.Errorf("expected 0 windows, got %d", len(resp.Windows))
		}
		windowQueries := qc.Count()
		if windowQueries != 2 {
			t.Errorf("expected exactly 2 queries for empty windows, got %d", windowQueries)
		}
	})

	t.Run("WindowQueryCount", func(t *testing.T) {
		// 验证查询次数恒定性：N=1, N=10, N=50 窗口时，查询次数保持恒定（8 次），消除 O(N) 扇出
		counts := []int{1, 10, 50}
		cardQueryCounts := make(map[int]int64)
		listQueryCounts := make(map[int]int64)

		for _, n := range counts {
			if err := seedTestDatabase(ctx, db, 10, 10, n); err != nil {
				t.Fatalf("failed to seed database for N=%d: %v", n, err)
			}

			// 测试 ListWindowCards
			qc.Reset()
			cards, err := svc.ListWindowCards(ctx, actor)
			if err != nil {
				t.Fatalf("ListWindowCards N=%d error: %v", n, err)
			}
			if len(cards) != n {
				t.Fatalf("ListWindowCards N=%d returned %d cards", n, len(cards))
			}
			cardQueryCounts[n] = qc.Count()

			// 测试 ListWindows
			qc.Reset()
			resp, err := svc.ListWindows(ctx, actor)
			if err != nil {
				t.Fatalf("ListWindows N=%d error: %v", n, err)
			}
			if len(resp.Windows) != n {
				t.Fatalf("ListWindows N=%d returned %d windows", n, len(resp.Windows))
			}
			listQueryCounts[n] = qc.Count()

			t.Logf("Scale N=%d: ListWindowCards queries=%d, ListWindows queries=%d",
				n, cardQueryCounts[n], listQueryCounts[n])
		}

		// 核心断言：查询数量在 N=1, 10, 50 下完全恒定，不得随 N 增长
		if cardQueryCounts[1] != cardQueryCounts[10] || cardQueryCounts[10] != cardQueryCounts[50] {
			t.Errorf("ListWindowCards query count scaled with N! Counts: N=1:%d, N=10:%d, N=50:%d",
				cardQueryCounts[1], cardQueryCounts[10], cardQueryCounts[50])
		}
		// 常数阶上限断言：ListWindowCards <= 8 (原先 N=10 时为 63 次, N=50 时为 303 次)
		if cardQueryCounts[50] > 8 {
			t.Errorf("ListWindowCards queries exceed constant bound 8: got %d", cardQueryCounts[50])
		}

		if listQueryCounts[1] != listQueryCounts[10] || listQueryCounts[10] != listQueryCounts[50] {
			t.Errorf("ListWindows query count scaled with N! Counts: N=1:%d, N=10:%d, N=50:%d",
				listQueryCounts[1], listQueryCounts[10], listQueryCounts[50])
		}
		// ListWindows 无敏捷小组查询，固定为 7 次
		if listQueryCounts[50] > 7 {
			t.Errorf("ListWindows queries exceed constant bound 7: got %d", listQueryCounts[50])
		}
	})

	t.Run("WindowStatsEquivalent", func(t *testing.T) {
		// 验证业务口径严格等价：含需求数、已消耗工时、负剩余工时、四舍五入、软删除排除
		if err := seedTestDatabase(ctx, db, 10, 10, 0); err != nil {
			t.Fatalf("failed to reset database: %v", err)
		}

		baseTime, _ := time.Parse(time.RFC3339, "2026-10-01T00:00:00Z")
		startDate := baseTime
		releaseDate := baseTime.AddDate(0, 0, 14) // 14天跨度

		// 插入 3 个窗口
		// 窗口 1: 正常需求与工时
		w1 := map[string]interface{}{
			"id":          uint64(101),
			"name":        "窗口-101",
			"releaseDate": releaseDate.Format("2006-01-02"),
			"startDate":   startDate.Format("2006-01-02"),
			"teamgroup":   1,
			"groupSize":   5,
			"createdBy":   "user_a",
			"status":      "planning",
			"order":       1,
		}
		// 窗口 2: 消耗工时超容量（负剩余工时）
		w2 := map[string]interface{}{
			"id":          uint64(102),
			"name":        "窗口-102",
			"releaseDate": releaseDate.Format("2006-01-02"),
			"startDate":   startDate.Format("2006-01-02"),
			"teamgroup":   1,
			"groupSize":   1,
			"createdBy":   "user_b",
			"status":      "planning",
			"order":       2,
		}
		// 窗口 3: 无关联需求/任务
		w3 := map[string]interface{}{
			"id":          uint64(103),
			"name":        "窗口-103",
			"releaseDate": releaseDate.Format("2006-01-02"),
			"startDate":   startDate.Format("2006-01-02"),
			"teamgroup":   0,
			"groupSize":   2,
			"createdBy":   "user_a",
			"status":      "planning",
			"order":       3,
		}
		for _, w := range []map[string]interface{}{w1, w2, w3} {
			if err := db.WithContext(ctx).Table("zt_versionwindow").Create(&w).Error; err != nil {
				t.Fatalf("failed to insert test window: %v", err)
			}
		}

		// 关联计划: 窗口 101 -> 计划 201; 窗口 102 -> 计划 202
		vwp1 := map[string]interface{}{"versionWindow": 101, "product": 1, "plan": 201}
		vwp2 := map[string]interface{}{"versionWindow": 102, "product": 1, "plan": 202}
		// 软删除关联（应被过滤）
		vwpDeleted := map[string]interface{}{"versionWindow": 103, "product": 1, "plan": 203, "deletedAt": time.Now()}
		for _, v := range []map[string]interface{}{vwp1, vwp2, vwpDeleted} {
			_ = db.WithContext(ctx).Table("zt_versionwindowproduct").Create(&v).Error
		}

		// 计划关联需求:
		// 计划 201: story 301 (demandpool, fromDemand=401), story 302 (demandpool, fromDemand=401 - 同一业需去重), story 303 (非demandpool独立软需)
		// 总需求数应为 2 (1个业需 + 1个独立软需)
		ps1 := map[string]interface{}{"plan": 201, "story": 301}
		ps2 := map[string]interface{}{"plan": 201, "story": 302}
		ps3 := map[string]interface{}{"plan": 201, "story": 303}
		// 计划 202: story 304
		ps4 := map[string]interface{}{"plan": 202, "story": 304}
		for _, ps := range []map[string]interface{}{ps1, ps2, ps3, ps4} {
			_ = db.WithContext(ctx).Table("zt_planstory").Create(&ps).Error
		}

		stories := []map[string]interface{}{
			{"id": 301, "sourceType": "demandpool", "fromDemand": 401, "deleted": "0", "stage": "developing"},
			{"id": 302, "sourceType": "demandpool", "fromDemand": 401, "deleted": "0", "stage": "developing"},
			{"id": 303, "sourceType": "feature", "fromDemand": 0, "deleted": "0", "stage": "developing"},
			{"id": 304, "sourceType": "feature", "fromDemand": 0, "deleted": "0", "stage": "testing"},
		}
		for _, s := range stories {
			_ = db.WithContext(ctx).Table("zt_story").Create(&s).Error
		}

		// 任务工时:
		// story 301: task 501 consumed=25.4 (应四舍五入为 25)
		// story 302: task 502 consumed=10.2 (总工时 35.6 -> 36)
		// story 304: task 503 consumed=500.0 (远超容量，验证负剩余工时)
		// soft-deleted task: task 504 consumed=99.0 (deleted='1' 应被忽略)
		tasks := []map[string]interface{}{
			{"id": 501, "name": "任务1", "story": 301, "consumed": 25.4, "deleted": "0", "status": "doing"},
			{"id": 502, "name": "任务2", "story": 302, "consumed": 10.2, "deleted": "0", "status": "doing"},
			{"id": 503, "name": "任务3", "story": 304, "consumed": 500.0, "deleted": "0", "status": "doing"},
			{"id": 504, "name": "已删任务", "story": 301, "consumed": 99.0, "deleted": "1", "status": "doing"},
		}
		for _, t := range tasks {
			_ = db.WithContext(ctx).Table("zt_task").Create(&t).Error
		}

		repo := schedule.NewRepo(db)
		svc := schedule.NewService(repo, nil)

		cards, err := svc.ListWindowCards(ctx, actor)
		if err != nil {
			t.Fatalf("ListWindowCards error: %v", err)
		}
		if len(cards) != 3 {
			t.Fatalf("expected 3 cards, got %d", len(cards))
		}

		cardMap := make(map[uint64]schedule.WindowCard)
		for _, c := range cards {
			cardMap[c.ID] = c
		}

		// 验证窗口 101 业务口径
		c101 := cardMap[101]
		if c101.DemandCount != 2 {
			t.Errorf("window 101 DemandCount: got %d, want 2 (1 distinct demandpool + 1 non-demandpool)", c101.DemandCount)
		}
		// consumed = 25.4 + 10.2 = 35.6 -> round = 36
		if c101.UsedHours != 36 {
			t.Errorf("window 101 UsedHours: got %d, want 36 (round of 35.6)", c101.UsedHours)
		}
		if c101.RemainingHours != c101.CapacityHours-36 {
			t.Errorf("window 101 RemainingHours: got %d, want %d", c101.RemainingHours, c101.CapacityHours-36)
		}
		if !c101.CanEdit || c101.CanDelete || !c101.HasLinkedDemands {
			t.Errorf("window 101 permissions: got CanEdit=%v, CanDelete=%v, HasLinkedDemands=%v",
				c101.CanEdit, c101.CanDelete, c101.HasLinkedDemands)
		}

		// 验证窗口 102 业务口径 (负剩余工时、非创建者)
		c102 := cardMap[102]
		if c102.UsedHours != 500 {
			t.Errorf("window 102 UsedHours: got %d, want 500", c102.UsedHours)
		}
		if c102.RemainingHours >= 0 {
			t.Errorf("window 102 RemainingHours should be negative: got %d", c102.RemainingHours)
		}
		if c102.CanEdit || c102.CanDelete {
			t.Errorf("window 102 should not be editable by user_a: CanEdit=%v, CanDelete=%v",
				c102.CanEdit, c102.CanDelete)
		}

		// 验证窗口 103 (空需求、软删除过滤)
		c103 := cardMap[103]
		if c103.DemandCount != 0 {
			t.Errorf("window 103 DemandCount should be 0 (deleted link excluded): got %d", c103.DemandCount)
		}
		if c103.UsedHours != 0 {
			t.Errorf("window 103 UsedHours should be 0: got %d", c103.UsedHours)
		}
		if !c103.CanEdit || !c103.CanDelete || c103.HasLinkedDemands {
			t.Errorf("window 103 permissions: got CanEdit=%v, CanDelete=%v, HasLinkedDemands=%v",
				c103.CanEdit, c103.CanDelete, c103.HasLinkedDemands)
		}
	})
}
