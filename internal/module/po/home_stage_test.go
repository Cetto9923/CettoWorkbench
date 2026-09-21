package po

import (
	"context"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"

	"workbench/internal/model"
)

func TestAllStageRefQuery_ToolbarFilters_Priority(t *testing.T) {
	db, _ := openSQLMock(t)
	repo := NewRepo(db, nil)

	cases := []struct {
		pri        string
		wantDemand string
		wantStory  int
	}{
		{"p1", "pri = '1'", 1},
		{"p2", "pri = '2'", 2},
		{"p3", "pri IN ('3', '4')", 3},
	}

	for _, tc := range cases {
		t.Run(tc.pri, func(t *testing.T) {
			query, queryErr := repo.allStageRefQuery(context.Background(), "alice", DemandsReq{Priority: tc.pri})
			if queryErr != nil {
				t.Fatal(queryErr)
			}
			var rows []struct {
				Kind string
				ID   int
			}
			stmt := db.Table("(?) AS all_stages", query).Select("kind, id").
				Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
			sql := stmt.SQL.String()

			if !strings.Contains(sql, tc.wantDemand) {
				t.Fatalf("expected demand SQL to contain %q, got: %s", tc.wantDemand, sql)
			}
			if !strings.Contains(sql, "pri = ?") && !strings.Contains(sql, "pri IN (?,?)") {
				t.Fatalf("expected story SQL to filter pri, got: %s", sql)
			}

			foundStoryPriArg := false
			for _, v := range stmt.Vars {
				if n, ok := v.(int); ok && n == tc.wantStory {
					foundStoryPriArg = true
					break
				}
				if n, ok := v.(int); ok && tc.pri == "p3" && (n == 3 || n == 4) {
					foundStoryPriArg = true
					break
				}
			}
			if !foundStoryPriArg {
				t.Fatalf("expected story pri arg in vars: %+v", stmt.Vars)
			}
		})
	}
}

func TestAllStageRefQuery_ToolbarFilters_KeywordAndRelation(t *testing.T) {
	db, mock := openSQLMock(t)
	repo := NewRepo(db, nil)

	// Keyword filter
	query, queryErr := repo.allStageRefQuery(context.Background(), "alice", DemandsReq{Keyword: "搜索测试"})
	if queryErr != nil {
		t.Fatal(queryErr)
	}
	var rows []struct {
		Kind string
		ID   int
	}
	stmt := db.Table("(?) AS all_stages", query).Select("kind, id").
		Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	sql := stmt.SQL.String()
	if !strings.Contains(sql, "LIKE ?") {
		t.Fatalf("expected keyword LIKE filter in SQL: %s", sql)
	}

	// Relation handling filter (expects review IDs query)
	mock.ExpectQuery("SELECT `demand` FROM `zt_demandreview` WHERE reviewer = ?").
		WithArgs("alice").
		WillReturnRows(sqlmock.NewRows([]string{"demand"}))

	queryRel, queryErr := repo.allStageRefQuery(context.Background(), "alice", DemandsReq{Relation: "handling"})
	if queryErr != nil {
		t.Fatal(queryErr)
	}
	stmtRel := db.Table("(?) AS all_stages", queryRel).Select("kind, id").
		Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	sqlRel := stmtRel.SQL.String()
	if !strings.Contains(sqlRel, "id IN (SELECT id FROM zt_demand WHERE") {
		t.Fatalf("expected relation handling filter in demand SQL: %s", sqlRel)
	}
	if !strings.Contains(sqlRel, "assignedTo = ?") {
		t.Fatalf("expected assignedTo = ? in story SQL: %s", sqlRel)
	}

	// Relation lead filter (我牵头: BRA = account, assignedTo = account)
	queryLead, queryErr := repo.allStageRefQuery(context.Background(), "alice", DemandsReq{Relation: "lead"})
	if queryErr != nil {
		t.Fatal(queryErr)
	}
	stmtLead := db.Table("(?) AS all_stages", queryLead).Select("kind, id").
		Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	sqlLead := stmtLead.SQL.String()
	if !strings.Contains(sqlLead, "BRA = ?") {
		t.Fatalf("expected BRA = ? in demand SQL: %s", sqlLead)
	}
	if !strings.Contains(sqlLead, "assignedTo = ?") {
		t.Fatalf("expected assignedTo = ? in story SQL: %s", sqlLead)
	}

	// Relation participate filter：澄清仍能看到业务需求，排期阶段切换为关联研发需求。
	queryPart, queryErr := repo.allStageRefQuery(context.Background(), "alice", DemandsReq{Relation: "participate"})
	if queryErr != nil {
		t.Fatal(queryErr)
	}
	stmtPart := db.Table("(?) AS all_stages", queryPart).Select("kind, id").
		Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	sqlPart := stmtPart.SQL.String()
	if !strings.Contains(sqlPart, "d.BRA <> ?") ||
		!strings.Contains(sqlPart, "d.pool IS NOT NULL AND d.pool <> 0") ||
		!strings.Contains(sqlPart, "zt_demandpool") ||
		!strings.Contains(sqlPart, "FIND_IN_SET") ||
		!strings.Contains(sqlPart, "fromDemand") {
		t.Fatalf("expected demand-pool analyst participation and associated story SQL: %s", sqlPart)
	}
	if !strings.Contains(sqlPart, "stage_index <>") {
		t.Fatalf("participate must exclude the business-demand schedule row: %s", sqlPart)
	}
}

