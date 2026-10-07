package schedule

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestNormalizeDemandFilterWithWindows(t *testing.T) {
	// 选了窗口，默认/待排期均自动转为 all_open
	if got := NormalizeDemandFilterWithWindows("", "1"); got != FilterAllOpen {
		t.Fatalf("NormalizeDemandFilterWithWindows(\"\", \"1\") = %q, want %q", got, FilterAllOpen)
	}
	if got := NormalizeDemandFilterWithWindows(FilterUnscheduled, "1"); got != FilterAllOpen {
		t.Fatalf("NormalizeDemandFilterWithWindows(unscheduled, \"1\") = %q, want %q", got, FilterAllOpen)
	}
	// 显式指定 closed 时保留
	if got := NormalizeDemandFilterWithWindows(FilterClosed, "1"); got != FilterClosed {
		t.Fatalf("NormalizeDemandFilterWithWindows(closed, \"1\") = %q, want %q", got, FilterClosed)
	}
	// 没有窗口时，保持原默认待排期
	if got := NormalizeDemandFilterWithWindows("", ""); got != FilterUnscheduled {
		t.Fatalf("NormalizeDemandFilterWithWindows(\"\", \"\") = %q, want %q", got, FilterUnscheduled)
	}
}

func TestStageParameterCompatibility(t *testing.T) {
	bizReq1 := ListBizDemandsReq{Stage: "schedule"}
	bizReq1.Normalize()
	bizReq2 := ListBizDemandsReq{Stages: "schedule"}
	bizReq2.Normalize()
	if bizReq1.Stages != bizReq2.Stages {
		t.Fatalf("bizReq stage=schedule gave %q, stages=schedule gave %q; want equal", bizReq1.Stages, bizReq2.Stages)
	}

	stages := ParseCommaSeparatedStages("schedule")
	if len(stages) != 1 || stages[0] != "schedule" {
		t.Fatalf("ParseCommaSeparatedStages(\"schedule\") = %v, want [\"schedule\"]", stages)
	}
}

func TestGetWindowDemandCountDirectBindingSQL(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery(`(?s)SELECT.*zt_demandwindow.*zt_demand.*zt_demandwindow.*zt_story`).
		WithArgs(uint64(1), uint64(1), uint64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"demandCount"}).AddRow(2))

	count, err := NewRepo(db).GetWindowDemandCount(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetWindowDemandCount error: %v", err)
	}
	if count != 2 {
		t.Fatalf("GetWindowDemandCount = %d, want 2", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations not met: %v", err)
	}
}

func TestListFilterWindowsTeamNameAndOrder(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery(`SELECT count\(\*\) FROM .*zt_versionwindow.*`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	releaseDate := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT .* FROM .*zt_versionwindow.*ORDER BY releaseDate ASC, id ASC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "releaseDate", "teamgroup"}).
			AddRow(1, "26-0918窗口", releaseDate, 181))

	mock.ExpectQuery(`SELECT .* FROM .*zt_teamgroup.*WHERE id IN \(\?\)`).
		WithArgs(uint(181)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "parent"}).
			AddRow(181, "前端敏捷小组", 0))

	svc := NewService(NewRepo(db), nil, nil, nil, nil)
	opts, err := svc.ListFilterWindows(context.Background())
	if err != nil {
		t.Fatalf("ListFilterWindows error: %v", err)
	}
	if len(opts) != 1 {
		t.Fatalf("opts count = %d, want 1", len(opts))
	}
	if opts[0].TeamName != "前端敏捷小组" {
		t.Fatalf("opts[0].TeamName = %q, want %q", opts[0].TeamName, "前端敏捷小组")
	}
	if opts[0].ReleaseDate != "2026-09-18" {
		t.Fatalf("opts[0].ReleaseDate = %q, want %q", opts[0].ReleaseDate, "2026-09-18")
	}
}

func TestWindowFilterIncludesDirectAndPlanChain(t *testing.T) {
	cases := []struct {
		name  string
		build func(advancedFilterParams) filterClause
	}{
		{"biz", buildBizDemandAdvancedClause},
		{"indep", buildIndepStoryAdvancedClause},
	}
	for _, tc := range cases {
		withWindows := tc.build(advancedFilterParams{windowIDs: []uint{2}})
		if !strings.Contains(withWindows.sql, "zt_demandwindow") || !strings.Contains(withWindows.sql, "zt_versionwindowproduct") {
			t.Fatalf("%s window clause must contain direct binding and plan chain:\n%s", tc.name, withWindows.sql)
		}
		if got := strings.Count(withWindows.sql, "?"); got != len(withWindows.args) || got != 4 {
			t.Fatalf("%s placeholders = %d, args = %d, want 4", tc.name, got, len(withWindows.args))
		}
		for i, arg := range withWindows.args {
			if ids, ok := arg.([]uint); !ok || len(ids) != 1 || ids[0] != 2 {
				t.Fatalf("%s arg[%d] = %#v, want []uint{2}", tc.name, i, arg)
			}
		}
		if without := tc.build(advancedFilterParams{}); without.sql != "" || len(without.args) != 0 {
			t.Fatalf("%s clause without windows = %q %v, want empty", tc.name, without.sql, without.args)
		}
	}
}
