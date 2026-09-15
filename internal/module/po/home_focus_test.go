package po

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
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

func TestHomeFocusStoryParticipateUsesAssociatedStories(t *testing.T) {
	db, _ := openSQLMock(t)
	repo := NewRepo(db, nil)
	query := repo.homeFocusStoryQuery(context.Background(), "alice", DemandsReq{
		Focus: "my_action", Status: "schedule", Relation: "participate",
	})
	var rows []struct{ ID int }
	stmt := query.Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	sql := stmt.SQL.String()
	for _, want := range []string{"fromDemand", "d.pool IS NOT NULL AND d.pool <> 0", "FIND_IN_SET"} {
		if !strings.Contains(sql, want) {
			t.Fatalf("participate schedule stories must include %q: %s", want, sql)
		}
	}
	if !strings.Contains(sql, "s.sourceType <> 'demandpool' AND s.assignedTo = ?") {
		t.Fatalf("participate schedule stories must include assigned independent stories: %s", sql)
	}
}

func TestHomeFocusParticipateScheduleExcludesBusinessDemand(t *testing.T) {
	db, _ := openSQLMock(t)
	repo := NewRepo(db, nil)

	schedule := repo.homeFocusQueryWithReviews(context.Background(), "alice", DemandsReq{
		Focus: "my_action", Status: "schedule", Relation: "participate",
	}, nil)
	var rows []struct{ ID int }
	scheduleStmt := db.Table("(?) AS focused", schedule).Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	if !strings.Contains(scheduleStmt.SQL.String(), "1 = 0") {
		t.Fatalf("participate schedule must not return the business-demand row: %s", scheduleStmt.SQL.String())
	}

	all := repo.homeFocusQueryWithReviews(context.Background(), "alice", DemandsReq{
		Focus: "my_action", Status: "all", Relation: "participate",
	}, nil)
	allStmt := db.Table("(?) AS focused", all).Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	if !strings.Contains(allStmt.SQL.String(), "NOT (status = ?") || !strings.Contains(allStmt.SQL.String(), "estimateLaunch") {
		t.Fatalf("participate all must retain clarify demand and exclude schedule demand: %s", allStmt.SQL.String())
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
	if !strings.Contains(clarifySQL, "CASE LOWER(TRIM(status))") || !strings.Contains(clarifySQL, ") = ?") {
		t.Fatalf("clarify focus story query must filter by the computed stage index: %s", clarifySQL)
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
	if !strings.Contains(scheduleSQL, "CASE LOWER(TRIM(status))") || !strings.Contains(scheduleSQL, ") = ?") {
		t.Fatalf("schedule focus story query must filter by the computed stage index: %s", scheduleSQL)
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
	if !strings.Contains(stmt.SQL.String(), "LOWER(IFNULL(BRA, '')) LIKE") || !strings.Contains(stmt.SQL.String(), "LOWER(IFNULL(QD, '')) LIKE") {
		t.Fatalf("keyword must include current stage handler: %s", stmt.SQL.String())
	}
}

func TestHomeFocusMyActionUsesStageRoleMatrix(t *testing.T) {
	db, _ := openSQLMock(t)
	repo := NewRepo(db, nil)
	query := repo.homeFocusQueryWithReviews(context.Background(), "alice", DemandsReq{
		Status: "all", Focus: "my_action", Page: 1, PageSize: 15,
	}, []int{101, 102})
	var rows []struct{ ID int }
	stmt := query.Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	sql := stmt.SQL.String()
	for _, want := range []string{
		"status IN ('draft', 'refuse') AND createdBy = ?",
		"status = 'wait' AND id IN (",
		"status = 'active' AND (",
		"status = 'clarified' AND (",
		"status = 'developing' AND BRA = ?",
		"status = 'testing' AND (QD = ? OR (accepter = ?",
		"status = 'waitacceptance' AND accepter = ?",
		"status IN ('acceptanced', 'waitdeliver') AND BRA = ?",
		"status = 'released' AND (originator = ? OR BRA = ?)",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("my_action role matrix missing %q: %s", want, sql)
		}
	}
	if strings.Contains(sql, "status IN ('developing', 'testing', 'waitacceptance') AND RD = ?") {
		t.Fatal("my_action must not use RD as a cross-stage fallback")
	}
}

func TestHomeFocusPageStoryBranchIsFlatAndZeroDateSafe(t *testing.T) {
	db, _ := openSQLMock(t)
	repo := NewRepo(db, nil)
	query := repo.focusPageQuery(context.Background(), "alice", DemandsReq{
		Status: "developing", Focus: "my_action", Page: 1, PageSize: 10,
	}, nil)
	var rows []struct{ ID int }
	stmt := query.Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	sql := stmt.SQL.String()
	if strings.Contains(sql, "focused_stories") {
		t.Fatalf("story branch must not nest a derived-table alias inside the UNION: %s", sql)
	}
	if strings.Contains(sql, "= '0000-00-00'") || strings.Contains(sql, "!= '0000-00-00'") {
		t.Fatalf("homepage focus query must not compare date columns with zero-date literals: %s", sql)
	}
	if !strings.Contains(sql, "CASE LOWER(TRIM(status))") || !strings.Contains(sql, ") = ?") {
		t.Fatalf("story branch must filter the computed stage in SQL: %s", sql)
	}
}

func TestHomeFocusStageSummary_DurationCalculation(t *testing.T) {
	db, mock := openSQLMock(t)
	repo := NewRepo(db, nil)

	mock.ExpectQuery("SELECT `demand` FROM `zt_demandreview` WHERE reviewer = ?").
		WithArgs("alice").
		WillReturnRows(sqlmock.NewRows([]string{"demand"}))

	mock.ExpectQuery("(?s)SELECT stage_index, COUNT\\(\\*\\) AS count, IFNULL\\(SUM\\(duration_days\\), 0\\) AS total_duration, COUNT\\(duration_days\\) AS duration_count FROM .* GROUP BY `stage_index`").
		WillReturnRows(sqlmock.NewRows([]string{"stage_index", "count", "total_duration", "duration_count"}).
			AddRow(2, 5, 50, 5). // 澄清：50/5 = 10天
			AddRow(3, 4, 48, 4)) // 排期：48/4 = 12天

	stages, err := repo.HomeFocusStageSummary(context.Background(), "alice", DemandsReq{
		Focus:      "my_action",
		ObjectType: "demand",
	})
	if err != nil {
		t.Fatalf("HomeFocusStageSummary error: %v", err)
	}

	if stages[2].AvgDurationDays != 10 || stages[2].AvgDurationText != "均10天" {
		t.Errorf("clarify stage duration mismatch: days=%d, text=%q, want 10, '均10天'", stages[2].AvgDurationDays, stages[2].AvgDurationText)
	}
	if stages[3].AvgDurationDays != 12 || stages[3].AvgDurationText != "均12天" {
		t.Errorf("schedule stage duration mismatch: days=%d, text=%q, want 12, '均12天'", stages[3].AvgDurationDays, stages[3].AvgDurationText)
	}
	// 汇总卡片
	if stages[0].AvgDurationDays != 11 || stages[0].AvgDurationText != "均11天" {
		t.Errorf("all stage duration mismatch: days=%d, text=%q, want 11, '均11天'", stages[0].AvgDurationDays, stages[0].AvgDurationText)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations unmet: %v", err)
	}
}

func TestHomeFocusMyActionEmptyReviewIDs_NoEmptyInClause(t *testing.T) {
	// 验证：当待评审切片为空 []int{} 时，生成的 SQL 必须使用恒假 (1 = 0)，不得生成非法空 IN ()
	where, args := currentHandlerDemandWhereWithReviews("alice", []int{})
	if strings.Contains(where, "IN ()") || strings.Contains(where, "IN (?)") {
		t.Fatalf("empty review IDs must not generate IN clause: %s", where)
	}
	if !strings.Contains(where, "status = 'wait' AND 1 = 0") {
		t.Fatalf("empty review IDs must set wait stage to 1 = 0: %s", where)
	}
	if len(args) != 14 {
		t.Fatalf("expected 14 args for empty review IDs, got %d", len(args))
	}

	// 验证 dry-run 查询不生成空 IN
	db, _ := openSQLMock(t)
	repo := NewRepo(db, nil)
	query := repo.homeFocusQueryWithReviews(context.Background(), "alice", DemandsReq{
		Status: "all", Focus: "my_action", Page: 1, PageSize: 15,
	}, []int{})
	var rows []struct{ ID int }
	stmt := query.Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	sql := stmt.SQL.String()
	if strings.Contains(sql, "IN ()") {
		t.Fatalf("dry-run SQL contains illegal empty IN clause: %s", sql)
	}
	if !strings.Contains(sql, "status = 'wait' AND 1 = 0") {
		t.Fatalf("dry-run SQL missing wait false guard: %s", sql)
	}
}
