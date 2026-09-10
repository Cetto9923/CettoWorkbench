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
			query := repo.allStageRefQuery(context.Background(), "alice", DemandsReq{Priority: tc.pri})
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
	query := repo.allStageRefQuery(context.Background(), "alice", DemandsReq{Keyword: "搜索测试"})
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

	queryRel := repo.allStageRefQuery(context.Background(), "alice", DemandsReq{Relation: "handling"})
	stmtRel := db.Table("(?) AS all_stages", queryRel).Select("kind, id").
		Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	sqlRel := stmtRel.SQL.String()
	if !strings.Contains(sqlRel, "id IN (SELECT id FROM zt_demand WHERE") {
		t.Fatalf("expected relation handling filter in demand SQL: %s", sqlRel)
	}
	if !strings.Contains(sqlRel, "assignedTo = ?") {
		t.Fatalf("expected assignedTo = ? in story SQL: %s", sqlRel)
	}
}

func TestSingleStage_CountAndPagedWithFilters_Priority(t *testing.T) {
	db, _ := openSQLMock(t)
	repo := NewRepo(db, nil)

	filter := mysqlStageFilters["clarify"]

	// CountRoleDemandsWithFilters
	qCount := repo.roleDemandScopeWithFilters(context.Background(), "alice", filter, DemandsReq{Priority: "p1"})
	var total int64
	stmtCount := qCount.Session(&gorm.Session{DryRun: true}).Count(&total).Statement
	if !strings.Contains(stmtCount.SQL.String(), "pri = '1'") {
		t.Fatalf("expected clarify count to include pri = '1', got: %s", stmtCount.SQL.String())
	}

	// FindRoleDemandsPagedWithFilters
	qPaged := repo.roleDemandScopeWithFilters(context.Background(), "alice", filter, DemandsReq{Priority: "p1"})
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
