package po

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func openSQLMock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	gdb, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return gdb, mock
}

// TestSaveDemandFollow_UsesWriteDB 确认关注写入只走 writeDB，不触碰只读连接。
func TestSaveDemandFollow_UsesWriteDB(t *testing.T) {
	readDB, readMock := openSQLMock(t)
	writeDB, writeMock := openSQLMock(t)
	repo := NewRepo(readDB, writeDB)

	writeMock.ExpectQuery("(?i)SELECT count\\(\\*\\) FROM `zt_starinfo`").
		WithArgs("demand", int64(42), "alice").
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(0))
	writeMock.ExpectBegin()
	writeMock.ExpectExec("(?i)INSERT INTO `zt_starinfo`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	writeMock.ExpectCommit()

	followed := true
	if err := repo.SaveDemandFollow(context.Background(), RepoSaveDemandFollowReq{
		Account: "alice", DemandID: 42, Followed: followed,
	}); err != nil {
		t.Fatalf("SaveDemandFollow: %v", err)
	}
	if err := writeMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("writeDB expectations: %v", err)
	}
	if err := readMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("readDB must stay unused: %v", err)
	}
}

// TestSaveNoticeRead_UsesWriteDB 确认已读写入只走 writeDB。
func TestSaveNoticeRead_UsesWriteDB(t *testing.T) {
	readDB, readMock := openSQLMock(t)
	writeDB, writeMock := openSQLMock(t)
	repo := NewRepo(readDB, writeDB)

	writeMock.ExpectExec("(?s)INSERT INTO zt_workbench_notify_reads").
		WithArgs("alice", "alice", int64(7), "alice").
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.SaveNoticeRead(context.Background(), "alice", 7); err != nil {
		t.Fatalf("SaveNoticeRead: %v", err)
	}
	if err := writeMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("writeDB expectations: %v", err)
	}
	if err := readMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("readDB must stay unused: %v", err)
	}
}

// TestCheckNoticeAccess_UsesReadDB 确认授权查询仍走只读连接。
func TestCheckNoticeAccess_UsesReadDB(t *testing.T) {
	readDB, readMock := openSQLMock(t)
	writeDB, writeMock := openSQLMock(t)
	repo := NewRepo(readDB, writeDB)

	readMock.ExpectQuery(`SELECT id, toList FROM `+"`zt_notify`"+` WHERE id = \?`).
		WithArgs(int64(7), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "toList"}).AddRow(7, "alice"))

	exists, auth, err := repo.CheckNoticeAccess(context.Background(), "alice", 7)
	if err != nil {
		t.Fatalf("CheckNoticeAccess: %v", err)
	}
	if !exists || !auth {
		t.Fatalf("expected authorized notice, got exists=%v auth=%v", exists, auth)
	}
	if err := readMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("readDB expectations: %v", err)
	}
	if err := writeMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("writeDB must stay unused: %v", err)
	}
}
