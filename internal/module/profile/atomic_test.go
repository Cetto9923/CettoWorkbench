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

func TestProfileSaveAtomicCases(t *testing.T) {
	failure := errors.New("fixture preference write failed")
	cases := []profileSaveCase{
		{"preference failure", true, 1, failure},
		{"deleted after read", false, 0, gorm.ErrRecordNotFound},
		{"unchanged profile", true, 0, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { testProfileSave(t, tc) })
	}
}

type profileSaveCase struct {
	name    string
	present bool
	changed int64
	want    error
}

func testProfileSave(t *testing.T, tc profileSaveCase) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{"id"})
	if tc.present {
		rows.AddRow(1)
	}
	mock.ExpectQuery("SELECT .*zt_user.*FOR UPDATE").WithArgs(int64(1), "fixture", "0", 1).WillReturnRows(rows)
	if tc.present {
		mock.ExpectExec("UPDATE `zt_user`").WillReturnResult(sqlmock.NewResult(0, tc.changed))
		prefs := mock.ExpectExec("INSERT INTO zt_wb_profile_pref_kv")
		if tc.want != nil {
			prefs.WillReturnError(tc.want)
		} else {
			prefs.WillReturnResult(sqlmock.NewResult(0, 0))
		}
	}
	if tc.want != nil {
		mock.ExpectRollback()
	} else {
		mock.ExpectCommit()
	}
	err = NewRepo(db).SaveSelfProfile(t.Context(), &model.User{ID: 1, Account: "fixture"}, UpdateReq{Email: "fixture@example.invalid"}, []string{"product"})
	if !errors.Is(err, tc.want) {
		t.Errorf("%s: expected %v, got %v", tc.name, tc.want, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("%s: %v", tc.name, err)
	}
	_ = sqlDB.Close()
}
