package po

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestBlockedKPI_SummaryUsesCurrentStateWithinActorScope(t *testing.T) {
	matcher := sqlmock.QueryMatcherFunc(func(_, query string) error {
		for _, required := range []string{
			"COUNT(CASE WHEN status = 'refuse'",
			"developFinish",
			"isManagerReview",
			"zt_demandmanagerreview",
			"accepter = ?",
			"deleted = ?",
			"NOT EXISTS",
		} {
			if !strings.Contains(query, required) {
				return fmt.Errorf("missing %s", required)
			}
		}
		for _, forbidden := range []string{"resultStatus", "testFinish"} {
			if strings.Contains(query, forbidden) {
				return fmt.Errorf("historical source %s must not imply current blocking", forbidden)
			}
		}
		return nil
	})
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("kpi summary").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "0", "closed", "0", "alice", "alice", "alice", "alice", "alice", "alice", "alice").
		WillReturnRows(sqlmock.NewRows([]string{"today", "overdue", "suspended", "blocked"}).AddRow(0, 0, 0, 2))
	summary, err := NewRepo(db, nil).CountKPISummary(context.Background(), "alice")
	if err != nil || summary.Blocked != 2 {
		t.Fatalf("blocked=%d err=%v summary=%+v", summary.Blocked, err, summary)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
