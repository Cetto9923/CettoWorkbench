// =============================================================================
// 文件: internal/pkg/perm/loader_test.go
// 模块: 基础设施
// 类型: test
// 职责: 验证 RBAC 权限加载器失效边过滤与边界情况。
// 依赖: github.com/DATA-DOG/go-sqlmock
//       gorm.io/driver/mysql
//       gorm.io/gorm
// =============================================================================

package perm

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	dialector := mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm database: %v", err)
	}
	return gormDB, mock
}

// expectedLoaderQueryPattern 匹配包含全部 4 个软删除/失效谓词的 SQL 查询。
var expectedLoaderQueryPattern = regexp.QuoteMeta(
	"SELECT DISTINCT rp.permCode FROM zt_role_permissions rp " +
		"JOIN zt_gf_user_roles ur ON ur.roleId = rp.roleId " +
		"JOIN zt_roles r ON r.id = ur.roleId " +
		"WHERE ur.userId = ? AND ur.deleted = ? AND r.deleted = ? AND r.isActive = ? AND rp.deletedAt IS NULL",
)

// CASE 1: active role + role.deleted = 0 + user-role.deleted = 0 + permission.deletedAt IS NULL
// -> permission loaded
func TestCase1_ActiveRole_ValidEdges_Loaded(t *testing.T) {
	db, mock := setupMockDB(t)
	userID := int64(1001)

	mock.ExpectQuery(expectedLoaderQueryPattern).
		WithArgs(userID, 0, 0, true).
		WillReturnRows(sqlmock.NewRows([]string{"permCode"}).
			AddRow(ScheduleList.String()).
			AddRow(ScheduleCreate.String()),
		)

	perms, err := LoadUserPermissionSet(context.Background(), db, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !perms[ScheduleList.String()] {
		t.Errorf("expected %s to be loaded", ScheduleList.String())
	}
	if !perms[ScheduleCreate.String()] {
		t.Errorf("expected %s to be loaded", ScheduleCreate.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// CASE 2: role.deleted = 1 -> DB query filters by r.deleted = 0, no rows returned -> not loaded
func TestCase2_RoleDeleted_NotLoaded(t *testing.T) {
	db, mock := setupMockDB(t)
	userID := int64(1002)

	mock.ExpectQuery(expectedLoaderQueryPattern).
		WithArgs(userID, 0, 0, true).
		WillReturnRows(sqlmock.NewRows([]string{"permCode"}))

	perms, err := LoadUserPermissionSet(context.Background(), db, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(perms) != 0 {
		t.Errorf("expected empty permissions for deleted role, got %v", perms)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// CASE 3: role.isActive = 0 -> DB query filters by r.isActive = true, no rows returned -> not loaded
func TestCase3_RoleInactive_NotLoaded(t *testing.T) {
	db, mock := setupMockDB(t)
	userID := int64(1003)

	mock.ExpectQuery(expectedLoaderQueryPattern).
		WithArgs(userID, 0, 0, true).
		WillReturnRows(sqlmock.NewRows([]string{"permCode"}))

	perms, err := LoadUserPermissionSet(context.Background(), db, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(perms) != 0 {
		t.Errorf("expected empty permissions for inactive role, got %v", perms)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// CASE 4: user-role.deleted = 1 -> DB query filters by ur.deleted = 0, no rows returned -> not loaded
func TestCase4_UserRoleDeleted_NotLoaded(t *testing.T) {
	db, mock := setupMockDB(t)
	userID := int64(1004)

	mock.ExpectQuery(expectedLoaderQueryPattern).
		WithArgs(userID, 0, 0, true).
		WillReturnRows(sqlmock.NewRows([]string{"permCode"}))

	perms, err := LoadUserPermissionSet(context.Background(), db, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(perms) != 0 {
		t.Errorf("expected empty permissions for deleted user-role edge, got %v", perms)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// CASE 5: permission.deletedAt IS NOT NULL -> DB query filters by rp.deletedAt IS NULL, no rows returned -> not loaded
func TestCase5_PermissionRevoked_NotLoaded(t *testing.T) {
	db, mock := setupMockDB(t)
	userID := int64(1005)

	mock.ExpectQuery(expectedLoaderQueryPattern).
		WithArgs(userID, 0, 0, true).
		WillReturnRows(sqlmock.NewRows([]string{"permCode"}))

	perms, err := LoadUserPermissionSet(context.Background(), db, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(perms) != 0 {
		t.Errorf("expected empty permissions for revoked permission edge, got %v", perms)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// CASE 6: 同一用户存在有效关系与失效关系 -> 仅有效关系的权限被返回
func TestCase6_MultipleEdges_OnlyValidReturned(t *testing.T) {
	db, mock := setupMockDB(t)
	userID := int64(1006)

	mock.ExpectQuery(expectedLoaderQueryPattern).
		WithArgs(userID, 0, 0, true).
		WillReturnRows(sqlmock.NewRows([]string{"permCode"}).
			AddRow(ScheduleUpdate.String()),
		)

	perms, err := LoadUserPermissionSet(context.Background(), db, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(perms) != 1 || !perms[ScheduleUpdate.String()] {
		t.Errorf("expected only %s to be returned, got %v", ScheduleUpdate.String(), perms)
	}
	if perms[ScheduleDelete.String()] {
		t.Errorf("revoked permission %s must not be returned", ScheduleDelete.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// CASE 7: Unknown permCode returned from DB -> ignored by All() whitelist
func TestCase7_UnknownPermCode_Ignored(t *testing.T) {
	db, mock := setupMockDB(t)
	userID := int64(1007)

	mock.ExpectQuery(expectedLoaderQueryPattern).
		WithArgs(userID, 0, 0, true).
		WillReturnRows(sqlmock.NewRows([]string{"permCode"}).
			AddRow("nonexistent:perm:code").
			AddRow(ScheduleList.String()),
		)

	perms, err := LoadUserPermissionSet(context.Background(), db, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(perms) != 1 || !perms[ScheduleList.String()] {
		t.Errorf("expected only valid known permission, got %v", perms)
	}
	if perms["nonexistent:perm:code"] {
		t.Errorf("unknown permission must not be present in set")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// CASE 8: DB error -> propagated upwards, not masked as empty set
func TestCase8_DBError_Propagated(t *testing.T) {
	db, mock := setupMockDB(t)
	userID := int64(1008)

	expectedErr := errors.New("database connection failed")
	mock.ExpectQuery(expectedLoaderQueryPattern).
		WithArgs(userID, 0, 0, true).
		WillReturnError(expectedErr)

	perms, err := LoadUserPermissionSet(context.Background(), db, userID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if len(perms) != 0 {
		t.Errorf("expected empty permissions on error, got %v", perms)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// CASE 9: Invalid input -> nil DB or non-positive userID returns empty set without DB query
func TestCase9_InvalidInput_EarlyExit(t *testing.T) {
	perms, err := LoadUserPermissionSet(context.Background(), nil, 1001)
	if err != nil || len(perms) != 0 {
		t.Errorf("expected empty set, nil err for nil db, got %v, %v", perms, err)
	}

	db, _ := setupMockDB(t)
	perms, err = LoadUserPermissionSet(context.Background(), db, 0)
	if err != nil || len(perms) != 0 {
		t.Errorf("expected empty set, nil err for userID=0, got %v, %v", perms, err)
	}
	perms, err = LoadUserPermissionSet(context.Background(), db, -1)
	if err != nil || len(perms) != 0 {
		t.Errorf("expected empty set, nil err for userID=-1, got %v, %v", perms, err)
	}
}
