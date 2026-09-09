// =============================================================================
// 文件: internal/module/po/servicedone_authz_test.go
// 模块: PO 工作台
// 类型: test
// 职责: F01 已办动作详情对象级授权单元测试：
//   - nil / 空 actor 拒绝
//   - 动作 actor 自身允许访问
//   - 超级管理员旁路
//   - 普通账号命中对象关系（PO / assignedTo）放行
//   - 无关系账号 403
//   - 动作不存在 404
//   - DB 错误向上抛
//   - Handler 出口 401/403/404/500 区分
// =============================================================================

package po

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

func nowTime() time.Time {
	return time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
}

func TestDoneDetail_RejectsNilActor(t *testing.T) {
	gormDB, _ := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	_, err := svc.DoneDetail(t.Context(), nil, 100)
	biz, ok := errorx.IsBizError(err)
	if !ok || biz.Code != errorx.ErrCodeInvalidParam {
		t.Fatalf("expected invalidparam for nil actor, got: %v", err)
	}
}

func TestDoneDetail_RejectsEmptyActor(t *testing.T) {
	gormDB, _ := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	actor := &model.User{Account: "  "}
	_, err := svc.DoneDetail(t.Context(), actor, 100)
	biz, ok := errorx.IsBizError(err)
	if !ok || biz.Code != errorx.ErrCodeInvalidParam {
		t.Fatalf("expected invalidparam for empty actor, got: %v", err)
	}
}

func TestDoneDetail_RejectsZeroActionID(t *testing.T) {
	gormDB, _ := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	actor := &model.User{Account: "user_a"}
	_, err := svc.DoneDetail(t.Context(), actor, 0)
	biz, ok := errorx.IsBizError(err)
	if !ok || biz.Code != errorx.ErrCodeInvalidParam {
		t.Fatalf("expected invalidparam for zero id, got: %v", err)
	}
}

func TestDoneDetail_NotFoundReturns404Code(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	mock.ExpectQuery(`SELECT actor FROM .*zt_action.*`).
		WithArgs(int64(999)).
		WillReturnError(gorm.ErrRecordNotFound)

	actor := &model.User{Account: "user_a"}
	_, err := svc.DoneDetail(t.Context(), actor, 999)
	biz, ok := errorx.IsBizError(err)
	if !ok || biz.Code != errorx.ErrCodeNotFound {
		t.Fatalf("expected notfound, got: %v", err)
	}
}

func TestDoneDetail_AllowsActorAsActionActor(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	actor := &model.User{Account: "user_a"}

	// FindDoneActionActor → actor == user_a
	mock.ExpectQuery(`SELECT actor FROM .*zt_action.*`).
		WithArgs(int64(200)).
		WillReturnRows(sqlmock.NewRows([]string{"actor"}).AddRow("user_a"))

	// FindDoneActionDetail (First(&row) → GORM emits full row select against zt_action)
	mock.ExpectQuery(`SELECT \* FROM .*zt_action.*`).
		WithArgs(int64(200)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "objectType", "objectID", "action", "actor", "date",
		}).AddRow(200, "demand", 100, "reviewed", "user_a", nowTime()))

	// fetchObjectContexts: 1 demand
	mock.ExpectQuery(`SELECT d\.id, d\.name, d\.status, d\.assignedTo[\s\S]*FROM zt_demand AS d`).
		WithArgs(int64(100)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "status", "assignedTo", "product_name",
		}))

	// fetchActionHistories
	mock.ExpectQuery(`SELECT action, field, .old., .new.[\s\S]*FROM .*zt_history.*`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"action", "field", "old", "new"}))

	// nearby timeline
	mock.ExpectQuery(`SELECT id, action, actor, date[\s\S]*FROM .*zt_action.*`).
		WithArgs("demand", int64(100)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "action", "actor", "date"}))

	_, err := svc.DoneDetail(t.Context(), actor, 200)
	if err != nil {
		t.Fatalf("expected success for action actor self, got error: %v", err)
	}
}

