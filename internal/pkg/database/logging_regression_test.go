package database

import (
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Exercise GORM's actual query callback, not a hand-built Trace callback.
func TestGORMBoundParameterNeverReachesLog(t *testing.T) {
	path := initTestSQLLog(t)
	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{Logger: newGormSQLLogger()})
	if err != nil {
		t.Fatal(err)
	}
	secret := "prefix\\'CANARY_ESCAPED_BIND"
	mock.ExpectQuery("SELECT id FROM zt_user WHERE account = \\?").WithArgs(secret).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(7))
	var id int
	if err := db.Raw("SELECT id FROM zt_user WHERE account = ?", secret).Scan(&id).Error; err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "CANARY_ESCAPED_BIND") {
		t.Fatal("GORM expanded bind parameter leaked into SQL log")
	}
	if !strings.Contains(string(data), "SELECT id FROM zt_user WHERE account = ?") {
		t.Fatal("parameterized SQL template missing")
	}
}
