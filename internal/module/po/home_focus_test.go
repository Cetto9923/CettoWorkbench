package po

import (
	"context"
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
	for _, relation := range []string{"all", "handling", "following"} {
		req := DemandsReq{Status: "all", Relation: relation}
		if errs := req.Validate(); len(errs) > 0 {
			t.Fatalf("relation %s: %v", relation, errs)
		}
	}
	if errs := (&DemandsReq{Status: "all", Relation: "owner"}).Validate(); len(errs) == 0 {
		t.Fatal("legacy relation must not silently acquire the new semantics")
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