func TestDoneDetail_ForbiddenForUnrelatedAccount(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	// FindDoneActionActor → actor == "user_other"
	mock.ExpectQuery(`SELECT actor FROM .*zt_action.*`).
		WithArgs(int64(300)).
		WillReturnRows(sqlmock.NewRows([]string{"actor"}).AddRow("user_other"))

	// CheckDoneActionObjectVisibility
	mock.ExpectQuery(`SELECT objectType, objectID FROM .*zt_action.*`).
		WithArgs(int64(300)).
		WillReturnRows(sqlmock.NewRows([]string{"objectType", "objectID"}).AddRow("demand", 500))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_demand d`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	actor := &model.User{Account: "user_stranger", IsSuperAdmin: false}
	_, err := svc.DoneDetail(t.Context(), actor, 300)
	biz, ok := errorx.IsBizError(err)
	if !ok || biz.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected forbidden, got: %v", err)
	}
}

func TestDoneDetail_AllowsAccountRelatedToObject(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	// FindDoneActionActor
	mock.ExpectQuery(`SELECT actor FROM .*zt_action.*`).
		WithArgs(int64(400)).
		WillReturnRows(sqlmock.NewRows([]string{"actor"}).AddRow("user_other"))

	// CheckDoneActionObjectVisibility: demand object, 命中 1 条
	mock.ExpectQuery(`SELECT objectType, objectID FROM .*zt_action.*`).
		WithArgs(int64(400)).
		WillReturnRows(sqlmock.NewRows([]string{"objectType", "objectID"}).AddRow("demand", 700))

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_demand d`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// FindDoneActionDetail (GORM First(&row) emits full-row scan SQL)
	mock.ExpectQuery(`SELECT \* FROM .*zt_action.*`).
		WithArgs(int64(400)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "objectType", "objectID", "action", "actor", "date",
		}).AddRow(400, "demand", 700, "reviewed", "user_other", nowTime()))

	mock.ExpectQuery(`SELECT d\.id, d\.name, d\.status, d\.assignedTo[\s\S]*FROM zt_demand AS d`).
		WithArgs(int64(700)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "status", "assignedTo", "product_name",
		}))
	mock.ExpectQuery(`SELECT action, field, .old., .new.[\s\S]*FROM .*zt_history.*`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"action", "field", "old", "new"}))
	mock.ExpectQuery(`SELECT id, action, actor, date[\s\S]*FROM .*zt_action.*`).
		WithArgs("demand", int64(700)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "action", "actor", "date"}))

	actor := &model.User{Account: "user_bra"} // 模拟 BRA 字段命中
	_, err := svc.DoneDetail(t.Context(), actor, 400)
	if err != nil {
		t.Fatalf("expected success for related object, got error: %v", err)
	}
}

