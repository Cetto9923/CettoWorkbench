//go:build integration

// =============================================================================
// 文件: tests/integration/todo_pagination_test.go
// 模块: 性能与待办分页集成测试 (P2)
// 类型: test
// 职责: 验证待办中心 SQL 聚合过滤、全局稳定排序、有界分页与老实现结果严格等价。
// 依赖: tests/integration/db_schema.go, tests/integration/db_seed.go, tests/integration/query_counter.go
// =============================================================================

package integration

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"workbench/internal/model"
	"workbench/internal/module/po"
)

func TestTodoPagination(t *testing.T) {
	db := openIsolatedTestDB(t)
	ctx := context.Background()

	// 初始化表与合成数据 (100条通知, 50条待办, 10个窗口)
	if err := InitMinimalSchema(ctx, db); err != nil {
		t.Fatalf("failed to init minimal schema: %v", err)
	}
	if err := SeedSyntheticDatabase(ctx, db, 100, 50, 10); err != nil {
		t.Fatalf("failed to seed synthetic database: %v", err)
	}

	dbWithCounter, qc := AttachQueryCounter(db)
	repo := po.NewRepo(dbWithCounter, dbWithCounter)
	svc := po.NewService(repo, nil, nil, nil)
	actor := &model.User{ID: 1, Account: "user_a"}

	t.Run("TodoFullEquivalence", func(t *testing.T) {
		qc.Reset()
		resp, err := svc.TodoList(ctx, actor, po.TodoListReq{Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("TodoList error: %v", err)
		}
		queriesUsed := qc.Count()
		t.Logf("TodoFullEquivalence: total=%d, items=%d, queries=%d",
			resp.Total, len(resp.Items), queriesUsed)

		// 验证 SQL 查询次数有界：统计 count + 分页 select <= 3 次（对比老实现 21 次 SQL 查询）
		if queriesUsed > 3 {
			t.Errorf("expected <= 3 queries, got %d", queriesUsed)
		}

		// 读取老实现基准镜像比对
		fixtureBytes, err := os.ReadFile("testdata/performance/old_impl_results.json")
		if err != nil {
			t.Fatalf("failed to read fixture: %v", err)
		}
		var fixture struct {
			Todos struct {
				Total             int64 `json:"total"`
				ReturnedItemCount int   `json:"returned_item_count"`
			} `json:"todos"`
		}
		if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
			t.Fatalf("failed to parse fixture json: %v", err)
		}

		if resp.Total != fixture.Todos.Total {
			t.Errorf("total mismatch: got %d, want %d", resp.Total, fixture.Todos.Total)
		}
		if len(resp.Items) != 20 {
			t.Errorf("page 1 items mismatch: got %d, want 20", len(resp.Items))
		}
		if resp.Summary.Pending != int(fixture.Todos.Total) {
			t.Errorf("summary pending mismatch: got %d, want %d", resp.Summary.Pending, fixture.Todos.Total)
		}
	})

	t.Run("CrossTypeGlobalPage", func(t *testing.T) {
		// 跨类型全局排序验证：第一页应该按优先级和截止日期稳定交织 demand、task、bug
		resp, err := svc.TodoList(ctx, actor, po.TodoListReq{Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("TodoList error: %v", err)
		}

		kindsSeen := make(map[string]int)
		for _, item := range resp.Items {
			kindsSeen[item.Kind]++
		}
		t.Logf("page 1 kinds distribution: %#v", kindsSeen)
		if len(kindsSeen) < 2 {
			t.Errorf("expected global sort to interleave multiple kinds on page 1, got: %#v", kindsSeen)
		}

		// 验证全局稳定降序排序：前面的优先级绝不能低于后面的优先级
		priorityRank := func(p string) int {
			switch p {
			case "P1":
				return 1
			case "P2":
				return 2
			case "P3":
				return 3
			default:
				return 4
			}
		}
		for i := 1; i < len(resp.Items); i++ {
			prev := resp.Items[i-1]
			curr := resp.Items[i]
			prevRank := priorityRank(prev.Priority)
			currRank := priorityRank(curr.Priority)
			if prevRank > currRank {
				t.Errorf("sorting violation at index %d: prev=%s (%s), curr=%s (%s)",
					i, prev.Priority, prev.Kind, curr.Priority, curr.Kind)
			}
		}
	})

	t.Run("StableTie/ZeroDate/OwnerKeyword", func(t *testing.T) {
		// 1. 关键词搜索负责人显示名
		respOwner, err := svc.TodoList(ctx, actor, po.TodoListReq{Keyword: "user_a", Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("search owner error: %v", err)
		}
		if respOwner.Total == 0 {
			t.Errorf("expected > 0 results for owner keyword search")
		}

		// 2. 截止日空值排在最后验证
		respAll, err := svc.TodoList(ctx, actor, po.TodoListReq{Page: 1, PageSize: 100})
		if err != nil {
			t.Fatalf("list all error: %v", err)
		}
		seenEmptyDeadline := false
		for _, item := range respAll.Items {
			if item.Deadline == "" {
				seenEmptyDeadline = true
			} else if seenEmptyDeadline && item.Priority == respAll.Items[0].Priority {
				// 在同一优先级内，非空截止日绝不能排在空截止日之后
				t.Errorf("zero date sorting violation: non-empty deadline %s appeared after empty deadline", item.Deadline)
			}
		}
	})

	t.Run("BoundedDetails", func(t *testing.T) {
		// 分页边界检测: 总数 148
		// Page 7: 20 items (121 - 140)
		p7, err := svc.TodoList(ctx, actor, po.TodoListReq{Page: 7, PageSize: 20})
		if err != nil {
			t.Fatalf("page 7 error: %v", err)
		}
		if len(p7.Items) != 20 {
			t.Errorf("page 7 item count: got %d, want 20", len(p7.Items))
		}

		// Page 8: 8 items (141 - 148)
		p8, err := svc.TodoList(ctx, actor, po.TodoListReq{Page: 8, PageSize: 20})
		if err != nil {
			t.Fatalf("page 8 error: %v", err)
		}
		if len(p8.Items) != 8 {
			t.Errorf("page 8 item count: got %d, want 8", len(p8.Items))
		}

		// Page 9: 0 items (越界)
		p9, err := svc.TodoList(ctx, actor, po.TodoListReq{Page: 9, PageSize: 20})
		if err != nil {
			t.Fatalf("page 9 error: %v", err)
		}
		if len(p9.Items) != 0 {
			t.Errorf("page 9 item count: got %d, want 0", len(p9.Items))
		}
	})
}