func TestSingleStage_CountAndPagedWithFilters_Priority(t *testing.T) {
	db, _ := openSQLMock(t)
	repo := NewRepo(db, nil)

	filter := mysqlStageFilters["clarify"]

	// CountRoleDemandsWithFilters
	qCount, queryErr := repo.roleDemandScopeWithFilters(context.Background(), "alice", filter, DemandsReq{Priority: "p1"})
	if queryErr != nil {
		t.Fatal(queryErr)
	}
	var total int64
	stmtCount := qCount.Session(&gorm.Session{DryRun: true}).Count(&total).Statement
	if !strings.Contains(stmtCount.SQL.String(), "pri = '1'") {
		t.Fatalf("expected clarify count to include pri = '1', got: %s", stmtCount.SQL.String())
	}

	// FindRoleDemandsPagedWithFilters
	qPaged, queryErr := repo.roleDemandScopeWithFilters(context.Background(), "alice", filter, DemandsReq{Priority: "p1"})
	if queryErr != nil {
		t.Fatal(queryErr)
	}
	var dRows []DemandRow
	stmtPaged := qPaged.Session(&gorm.Session{DryRun: true}).Find(&dRows).Statement
	if !strings.Contains(stmtPaged.SQL.String(), "pri = '1'") {
		t.Fatalf("expected clarify paged to include pri = '1', got: %s", stmtPaged.SQL.String())
	}

	// acceptRefsPaged with Priority="p1"
	baseAccept := repo.roleDemandScope(context.Background(), "alice", mysqlStageFilters["accept"])
	baseAccept = repo.applyHomeFocusToolbarFiltersWithReviews(baseAccept, "alice", DemandsReq{Priority: "p1"}, nil)
	var acceptRows []struct{ ID int }
	stmtAccept := baseAccept.Session(&gorm.Session{DryRun: true}).Find(&acceptRows).Statement
	if !strings.Contains(stmtAccept.SQL.String(), "pri = '1'") {
		t.Fatalf("expected acceptRefsPaged to include pri = '1', got: %s", stmtAccept.SQL.String())
	}

	// FindScheduleStoryIDsWithFilters with Priority="p1"
	storyScope := repo.scheduleStoryScope(context.Background(), "alice")
	storyScope = applyStoryToolbarFilters(storyScope, "alice", DemandsReq{Priority: "p1"})
	var storyIDs []int
	stmtStory := storyScope.Session(&gorm.Session{DryRun: true}).Pluck("id", &storyIDs).Statement
	if !strings.Contains(stmtStory.SQL.String(), "pri = ?") {
		t.Fatalf("expected schedule story scope to include pri = ?, got: %s", stmtStory.SQL.String())
	}
}

