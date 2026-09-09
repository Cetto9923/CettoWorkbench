//go:build integration

package po

import (
	"context"
	"os"
	"reflect"
	"sort"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Explicit opt-in, SELECT-only parity checks against an existing local dataset.
// No migrations, fixtures, transactions with writes, or business actions.
func TestHomePerformanceReadOnlyParity(t *testing.T) {
	dsn, account := os.Getenv("PO_PERF_DSN"), os.Getenv("PO_PERF_ACCOUNT")
	if dsn == "" || account == "" {
		t.Skip("set PO_PERF_DSN and PO_PERF_ACCOUNT for read-only parity")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("cannot open parity database")
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	r := NewRepo(db, db)
	ctx := context.Background()
	var expected []itemRef
	seen := map[itemRef]bool{}
	for _, stage := range valueStreamStages {
		filter, ok := mysqlStageFilters[stage.status]
		if !ok {
			continue
		}
		ids, err := r.FindRoleDemandIDs(ctx, account, filter)
		if err != nil {
			t.Fatal(err)
		}
		appendIDs := func(kind string, ids []int) {
			for _, id := range ids {
				key := itemRef{kind: kind, id: id}
				if !seen[key] {
					seen[key] = true
					expected = append(expected, itemRef{kind: kind, id: id, stageStatus: stage.status})
				}
			}
		}
		appendIDs("demand", ids)
		if filter.scheduleIncomplete {
			ids, err = r.FindScheduleStoryIDs(ctx, account)
		} else if filter.deliverStories {
			ids, err = r.FindDeliverStoryIDs(ctx, account)
		} else {
			ids = nil
		}
		if err != nil {
			t.Fatal(err)
		}
		appendIDs("story", ids)
	}
	if len(expected) == 0 {
		t.Fatal("parity requires nonempty data")
	}
	for _, page := range []int{1, 2, (len(expected) + 14) / 15, (len(expected)+14)/15 + 1} {
		refs, total, err := r.FindAllStageRefsPaged(ctx, account, DemandsReq{Page: page, PageSize: 15})
		if err != nil {
			t.Fatal(err)
		}
		start := min((page-1)*15, len(expected))
		end := min(start+15, len(expected))
		if total != len(expected) || len(refs) != end-start {
			t.Fatalf("page %d count mismatch", page)
		}
		for i := range refs {
			if refs[i] != expected[start+i] {
				t.Fatalf("page %d order or membership mismatch", page)
			}
		}
	}
	counts, err := r.countStageRefs(ctx, account)
	if err != nil {
		t.Fatal(err)
	}
	wantCounts := map[string]int64{}
	for _, ref := range expected {
		wantCounts[ref.kind+":"+ref.stageStatus]++
	}
	gotCounts := map[string]int64{}
	for _, row := range counts {
		gotCounts[row.Kind+":"+valueStreamStages[row.StageIndex].status] = row.Count
	}
	if !reflect.DeepEqual(wantCounts, gotCounts) {
		t.Fatal("stage aggregate differs from legacy partition")
	}
	ids, err := r.FindRoleDemandIDs(ctx, account, mysqlStageFilters["accept"])
	if err != nil {
		t.Fatal(err)
	}
	pending, err := r.FindPendingReviewDemandIDs(ctx, account, ids)
	if err != nil {
		t.Fatal(err)
	}
	var pinned, rest []int
	for _, id := range ids {
		if _, ok := pending[id]; ok {
			pinned = append(pinned, id)
		} else {
			rest = append(rest, id)
		}
	}
	ordered := append(pinned, rest...)
	for _, page := range []int{1, 2} {
		refs, total, err := r.acceptRefsPaged(ctx, account, DemandsReq{Page: page, PageSize: 15})
		if err != nil {
			t.Fatal(err)
		}
		start := min((page-1)*15, len(ordered))
		end := min(start+15, len(ordered))
		if total != len(ordered) || len(refs) != end-start {
			t.Fatal("accept count mismatch")
		}
		for i := range refs {
			if refs[i].id != ordered[start+i] {
				t.Fatal("accept pin order mismatch")
			}
		}
	}
	t.Logf("read-only parity: %d all-stage objects; %d accept objects; %d pending reviews", len(expected), len(ids), len(pinned))
	for _, focus := range []string{"today", "my_action"} {
		t.Run(focus, func(t *testing.T) { checkFocusParity(t, r, account, focus) })
	}
}

func checkFocusParity(t *testing.T, r *Repo, account, focus string) {
	ctx := context.Background()
	reviews, err := r.FindAccountPendingReviewDemandIDs(ctx, account)
	if err != nil {
		t.Fatal(err)
	}
	pending := map[int]bool{}
	for _, id := range reviews {
		pending[id] = true
	}
	var expected []itemRef
	seen := map[int]bool{}
	for _, stage := range valueStreamStages {
		if stage.status == "all" {
			continue
		}
		var rows []struct {
			ID         int
			Status     string
			AssignedTo string
			CreatedBy  string
		}
		q := r.homeFocusQueryWithReviews(ctx, account, DemandsReq{Status: stage.status, Focus: focus}, reviews)
		if err := r.db.Table("(?) AS legacy_stage", q).Select("id, status, assignedTo, createdBy").Order("id DESC").Scan(&rows).Error; err != nil {
			t.Fatal(err)
		}
		for _, row := range rows {
			if seen[row.ID] {
				continue
			}
			seen[row.ID] = true
			rank := 2
			if row.Status == "wait" && pending[row.ID] {
				rank = 0
			} else if (row.Status == "draft" || row.Status == "refuse") && (row.AssignedTo == account || row.CreatedBy == account) {
				rank = 1
			}
			expected = append(expected, itemRef{kind: "demand", id: row.ID, stageStatus: stage.status, actionRank: rank})
		}
	}
	var stories []struct {
		ID         int
		StageIndex int
	}
	req := DemandsReq{Status: "all", Focus: focus, Page: 1, PageSize: 15}
	if err := r.homeFocusStoryQuery(ctx, account, req).Select("id, stage_index").Order("id DESC").Scan(&stories).Error; err != nil {
		t.Fatal(err)
	}
	for _, story := range stories {
		stage := ""
		if story.StageIndex > 0 && story.StageIndex < len(valueStreamStages) {
			stage = valueStreamStages[story.StageIndex].status
		}
		expected = append(expected, itemRef{kind: "story", id: story.ID, stageStatus: stage, actionRank: 2})
	}
	sort.SliceStable(expected, func(i, j int) bool {
		a, b := expected[i], expected[j]
		if a.actionRank != b.actionRank {
			return a.actionRank < b.actionRank
		}
		if a.stageStatus != b.stageStatus {
			return homeFocusStageIndex(a.stageStatus) < homeFocusStageIndex(b.stageStatus)
		}
		return a.id > b.id
	})
	for _, page := range []int{1, 2, (len(expected)+14)/15 + 1} {
		req.Page = page
		refs, total, err := r.FindHomeFocus(ctx, account, req)
		if err != nil {
			t.Fatal(err)
		}
		start := min((page-1)*15, len(expected))
		end := min(start+15, len(expected))
		if total != len(expected) || len(refs) != end-start {
			t.Fatal("focus count mismatch")
		}
		for i := range refs {
			if refs[i] != expected[start+i] {
				t.Fatal("focus page order or membership mismatch")
			}
		}
	}
	summary, err := r.HomeFocusStageSummary(ctx, account, req)
	if err != nil {
		t.Fatal(err)
	}
	for _, stage := range summary[1:] {
		var count int64
		for _, ref := range expected {
			if ref.stageStatus == stage.Status {
				count++
			}
		}
		if stage.Count != count {
			t.Fatalf("focus summary mismatch for %s", stage.Status)
		}
	}
	t.Logf("focus parity: %d objects", len(expected))
}
