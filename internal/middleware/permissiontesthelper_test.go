// =============================================================================
// 文件: internal/middleware/permissiontesthelper_test.go
// 模块: 中间件
// 类型: test
// 职责: 构造诊断入口登录测试的数据库 mock。
// 依赖: 无
// =============================================================================

package middleware

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func debugLoginMockDB(t *testing.T, userID int64, account string) *gorm.DB {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		_ = sqlDB.Close()
	})
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if userID > 0 {
		mock.ExpectQuery("SELECT .* FROM `zt_user`").WithArgs(userID, "0", 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "account", "deleted"}).AddRow(userID, account, "0"))
		if account != "admin" {
			mock.ExpectQuery("SELECT DISTINCT .* FROM zt_role_permissions").WithArgs(userID, true).
				WillReturnRows(sqlmock.NewRows([]string{"permCode"}))
		}
		mock.ExpectQuery("SELECT .* FROM `zt_menus`").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	}
	return db
}