func TestServiceDemands_AllStage_PriorityP1(t *testing.T) {
	db, mock := openSQLMock(t)
	repo := NewRepo(db, db)
	svc := NewService(repo, nil, nil, nil)

	// 1. Count query for FindAllStageRefsPaged (contains pri = '1')
	mock.ExpectQuery("(?s)SELECT count\\(\\*\\) FROM \\(SELECT kind, id.*pri = '1'.*AS all_stages").
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(1))

	// 2. Fetch page query
	mock.ExpectQuery("(?s)SELECT id, kind, stage_index FROM \\(SELECT kind, id.*pri = '1'.*AS all_stages").
		WillReturnRows(sqlmock.NewRows([]string{"id", "kind", "stage_index"}).AddRow(8001, "demand", 1))

	// 3. FindRoleDemandsByIDs
	mock.ExpectQuery("SELECT .* FROM `zt_demand`").
		WithArgs(8001).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "pri", "status", "hang", "assignedTo", "QD", "RD", "BRA", "pm"}).
			AddRow(8001, "高优需求", "1", "wait", "0", "alice", "", "", "", ""))

	// 4. Primary actions
	mock.ExpectQuery("SELECT .*FROM zt_demand WHERE id IN").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "stage", "status", "assignedTo", "accepter"}).
			AddRow(8001, "", "wait", "alice", ""))
	mock.ExpectQuery("(?s)SELECT d\\.id AS demand_id.*FROM zt_demand.*fromDemand IN").
		WithArgs(8001, 8001).
		WillReturnRows(sqlmock.NewRows([]string{"demand_id", "count", "first_id"}).AddRow(8001, 0, 0))
	mock.ExpectQuery("(?s)SELECT d\\.id AS demand_id.*zt_demandappraise").
		WithArgs("alice", 8001).
		WillReturnRows(sqlmock.NewRows([]string{"demand_id", "has_pending", "has_any"}).AddRow(8001, false, false))

	// 5. DeriveDemandPrimaryActions waitIDs review query
	mock.ExpectQuery("SELECT `demand` FROM `zt_demandreview`").
		WithArgs(8001, "alice", "").
		WillReturnRows(sqlmock.NewRows([]string{"demand"}).AddRow(8001))

	// 6. enrichDemandCanReview (status=wait)
	mock.ExpectQuery("SELECT `demand` FROM `zt_demandreview`").
		WithArgs(8001, "alice", "").
		WillReturnRows(sqlmock.NewRows([]string{"demand"}).AddRow(8001))

	resp, err := svc.Demands(context.Background(), &model.User{Account: "alice"}, DemandsReq{
		Status:   "all",
		Focus:    "all",
		Priority: "p1",
		Page:     1,
		PageSize: 15,
	})
	if err != nil {
		t.Fatalf("svc.Demands error: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got total=%d items=%d", resp.Total, len(resp.Items))
	}
	if resp.Items[0].Pri != "P1" {
		t.Fatalf("expected item pri P1, got %s", resp.Items[0].Pri)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations unmet: %v", err)
	}
}

func TestCountAllStageBreakdown_DurationCalculation(t *testing.T) {
	db, mock := openSQLMock(t)
	repo := NewRepo(db, nil)
	svc := NewService(repo, nil, nil, nil)

	mock.ExpectQuery("(?s)SELECT stage_index, kind, COUNT\\(\\*\\) AS count, IFNULL\\(SUM\\(duration_days\\), 0\\) AS total_duration, COUNT\\(duration_days\\) AS duration_count FROM .* GROUP BY stage_index, kind").
		WillReturnRows(sqlmock.NewRows([]string{"stage_index", "kind", "count", "total_duration", "duration_count"}).
			AddRow(1, "demand", 10, 80, 10). // 受理：均 80/10 = 8天
			AddRow(3, "demand", 2, 25, 2).   // 排期：均 25/2 = 12.5 -> 13天
			AddRow(3, "story", 5, 0, 0))     // 研发需求不计入

	breakdown, err := svc.countAllStageBreakdown(context.Background(), "alice")
	if err != nil {
		t.Fatalf("countAllStageBreakdown error: %v", err)
	}

	if len(breakdown) != len(valueStreamStages) {
		t.Fatalf("expected %d stages, got %d", len(valueStreamStages), len(breakdown))
	}

	// 验证受理阶段
	if breakdown[1].AvgDurationDays != 8 || breakdown[1].AvgDurationText != "均8天" {
		t.Errorf("accept stage duration mismatch, got days=%d, text=%q, want 8, '均8天'", breakdown[1].AvgDurationDays, breakdown[1].AvgDurationText)
	}

	// 验证排期阶段（含四舍五入）
	if breakdown[3].AvgDurationDays != 13 || breakdown[3].AvgDurationText != "均13天" {
		t.Errorf("schedule stage duration mismatch, got days=%d, text=%q, want 13, '均13天'", breakdown[3].AvgDurationDays, breakdown[3].AvgDurationText)
	}

	// 验证无需求阶段默认为 "—"
	if breakdown[4].AvgDurationDays != 0 || breakdown[4].AvgDurationText != "—" {
		t.Errorf("empty stage duration mismatch, got days=%d, text=%q, want 0, '—'", breakdown[4].AvgDurationDays, breakdown[4].AvgDurationText)
	}

	// 验证全部卡片全流均值：(80+25)/(10+2) = 105/12 = 8.75 -> 9天
	if breakdown[0].AvgDurationDays != 9 || breakdown[0].AvgDurationText != "均9天" {
		t.Errorf("all stage duration mismatch, got days=%d, text=%q, want 9, '均9天'", breakdown[0].AvgDurationDays, breakdown[0].AvgDurationText)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations unmet: %v", err)
	}
}
