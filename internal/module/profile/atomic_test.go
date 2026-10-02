// =============================================================================
// 文件: internal/module/profile/atomic_test.go
// 模块: 个人资料
// 类型: test
// 职责: 偏好保存失败不得提交已修改的用户资料。
// =============================================================================

package profile

import (
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"testing"
	"workbench/internal/model"
)

func TestProfilePreferenceFailureRollsBackContact(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `zt_user`").WillReturnResult(sqlmock.NewResult(0, 1))
	failure := errors.New("fixture preference write failed")
	mock.ExpectExec("INSERT INTO zt_wb_profile_pref_kv").WillReturnError(failure)
	mock.ExpectRollback()
	err = NewRepo(db).SaveSelfProfile(t.Context(), &model.User{ID: 1, Account: "fixture"}, UpdateReq{Email: "fixture@example.invalid"}, []string{"product"})
	if !errors.Is(err, failure) {
		t.Fatalf("expected preference failure, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