func TestDoneDetail_SuperAdminBypassesObjectAuth(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	actor := &model.User{Account: "admin", IsSuperAdmin: true}

	// FindDoneActionActor
	mock.ExpectQuery(`SELECT actor FROM .*zt_action.*`).
		WithArgs(int64(500)).
		WillReturnRows(sqlmock.NewRows([]string{"actor"}).AddRow("user_other"))

	// FindDoneActionDetail (GORM First(&row) emits full-row scan SQL)
	mock.ExpectQuery(`SELECT \* FROM .*zt_action.*`).
		WithArgs(int64(500)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "objectType", "objectID", "action", "actor", "date",
		}).AddRow(500, "demand", 800, "reviewed", "user_other", nowTime()))
	mock.ExpectQuery(`SELECT d\.id, d\.name, d\.status, d\.assignedTo[\s\S]*FROM zt_demand AS d`).
		WithArgs(int64(800)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "status", "assignedTo", "product_name",
		}))
	mock.ExpectQuery(`SELECT action, field, .old., .new.[\s\S]*FROM .*zt_history.*`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"action", "field", "old", "new"}))
	mock.ExpectQuery(`SELECT id, action, actor, date[\s\S]*FROM .*zt_action.*`).
		WithArgs("demand", int64(800)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "action", "actor", "date"}))

	_, err := svc.DoneDetail(t.Context(), actor, 500)
	if err != nil {
		t.Fatalf("expected success for super admin, got error: %v", err)
	}
	// 超级管理员跳过 CheckDoneActionObjectVisibility，不应触发额外 query
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestDoneDetail_DBErrorPropagatesNotMapped(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	mock.ExpectQuery(`SELECT actor FROM .*zt_action.*`).
		WithArgs(int64(600)).
		WillReturnError(errors.New("connection refused"))

	actor := &model.User{Account: "user_a", IsSuperAdmin: true}
	_, err := svc.DoneDetail(t.Context(), actor, 600)
	if err == nil {
		t.Fatal("expected DB error to propagate, got nil")
	}
	if _, ok := errorx.IsBizError(err); ok {
		t.Fatalf("DB error must NOT be wrapped as BizError, got: %v", err)
	}
}

// -- Handler 出口映射 --

func setupDoneDetailHandler(t *testing.T) (*Handler, sqlmock.Sqlmock) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	gormDB, mock := setupMockDB(t)
	handler := NewHandler(NewService(NewRepo(gormDB, gormDB), nil, nil, zap.NewNop()), zap.NewNop())
	return handler, mock
}

func runDoneDetail(t *testing.T, handler *Handler, actor *model.User, actionID string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if actor != nil {
		c.Set("currentUser", actor)
	}
	req, _ := http.NewRequest(http.MethodGet, "/done/detail/"+actionID, nil)
	c.Request = req
	c.Params = gin.Params{gin.Param{Key: "actionId", Value: actionID}}
	handler.DoneDetail(c)
	return w
}

func TestDoneDetailHandler_403ForUnrelated(t *testing.T) {
	handler, mock := setupDoneDetailHandler(t)
	mock.ExpectQuery(`SELECT actor FROM .*zt_action.*`).
		WithArgs(int64(1100)).
		WillReturnRows(sqlmock.NewRows([]string{"actor"}).AddRow("user_other"))
	mock.ExpectQuery(`SELECT objectType, objectID FROM .*zt_action.*`).
		WithArgs(int64(1100)).
		WillReturnRows(sqlmock.NewRows([]string{"objectType", "objectID"}).AddRow("demand", 1200))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_demand d`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	w := runDoneDetail(t, handler, &model.User{Account: "user_outsider"}, "1100")
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected HTTP 403, got %d, body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "无权") {
		t.Errorf("expected body to mention 无权, got: %s", w.Body.String())
	}
}

func TestDoneDetailHandler_404ForMissing(t *testing.T) {
	handler, mock := setupDoneDetailHandler(t)
	mock.ExpectQuery(`SELECT actor FROM .*zt_action.*`).
		WithArgs(int64(1300)).
		WillReturnError(gorm.ErrRecordNotFound)

	w := runDoneDetail(t, handler, &model.User{Account: "admin", IsSuperAdmin: true}, "1300")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected HTTP 404, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestDoneDetailHandler_500ForDBFault(t *testing.T) {
	handler, mock := setupDoneDetailHandler(t)
	mock.ExpectQuery(`SELECT actor FROM .*zt_action.*`).
		WithArgs(int64(1400)).
		WillReturnError(errors.New("connection refused"))

	w := runDoneDetail(t, handler, &model.User{Account: "admin", IsSuperAdmin: true}, "1400")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected HTTP 500, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestDoneDetailHandler_401ForNilActor(t *testing.T) {
	handler, _ := setupDoneDetailHandler(t)

	w := runDoneDetail(t, handler, nil, "1500")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401, got %d, body: %s", w.Code, w.Body.String())
	}
}
