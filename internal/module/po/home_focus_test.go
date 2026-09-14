package po

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestHomeFocusSQLIntersectsStageBeforePagination(t *testing.T) {
	db, _ := openSQLMock(t)
	repo := NewRepo(db, nil)
	for _, focus := range []string{"today", "overdue", "blocked", "suspended"} {
		req := DemandsReq{Status: "testing", Focus: focus, Page: 2, PageSize: 15}
		query := repo.homeFocusQuery(context.Background(), "alice", req)
		var rows []struct{ ID int }
		stmt := db.Table("(?) AS focused", query).Select("id, stage_index").Order("stage_index ASC, id DESC").Offset(15).Limit(15).Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
		sql := stmt.SQL.String()
		for _, want := range []string{"GROUP BY `id`", "status IN", "zt_demandreview", "zt_demandmanagerreview", "accepter = ?", "LIMIT ? OFFSET ?"} {
			if !strings.Contains(sql, want) {
				t.Fatalf("%s missing %q: %s", focus, want, sql)
			}
		}
		if strings.Contains(sql, "UNION ALL") {
			t.Fatal("selected stage must not include other stages")
		}
		switch focus {
		case "today":
			if !strings.Contains(sql, "deadline <= ?") {
				t.Fatal("today must include overdue")
			}
		case "overdue":
			if !strings.Contains(sql, "deadline < ?") {
				t.Fatal("overdue must exclude today")
			}
		case "blocked":
			if strings.Contains(sql, "testFinish <") || strings.Contains(sql, "resultStatus") {
				t.Fatal("historical and time facts must not imply blocking")
			}
		case "suspended":
			if !strings.Contains(sql, "hang = ?") {
				t.Fatal("missing current suspension fact")
			}
		}
		if stmt.Vars[len(stmt.Vars)-1] != 15 {
			t.Fatal("missing SQL offset")
		}
	}
}

func TestHomeFocusRequestValidation(t *testing.T) {
	for _, focus := range []string{"all", "today", "blocked", "overdue", "suspended"} {
		req := DemandsReq{Status: "all", Focus: focus}
		if errs := req.Validate(); len(errs) > 0 {
			t.Fatalf("%s: %v", focus, errs)
		}
	}
	req := DemandsReq{Status: "all", Focus: "unknown"}
	if len(req.Validate()) == 0 {
		t.Fatal("unknown focus must not silently show all")
	}
	for _, relation := range []string{"all", "lead", "participate", "handling", "following"} {
		req := DemandsReq{Status: "all", Relation: relation}
		if errs := req.Validate(); len(errs) > 0 {
			t.Fatalf("relation %s: %v", relation, errs)
		}
	}
	if errs := (&DemandsReq{Status: "all", Relation: "owner"}).Validate(); len(errs) == 0 {
		t.Fatal("legacy relation must not silently acquire the new semantics")
	}
}

func TestHomeFocusToolbarRelationLeadAndParticipate(t *testing.T) {
	db, _ := openSQLMock(t)
	repo := NewRepo(db, nil)
	base := repo.homeFocusQuery(context.Background(), "alice", DemandsReq{Status: "developing", Page: 1, PageSize: 15})

	// lead: 我作为需求负责人 (BRA = account)
	queryLead := applyHomeFocusToolbarFilters(base, "alice", DemandsReq{Relation: "lead"})
	var rows []struct{ ID int }
	stmtLead := db.Table("(?) AS focused", queryLead).Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	if !strings.Contains(stmtLead.SQL.String(), "id IN (SELECT id FROM zt_demand WHERE BRA = ?)") {
		t.Fatalf("lead must filter by BRA = ?: %s", stmtLead.SQL.String())
	}

	// participate: 需求池业务需求 + 非当前负责人 + 当前用户是需求分析人
	queryPart := applyHomeFocusToolbarFilters(base, "alice", DemandsReq{Relation: "participate"})
	stmtPart := db.Table("(?) AS focused", queryPart).Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	if !strings.Contains(stmtPart.SQL.String(), "d.pool IS NOT NULL AND d.pool <> 0") ||
		!strings.Contains(stmtPart.SQL.String(), "zt_demandpool") ||
		!strings.Contains(stmtPart.SQL.String(), "FIND_IN_SET") ||
		!strings.Contains(stmtPart.SQL.String(), "d.BRA <> ? OR d.BRA IS NULL OR d.BRA = ''") {
		t.Fatalf("participate must require demand-pool source, non-lead and analyst: %s", stmtPart.SQL.String())
	}
}

func TestHomeFocusStoryParticipateExcludesStories(t *testing.T) {
	db, _ := openSQLMock(t)
	query := applyStoryToolbarFilters(db.Table("zt_story"), "alice", DemandsReq{Relation: "participate"})
	var rows []struct{ ID int }
	stmt := query.Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	sql := stmt.SQL.String()
	if !strings.Contains(sql, "1 = 0") {
		t.Fatalf("participate must not expose story rows: %s", sql)
	}
}

