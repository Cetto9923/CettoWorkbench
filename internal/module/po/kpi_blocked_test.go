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

func TestBlockedKPI_UsesCurrentStateWithinActorScope(t *testing.T) {
	matcher := sqlmock.QueryMatcherFunc(func(_, query string) error {
		for _, required := range []string{"status = ?", "developFinish", "isManagerReview", "zt_demandreview", "zt_demandmanagerreview", "accepter = ?", "deleted = ?", "NOT EXISTS"} {
			if !strings.Contains(query, required) {
				return fmt.Errorf("missing %s", required)
			}
		}
		for _, forbidden := range []string{"resultStatus", "testFinish", "deadline"} {
			if strings.Contains(query, forbidden) {
				return fmt.Errorf("historical/overdue source %s must not imply current blocking", forbidden)
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
	mock.ExpectQuery("current blocking").WithArgs("0", "closed", "0", "alice", "alice", "alice", "alice", "alice", "alice", "alice", "refuse", sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	count, err := NewRepo(db, nil).CountKPIBlocked(context.Background(), "alice")
	if err != nil || count != 2 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
