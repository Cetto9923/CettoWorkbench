package po

import (
	"context"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestHomeFocusManagedAndRelatedSQL(t *testing.T) {
	db, _ := openSQLMock(t)
	repo := NewRepo(db, nil)
	ctx := context.Background()
	account := "alice"

	// 1. my_managed: assignedTo = ? OR BRA = ?
	queryManaged, err := repo.homeFocusQuery(ctx, account, DemandsReq{Status: "developing", Focus: "my_managed"})
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct{ ID int }
	stmtManaged := db.Table("(?) AS focused", queryManaged).Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	if !strings.Contains(stmtManaged.SQL.String(), "assignedTo = ? OR BRA = ?") {
		t.Fatalf("my_managed must filter by assignedTo = ? OR BRA = ?: %s", stmtManaged.SQL.String())
	}

	// 2. my_related: 包含 zt_starinfo 关注
	queryRelated, err := repo.homeFocusQuery(ctx, account, DemandsReq{Status: "developing", Focus: "my_related"})
	if err != nil {
		t.Fatal(err)
	}
	stmtRelated := db.Table("(?) AS focused", queryRelated).Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	if !strings.Contains(stmtRelated.SQL.String(), "zt_starinfo") {
		t.Fatalf("my_related must include zt_starinfo follow table in candidate base: %s", stmtRelated.SQL.String())
	}

	// 3. 研发需求 my_managed 与 my_related
	storyManaged := repo.homeFocusStoryQuery(ctx, account, DemandsReq{Focus: "my_managed", Status: "developing"})
	stmtStoryM := storyManaged.Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	if !strings.Contains(stmtStoryM.SQL.String(), "zt_story.assignedTo = ?") || !strings.Contains(stmtStoryM.SQL.String(), "ReqM = ?") {
		t.Fatalf("story my_managed must check assignedTo or ReqM: %s", stmtStoryM.SQL.String())
	}

	storyRelated := repo.homeFocusStoryQuery(ctx, account, DemandsReq{Focus: "my_related", Status: "developing"})
	stmtStoryR := storyRelated.Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	if !strings.Contains(stmtStoryR.SQL.String(), "zt_starinfo") {
		t.Fatalf("story my_related must include zt_starinfo: %s", stmtStoryR.SQL.String())
	}
}

// TestHomeFocusExcludesStarinfoForNonRelated 验证关注（zt_starinfo）只进「我相关」：
// overdue / suspended / my_action / today / blocked 均不得包含 zt_starinfo
func TestHomeFocusExcludesStarinfoForNonRelated(t *testing.T) {
	db, _ := openSQLMock(t)
	repo := NewRepo(db, nil)
	ctx := context.Background()
	account := "alice"

	focuses := []string{"overdue", "suspended", "my_action", "today", "blocked"}
	for _, focus := range focuses {
		query := repo.homeFocusQueryWithReviews(ctx, account, DemandsReq{Status: "all", Focus: focus}, []int{})
		var rows []struct{ ID int }
		stmt := db.Table("(?) AS focused", query).Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
		sqlStr := stmt.SQL.String()
		if strings.Contains(sqlStr, "zt_starinfo") {
			t.Fatalf("focus %q must NOT contain zt_starinfo, got: %s", focus, sqlStr)
		}
	}
}

// TestTeamDemandStageRetainsDateThresholds 验证团队首页计数隔离：
// 团队首页 SQL 保留 developFinish 与 deliverDate 门槛，与需求首页全量进格隔离
func TestTeamDemandStageRetainsDateThresholds(t *testing.T) {
	db, _ := openSQLMock(t)
	repo := NewRepo(db, nil)
	ctx := context.Background()

	teamSQL, _ := repo.teamDemandStageCase(ctx)
	if !strings.Contains(teamSQL, "developFinish <=") {
		t.Fatalf("teamDemandStageCase must retain developFinish <= threshold: %s", teamSQL)
	}
	if !strings.Contains(teamSQL, "deliverDate <=") {
		t.Fatalf("teamDemandStageCase must retain deliverDate <= threshold: %s", teamSQL)
	}

	// 个人价值流 mysqlStageFilters 不得包含 developFinishDue 或 deliverDateDue
	devFilter := mysqlStageFilters["developing"]
	if devFilter.developFinishDue {
		t.Fatalf("personal mysqlStageFilters['developing'] must not have developFinishDue")
	}
	accFilter := mysqlStageFilters["acceptanced"]
	if accFilter.deliverDateDue {
		t.Fatalf("personal mysqlStageFilters['acceptanced'] must not have deliverDateDue")
	}
}

// TestDeliverStoryScopeAllInclusive 验证研需求发起交付全量进格：
// deliverStoryScope 去除 deliverDate <= today 门槛
func TestDeliverStoryScopeAllInclusive(t *testing.T) {
	db, _ := openSQLMock(t)
	repo := NewRepo(db, nil)
	ctx := context.Background()

	q := repo.deliverStoryScope(ctx, "alice")
	var rows []struct{ ID int }
	stmt := q.Session(&gorm.Session{DryRun: true}).Find(&rows).Statement
	sqlStr := stmt.SQL.String()
	if strings.Contains(sqlStr, "deliverDate <=") {
		t.Fatalf("deliverStoryScope must NOT contain deliverDate <= threshold, got: %s", sqlStr)
	}
}
