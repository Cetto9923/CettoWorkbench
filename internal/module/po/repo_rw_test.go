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

// TestEnsureDemandUnfollowed_UsesWriteDB 确认取消关注补写只走 writeDB，不触碰只读连接。
func TestEnsureDemandUnfollowed_UsesWriteDB(t *testing.T) {
	readDB, readMock := openSQLMock(t)
	writeDB, writeMock := openSQLMock(t)
	repo := NewRepo(readDB, writeDB)

	writeMock.ExpectQuery("(?i)SELECT count\\(\\*\\) FROM `zt_starinfo`").
		WithArgs("demand", int64(42), "alice").
		WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(1))
	writeMock.ExpectBegin()
	writeMock.ExpectExec("(?i)UPDATE `zt_starinfo`").
		WillReturnResult(sqlmock.NewResult(0, 1))
	writeMock.ExpectCommit()

	if err := repo.EnsureDemandUnfollowed(context.Background(), RepoSaveDemandFollowReq{
		Account: "alice", DemandID: 42,
	}); err != nil {
		t.Fatalf("EnsureDemandUnfollowed: %v", err)
	}
	if err := writeMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("writeDB expectations: %v", err)
	}
	if err := readMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("readDB must stay unused: %v", err)
	}
}

// TestRemoveProjectReportFollow_UsesProjectFollowContract guards against the
// former UI bug that sent a project ID to the demand-follow endpoint.
func TestRemoveProjectReportFollow_UsesProjectFollowContract(t *testing.T) {
	readDB, readMock := openSQLMock(t)
	writeDB, writeMock := openSQLMock(t)
	repo := NewRepo(readDB, writeDB)

	writeMock.ExpectExec("(?s)UPDATE zt_project AS p.*INNER JOIN zt_user AS u.*SET p.follow").
		WithArgs("alice", int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.RemoveProjectReportFollow(context.Background(), RepoRemoveProjectReportFollowReq{
		Account: "alice", ProjectID: 42,
	}); err != nil {
		t.Fatalf("RemoveProjectReportFollow: %v", err)
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

// TestSaveAllNoticeReads_UsesWriteDB 确认全部已读写入只走 writeDB，不触碰只读连接。
func TestSaveAllNoticeReads_UsesWriteDB(t *testing.T) {
	readDB, readMock := openSQLMock(t)
	writeDB, writeMock := openSQLMock(t)
	repo := NewRepo(readDB, writeDB)

	writeMock.ExpectExec("(?s)INSERT INTO zt_workbench_notify_reads").
		WithArgs("alice", sqlmock.AnyArg(), "alice", "alice").
		WillReturnResult(sqlmock.NewResult(0, 3))

	rows, err := repo.SaveAllNoticeReads(context.Background(), "alice", NoticeListReq{})
	if err != nil {
		t.Fatalf("SaveAllNoticeReads: %v", err)
	}
	if rows != 3 {
		t.Fatalf("RowsAffected=%d want 3", rows)
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
