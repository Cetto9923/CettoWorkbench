// =============================================================================
// 文件: internal/module/agileteam/service_merge_test.go
// 模块: 敏捷小组治理
// 类型: action
// 职责: MergedMembersForGroups 合并口径单测（sqlmock）。
// =============================================================================

package agileteam

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestService(t *testing.T) (*Service, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	gdb, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("gorm: %v", err)
	}
	return NewService(NewRepo(gdb, gdb), zap.NewNop()), mock
}

func TestMergedMembersForGroupsMarksPending(t *testing.T) {
	svc, mock := newTestService(t)
	// formal members
	mock.ExpectQuery("(?s)FROM zt_team t.*root IN").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"group_id", "account", "name", "role", "hours", "days", "join_date"}).
			AddRow(uint(1), "formal1", "正式一", "研发", 7.0, 1, "2026-01-01").
			AddRow(uint(1), "leave1", "待离一", "测试", 6.0, 1, "2026-01-02"))
	// pending add
	mock.ExpectQuery("(?s)zt_wb_agileteam_adjustment_item.*actionType").
		WithArgs(StatusPending, sqlmock.AnyArg(), ActionAdd).
		WillReturnRows(sqlmock.NewRows([]string{"teamgroupId", "account", "role", "availableHours", "submittedBy"}).
			AddRow(uint(1), "newbie", "研发", 7.0, "po1"))
	// pending remove
	mock.ExpectQuery("(?s)zt_wb_agileteam_adjustment_item.*actionType").
		WithArgs(StatusPending, sqlmock.AnyArg(), ActionRemove).
		WillReturnRows(sqlmock.NewRows([]string{"teamgroupId", "account"}).
			AddRow(uint(1), "leave1"))
	// ResolveRealnames for newbie (GORM IN 查询；失败时回退账号名仍可接受)
	mock.ExpectQuery("(?s)SELECT .+ FROM `zt_user`").
		WithArgs("newbie").
		WillReturnRows(sqlmock.NewRows([]string{"account", "realname"}).
			AddRow("newbie", "新人甲"))

	out, err := svc.MergedMembersForGroups(context.Background(), []uint{1})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	list := out[1]
	if len(list) != 3 {
		t.Fatalf("want 3 members, got %d %+v", len(list), list)
	}
	byAcc := map[string]MemberItem{}
	for _, m := range list {
		byAcc[m.Account] = m
	}
	if byAcc["formal1"].Status != "formal" {
		t.Fatalf("formal1 status=%s", byAcc["formal1"].Status)
	}
	if byAcc["leave1"].Status != "pendingRemove" {
		t.Fatalf("leave1 status=%s", byAcc["leave1"].Status)
	}
	if byAcc["newbie"].Status != "pendingAdd" {
		t.Fatalf("newbie=%+v", byAcc["newbie"])
	}
	if byAcc["newbie"].Name == "" {
		t.Fatal("newbie name empty")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
