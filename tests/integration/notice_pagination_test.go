//go:build integration

// =============================================================================
// 文件: tests/integration/notice_pagination_test.go
// 模块: 性能与分页集成测试 (P1)
// 类型: test
// 职责: 验证通知中心 SQL 过滤、分类计数、有界分页与老实现结果严格等价。
// 依赖: tests/integration/db_schema.go, tests/integration/db_seed.go, tests/integration/query_counter.go
// =============================================================================

package integration

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"workbench/internal/module/po"
)

func TestNoticePagination(t *testing.T) {
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
	repo := po.NewRepo(dbWithCounter)

	t.Run("NoticeFilterEquivalence", func(t *testing.T) {
		qc.Reset()
		resp, err := repo.FindNotices(ctx, "user_a", po.NoticeListReq{Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("FindNotices error: %v", err)
		}
		queriesUsed := qc.Count()
		t.Logf("NoticeFilterEquivalence: total=%d, filtered=%d, items=%d, queries=%d",
			resp.Total, resp.Filtered, len(resp.Items), queriesUsed)

		// 验证 SQL 查询次数有界：未筛选时只需 1 次聚合 count + 1 次分页 select = 2 次
		if queriesUsed > 3 {
			t.Errorf("expected <= 3 queries, got %d", queriesUsed)
		}

		// 读取老实现基准镜像比对
		fixtureBytes, err := os.ReadFile("testdata/performance/old_impl_results.json")
		if err != nil {
			t.Fatalf("failed to read fixture: %v", err)
		}
		var fixture struct {
			Notices struct {
				Total             int64            `json:"total"`
				Filtered          int64            `json:"filtered"`
				Unread            int64            `json:"unread"`
				Categories        map[string]int64 `json:"categories"`
				ReturnedItemCount int              `json:"returned_item_count"`
			} `json:"notices"`
		}
		if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
			t.Fatalf("failed to parse fixture json: %v", err)
		}

		if resp.Total != fixture.Notices.Total {
			t.Errorf("total mismatch: got %d, want %d", resp.Total, fixture.Notices.Total)
		}
		if resp.Filtered != fixture.Notices.Filtered {
			t.Errorf("filtered mismatch: got %d, want %d", resp.Filtered, fixture.Notices.Filtered)
		}
		if resp.Unread != fixture.Notices.Unread {
			t.Errorf("unread mismatch: got %d, want %d", resp.Unread, fixture.Notices.Unread)
		}
		if len(resp.Items) != fixture.Notices.ReturnedItemCount {
			t.Errorf("item count mismatch: got %d, want %d", len(resp.Items), fixture.Notices.ReturnedItemCount)
		}
		for cat, wantCount := range fixture.Notices.Categories {
			if gotCount := resp.Categories[cat]; gotCount != wantCount {
				t.Errorf("category %q count mismatch: got %d, want %d", cat, gotCount, wantCount)
			}
		}
	})

	t.Run("NoticeBoundedPage", func(t *testing.T) {
		// Page 1: 20 items
		qc.Reset()
		p1, err := repo.FindNotices(ctx, "user_a", po.NoticeListReq{Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("page 1 error: %v", err)
		}
		if len(p1.Items) != 20 {
			t.Errorf("page 1 item count: got %d, want 20", len(p1.Items))
		}

		// Page 2: 8 items (total 28)
		qc.Reset()
		p2, err := repo.FindNotices(ctx, "user_a", po.NoticeListReq{Page: 2, PageSize: 20})
		if err != nil {
			t.Fatalf("page 2 error: %v", err)
		}
		if len(p2.Items) != 8 {
			t.Errorf("page 2 item count: got %d, want 8", len(p2.Items))
		}

		// Page 3: 0 items (out of bounds)
		qc.Reset()
		p3, err := repo.FindNotices(ctx, "user_a", po.NoticeListReq{Page: 3, PageSize: 20})
		if err != nil {
			t.Fatalf("page 3 error: %v", err)
		}
		if len(p3.Items) != 0 {
			t.Errorf("page 3 item count: got %d, want 0", len(p3.Items))
		}
	})

	t.Run("NoticeScope", func(t *testing.T) {
		// user_c should only see notices containing user_c
		resp, err := repo.FindNotices(ctx, "user_c", po.NoticeListReq{Page: 1, PageSize: 50})
		if err != nil {
			t.Fatalf("FindNotices user_c error: %v", err)
		}
		t.Logf("user_c notices: total=%d, items=%d", resp.Total, len(resp.Items))
		if resp.Total > 0 && len(resp.Items) == 0 {
			t.Errorf("expected items for user_c")
		}
	})

	t.Run("KeywordLiteral/TimeBoundary", func(t *testing.T) {
		// Literal % search: 应该精确匹配包含字面值 '%' 的记录，不可作为通配符匹配所有 28 条
		respWildcard, err := repo.FindNotices(ctx, "user_a", po.NoticeListReq{Keyword: "%", Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("FindNotices literal percent error: %v", err)
		}
		// 种子数据中只有一条包含 "100%"
		if respWildcard.Filtered != 1 {
			t.Errorf("expected 1 match for literal '%%', got %d", respWildcard.Filtered)
		}

		// 搜索不存在的字面值 "999%"
		respNone, err := repo.FindNotices(ctx, "user_a", po.NoticeListReq{Keyword: "999%", Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("FindNotices literal percent none error: %v", err)
		}
		if respNone.Filtered != 0 {
			t.Errorf("expected 0 matches for literal '999%%', got %d", respNone.Filtered)
		}

		// Literal _ search: 匹配 "order_v2" (在种子数据中发给了 user_b)
		respUnderscore, err := repo.FindNotices(ctx, "user_b", po.NoticeListReq{Keyword: "order_v2", Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("FindNotices literal underscore error: %v", err)
		}
		if respUnderscore.Filtered == 0 {
			t.Errorf("expected >= 1 match for literal 'order_v2' for user_b")
		}

		// Time boundary: 30d should cover recently generated notices
		resp30d, err := repo.FindNotices(ctx, "user_a", po.NoticeListReq{TimeRange: "30d", Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("FindNotices 30d error: %v", err)
		}
		if resp30d.Filtered == 0 {
			t.Errorf("expected > 0 matches for 30d range")
		}

		// ObjectType approval filter
		respApproval, err := repo.FindNotices(ctx, "user_a", po.NoticeListReq{ObjectType: "approval", Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("FindNotices approval filter error: %v", err)
		}
		if respApproval.Filtered != 2 {
			t.Errorf("expected 2 approval notices, got %d", respApproval.Filtered)
		}
	})

	_ = time.Now()
}
