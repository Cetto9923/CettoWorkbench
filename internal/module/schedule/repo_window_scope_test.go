package schedule

import (
	"context"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestFindWindowWorkItemsMergesAndDeduplicatesDemandSources(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery(`(?s)SELECT item_kind, item_id FROM \(.*zt_demandwindow.*UNION.*zt_versionwindowproduct.*zt_planstory.*zt_story.*ORDER BY item_kind ASC, item_id ASC`).
		WithArgs(uint64(77), uint64(77)).
		WillReturnRows(sqlmock.NewRows([]string{"item_kind", "item_id"}).
			AddRow("demand", 123).AddRow("story", 456).AddRow("demand", 789))
	items, err := NewRepo(db).FindWindowWorkItems(context.Background(), 77)
	if err != nil {
		t.Fatalf("FindWindowWorkItems() error = %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("work item count = %d, want 3: %+v", len(items), items)
	}
	if items[0] != (WindowWorkItem{Kind: "demand", ID: 123}) ||
		items[1] != (WindowWorkItem{Kind: "story", ID: 456}) ||
		items[2] != (WindowWorkItem{Kind: "demand", ID: 789}) {
		t.Fatalf("unexpected merged work items: %+v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL expectations: %v", err)
	}
}

func TestFindWindowWorkItemsEmptyWindowDoesNotQuery(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	items, err := NewRepo(db).FindWindowWorkItems(context.Background(), 0)
	if err != nil || len(items) != 0 {
		t.Fatalf("empty window items = %v, %v; want empty without error", items, err)
	}
}

func TestFindWindowWorkItemsSQLUsesDistinctUnion(t *testing.T) {
	if strings.Contains(findWindowWorkItemsSQL, "UNION ALL") {
		t.Fatal("window work-item sources must be de-duplicated")
	}
	if !strings.Contains(findWindowWorkItemsSQL, "\n  UNION\n") {
		t.Fatal("window work-item query must use distinct UNION")
	}
	if !strings.Contains(findWindowWorkItemsSQL, "s.fromDemand") || !strings.Contains(findWindowWorkItemsSQL, "ELSE s.id") {
		t.Fatal("plan-chain items must map demand-pool stories to demands and retain independent stories")
	}
}