func TestHomeStoryStage_NoClarifyForIndependentStories(t *testing.T) {
	// 研需不走澄清；draft/wait/active（及排期主路径上的 planned 等）归排期。
	for _, status := range []string{"draft", "wait", "active", "clarified", "planned", "projected", "designed", "designing"} {
		if got := homeStoryStage(status); got != "schedule" {
			t.Fatalf("homeStoryStage(%q) = %q, want schedule", status, got)
		}
	}
	if got := homeStoryStage("developing"); got != "developing" {
		t.Fatalf("homeStoryStage(developing) = %q, want developing", got)
	}
	clarifyIdx := homeFocusStageIndex("clarify")
	scheduleIdx := homeFocusStageIndex("schedule")
	sql := homeFocusStoryStageSQL()
	for _, status := range []string{"draft", "wait", "active"} {
		wantClarify := fmt.Sprintf("WHEN '%s' THEN %d", status, clarifyIdx)
		wantSchedule := fmt.Sprintf("WHEN '%s' THEN %d", status, scheduleIdx)
		if strings.Contains(sql, wantClarify) {
			t.Fatalf("homeFocusStoryStageSQL must not map %s to clarify index %d: %s", status, clarifyIdx, sql)
		}
		if !strings.Contains(sql, wantSchedule) {
			t.Fatalf("homeFocusStoryStageSQL must map %s to schedule index %d: %s", status, scheduleIdx, sql)
		}
	}
}

func TestHomeFocusStoryQuery_ClarifyExcludesActiveStory_ScheduleIncludes(t *testing.T) {
	db, _ := openSQLMock(t)
	repo := NewRepo(db, nil)
	account := "alice"
	clarifyIdx := homeFocusStageIndex("clarify")
	scheduleIdx := homeFocusStageIndex("schedule")

	clarifyQ := repo.homeFocusStoryQuery(context.Background(), account, DemandsReq{
		Focus: "my_action", Status: "clarify", Page: 1, PageSize: 5,
	})
	var rows []struct{ ID int }
	clarifyStmt := clarifyQ.Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	clarifySQL := clarifyStmt.SQL.String()
	if !strings.Contains(clarifySQL, "stage_index = ?") {
		t.Fatalf("clarify focus story query must filter by stage_index: %s", clarifySQL)
	}
	foundClarifyIdx := false
	for _, v := range clarifyStmt.Vars {
		if idx, ok := v.(int); ok && idx == clarifyIdx {
			foundClarifyIdx = true
			break
		}
	}
	if !foundClarifyIdx {
		t.Fatalf("clarify query must bind stage_index=%d, vars=%v", clarifyIdx, clarifyStmt.Vars)
	}
	// CASE maps early story statuses to schedule, so clarify filter cannot list them.
	for _, status := range []string{"draft", "wait", "active"} {
		if strings.Contains(homeFocusStoryStageSQL(), fmt.Sprintf("WHEN '%s' THEN %d", status, clarifyIdx)) {
			t.Fatalf("active-path status %s must not land in clarify CASE branch", status)
		}
	}

	scheduleQ := repo.homeFocusStoryQuery(context.Background(), account, DemandsReq{
		Focus: "my_action", Status: "schedule", Page: 1, PageSize: 5,
	})
	scheduleStmt := scheduleQ.Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	scheduleSQL := scheduleStmt.SQL.String()
	if !strings.Contains(scheduleSQL, "stage_index = ?") {
		t.Fatalf("schedule focus story query must filter by stage_index: %s", scheduleSQL)
	}
	foundScheduleIdx := false
	for _, v := range scheduleStmt.Vars {
		if idx, ok := v.(int); ok && idx == scheduleIdx {
			foundScheduleIdx = true
			break
		}
	}
	if !foundScheduleIdx {
		t.Fatalf("schedule query must bind stage_index=%d, vars=%v", scheduleIdx, scheduleStmt.Vars)
	}
	for _, status := range []string{"draft", "wait", "active"} {
		want := fmt.Sprintf("WHEN '%s' THEN %d", status, scheduleIdx)
		if !strings.Contains(homeFocusStoryStageSQL(), want) {
			t.Fatalf("schedule CASE must include %s → %d", status, scheduleIdx)
		}
	}
}

func TestHomeFocusToolbarUsesCurrentHandler(t *testing.T) {
	db, _ := openSQLMock(t)
	repo := NewRepo(db, nil)
	base := repo.homeFocusQuery(context.Background(), "alice", DemandsReq{Status: "developing", Page: 1, PageSize: 15})
	for relation, want := range map[string]string{
		"handling":  "id IN (SELECT id FROM zt_demand WHERE",
		"following": "id NOT IN (SELECT id FROM zt_demand WHERE",
	} {
		query := applyHomeFocusToolbarFilters(base, "alice", DemandsReq{Relation: relation})
		var rows []struct{ ID int }
		stmt := db.Table("(?) AS focused", query).Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
		if !strings.Contains(stmt.SQL.String(), want) {
			t.Fatalf("%s must use current handler predicate: %s", relation, stmt.SQL.String())
		}
		if strings.Contains(stmt.SQL.String(), "mainDevelopers") || strings.Contains(stmt.SQL.String(), "distributedBy") {
			t.Fatalf("%s must not treat auxiliary participants as current handlers: %s", relation, stmt.SQL.String())
		}
	}

	query := applyHomeFocusToolbarFilters(base, "alice", DemandsReq{Keyword: "alice"})
	var rows []struct{ ID int }
	stmt := db.Table("(?) AS focused", query).Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	if !strings.Contains(stmt.SQL.String(), "currentHandlerDemandKeywordWhere") && !strings.Contains(stmt.SQL.String(), "LOWER(IFNULL(RD, '')) LIKE") {
		t.Fatalf("keyword must include current stage handler: %s", stmt.SQL.String())
	}
}
