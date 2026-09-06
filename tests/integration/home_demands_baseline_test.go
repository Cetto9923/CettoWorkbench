//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"workbench/internal/model"
	"workbench/internal/module/po"
	"workbench/internal/module/schedule"
	"workbench/internal/module/user"
)

func TestHomeAndDemandsBaseline(t *testing.T) {
	db := openIsolatedTestDB(t)
	ctx := context.Background()

	if err := InitMinimalSchema(ctx, db); err != nil {
		t.Fatalf("failed to init minimal schema: %v", err)
	}

	const nNotices = 100
	const nTodos = 100
	const nWindows = 10
	if err := SeedSyntheticDatabase(ctx, db, nNotices, nTodos, nWindows); err != nil {
		t.Fatalf("failed to seed database: %v", err)
	}

	dbWithCounter, qc := AttachQueryCounter(db)

	poRepo := po.NewRepo(dbWithCounter, dbWithCounter)
	scheduleRepo := schedule.NewRepo(dbWithCounter)
	scheduleSvc := schedule.NewService(scheduleRepo, nil)
	userRepo := user.NewRepo(dbWithCounter)
	userSvc := user.NewService(userRepo)
	poSvc := po.NewService(poRepo, scheduleSvc, userSvc, nil)

	actor := &model.User{ID: 1, Account: "user_a"}

	// --- 1. /home SSR 基线测量 ---
	qc.Reset()
	startHome := time.Now()
	homeResp, err := poSvc.Home(ctx, actor)
	homeDuration := time.Since(startHome)
	homeQueries := qc.Count()
	if err != nil {
		t.Fatalf("Home error: %v", err)
	}
	t.Logf("=== /home SSR Baseline ===")
	t.Logf("Duration: %v", homeDuration)
	t.Logf("SQL Queries: %d", homeQueries)
	t.Logf("Stages Valid: %v, Count of stages: %d", homeResp.StagesValid, len(homeResp.Stages))
	totalDemandStory := int64(0)
	for _, st := range homeResp.Stages {
		if st.Status == "all" {
			totalDemandStory = st.Count
		}
	}
	t.Logf("Home all stage total: %d", totalDemandStory)

	// --- 2. /demands?status=all 基线测量 ---
	qc.Reset()
	startDemands := time.Now()
	demandsResp, err := poSvc.Demands(ctx, actor, po.DemandsReq{Status: "all"})
	demandsDuration := time.Since(startDemands)
	demandsQueries := qc.Count()
	if err != nil {
		t.Fatalf("Demands(all) error: %v", err)
	}
	payloadBytes, _ := json.Marshal(demandsResp)
	t.Logf("=== /demands?status=all Baseline ===")
	t.Logf("Duration: %v", demandsDuration)
	t.Logf("SQL Queries: %d", demandsQueries)
	t.Logf("Total items: %d, Page items: %d", demandsResp.Total, len(demandsResp.Items))
	t.Logf("JSON Payload size: %d bytes (%.2f KB)", len(payloadBytes), float64(len(payloadBytes))/1024.0)

	if demandsResp.Total != 95 {
		t.Fatalf("demandsResp.Total = %d, want 95", demandsResp.Total)
	}
	if len(demandsResp.Items) != 15 {
		t.Fatalf("demandsResp.Items len = %d, want 15", len(demandsResp.Items))
	}
	if demandsResp.Page != 1 || demandsResp.PageSize != 15 {
		t.Fatalf("demandsResp page=%d pageSize=%d, want 1, 15", demandsResp.Page, demandsResp.PageSize)
	}

	// Page 2 verification
	page2, err := poSvc.Demands(ctx, actor, po.DemandsReq{Status: "all", Page: 2, PageSize: 15})
	if err != nil {
		t.Fatalf("Demands(all, page 2) error: %v", err)
	}
	if len(page2.Items) != 15 {
		t.Fatalf("page 2 items len = %d, want 15", len(page2.Items))
	}
	if page2.Items[0].ID == demandsResp.Items[0].ID {
		t.Fatalf("page 1 and page 2 first items should not match: %s", page2.Items[0].ID)
	}

	// Page 7 (last page with remaining 5 items)
	page7, err := poSvc.Demands(ctx, actor, po.DemandsReq{Status: "all", Page: 7, PageSize: 15})
	if err != nil {
		t.Fatalf("Demands(all, page 7) error: %v", err)
	}
	if len(page7.Items) != 5 {
		t.Fatalf("page 7 items len = %d, want 5", len(page7.Items))
	}

	// Page 8 (beyond total)
	page8, err := poSvc.Demands(ctx, actor, po.DemandsReq{Status: "all", Page: 8, PageSize: 15})
	if err != nil {
		t.Fatalf("Demands(all, page 8) error: %v", err)
	}
	if len(page8.Items) != 0 {
		t.Fatalf("page 8 items len = %d, want 0", len(page8.Items))
	}

	// --- 3. /demands?status=accept 单阶段基线测量 ---
	qc.Reset()
	startAccept := time.Now()
	acceptResp, err := poSvc.Demands(ctx, actor, po.DemandsReq{Status: "accept"})
	acceptDuration := time.Since(startAccept)
	acceptQueries := qc.Count()
	if err != nil {
		t.Fatalf("Demands(accept) error: %v", err)
	}
	acceptPayload, _ := json.Marshal(acceptResp)
	t.Logf("=== /demands?status=accept Baseline ===")
	t.Logf("Duration: %v", acceptDuration)
	t.Logf("SQL Queries: %d", acceptQueries)
	t.Logf("Items returned: %d", len(acceptResp.Items))
	t.Logf("JSON Payload size: %d bytes", len(acceptPayload))

	// --- 4. 规模 N=500 时的全量无分页开销 ---
	if err := SeedSyntheticDatabase(ctx, db, 100, 500, 10); err == nil {
		qc.Reset()
		start500 := time.Now()
		demandsResp500, err500 := poSvc.Demands(ctx, actor, po.DemandsReq{Status: "all"})
		duration500 := time.Since(start500)
		queries500 := qc.Count()
		if err500 == nil {
			payload500, _ := json.Marshal(demandsResp500)
			t.Logf("=== /demands?status=all at N=500 Baseline (Unbounded) ===")
			t.Logf("Duration: %v", duration500)
			t.Logf("SQL Queries: %d", queries500)
			t.Logf("Items returned: %d", len(demandsResp500.Items))
			t.Logf("JSON Payload size: %d bytes (%.2f KB)", len(payload500), float64(len(payload500))/1024.0)
		}
	}
}
