// =============================================================================
// 文件: internal/module/po/service_detail_authz_test.go
// 模块: PO 工作台
// 类型: test
// 职责: F01 业务需求详情对象级授权单元测试：
//   - nil / 空 actor 拒绝
//   - 超级管理员旁路
//   - 普通用户在关系命中时放行
//   - 无关系账号 403
//   - 不存在 ID 404
//   - DB 错误向上抛，不映射为 404
//   - Handler 出口 401/403/404/500 区分
// =============================================================================

package po

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

func newDemandDetailMockRow(id uint, parent int64) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "parent", "pool", "pri", "category", "source", "sourceNote",
		"name", "desc", "verifyPlan", "feedbackBy", "feedbackedBy",
		"assignedTo", "QD", "RD", "BRA", "mainSystem", "reviewer",
		"reviewedDate", "status", "stage", "originator", "createdBy",
		"createdDate", "closedBy", "closedDate", "closedReason", "editedBy",
		"editedDate", "duration", "BSA", "estimateLaunch", "publishWindow",
		"deliverDate", "product", "accepter", "estimateDelivery",
		"developFinish", "testFinish", "verifyFinish",
		"assigned_to_name", "bra_name", "qd_name", "originator_name",
		"accepter_name", "reviewer_name", "originator_dept", "pool_name",
		"product_name", "main_system_name",
	}).AddRow(
		id, parent, 0, "2", "", "", "",
		"detail-name", "desc", "plan", "", "user_po",
		"user_assigned", "user_qd", "", "user_bra", "0", "user_reviewer", nil,
		"wait", "developing", "user_orig", "user_creator", nil, "user_close", nil,
		"", "user_edit", nil, "", "", nil, nil, nil, "0", "user_acc", 0, nil, nil, nil,
		"assigned-name", "bra-name", "qd-name", "orig-name",
		"acc-name", "rev-name", "—", "pool-name",
		"product-name", "system-name",
	)
}

func TestFindDemandDetailByID_AllowsZenTaoTopLevelParentSentinel(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demand d[\s\S]*WHERE d\.id = \?`).
		WithArgs(uint(700)).
		WillReturnRows(newDemandDetailMockRow(700, -1))

	row, err := repo.FindDemandDetailByID(t.Context(), 700)
	if err != nil {
		t.Fatalf("FindDemandDetailByID() error = %v", err)
	}
	if row.Parent != -1 {
		t.Fatalf("parent = %d, want ZenTao top-level sentinel -1", row.Parent)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetDemandDetail_RejectsNilActor(t *testing.T) {
	gormDB, _ := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	_, err := svc.GetDemandDetail(t.Context(), nil, 100)
	biz, ok := errorx.IsBizError(err)
	if !ok || biz.Code != errorx.ErrCodeInvalidParam {
		t.Fatalf("expected invalidparam for nil actor, got: %v", err)
	}
}

func TestGetDemandDetail_RejectsEmptyActor(t *testing.T) {
	gormDB, _ := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	actor := &model.User{Account: "   "}
	_, err := svc.GetDemandDetail(t.Context(), actor, 100)
	biz, ok := errorx.IsBizError(err)
	if !ok || biz.Code != errorx.ErrCodeInvalidParam {
		t.Fatalf("expected invalidparam for empty actor, got: %v", err)
	}
}

func TestGetDemandDetail_RejectsZeroDemandID(t *testing.T) {
	gormDB, _ := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	actor := &model.User{Account: "user_a"}
	_, err := svc.GetDemandDetail(t.Context(), actor, 0)
	biz, ok := errorx.IsBizError(err)
	if !ok || biz.Code != errorx.ErrCodeInvalidParam {
		t.Fatalf("expected invalidparam for zero id, got: %v", err)
	}
}

func TestGetDemandDetail_NotFoundReturns404Code(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demand d[\s\S]*WHERE d\.id = \?`).
		WithArgs(uint(999)).
		WillReturnError(gorm.ErrRecordNotFound)

	actor := &model.User{Account: "user_a"}
	_, err := svc.GetDemandDetail(t.Context(), actor, 999)
	biz, ok := errorx.IsBizError(err)
	if !ok || biz.Code != errorx.ErrCodeNotFound {
		t.Fatalf("expected notfound, got: %v", err)
	}
}

func TestGetDemandDetail_DBErrorPropagatesNotMapped(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demand d[\s\S]*WHERE d\.id = \?`).
		WithArgs(uint(500)).
		WillReturnError(errors.New("connection refused"))

	actor := &model.User{Account: "user_a", IsSuperAdmin: true}
	_, err := svc.GetDemandDetail(t.Context(), actor, 500)
	if err == nil {
		t.Fatal("expected DB error to propagate, got nil")
	}
	if _, ok := errorx.IsBizError(err); ok {
		t.Fatalf("DB error must NOT be wrapped as BizError, got: %v", err)
	}
}

func TestGetDemandDetail_ForbiddenForUnrelatedActor(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demand d[\s\S]*WHERE d\.id = \?`).
		WithArgs(uint(700)).
		WillReturnRows(newDemandDetailMockRow(700, 0))

	// 授权计数返回 0（无关系）
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_demand d`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	actor := &model.User{Account: "user_stranger", IsSuperAdmin: false}
	resp, err := svc.GetDemandDetail(t.Context(), actor, 700)
	if resp != nil {
		t.Errorf("expected nil resp on forbidden, got: %+v", resp)
	}
	biz, ok := errorx.IsBizError(err)
	if !ok || biz.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected forbidden, got: %v", err)
	}
}

func TestCheckDemandVisibility_RepoHelper(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_demand d`).
		WithArgs(uint(123), "user_a", "user_a", "user_a", "user_a", "user_a",
			"user_a", "user_a", "user_a", "user_a", "user_a", "user_a", "user_a").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	ok, err := repo.CheckDemandVisibility(t.Context(), 123, "user_a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Errorf("expected visible=true")
	}
}

func TestCheckDemandVisibility_EmptyArgsReturnFalse(t *testing.T) {
	gormDB, _ := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)

	if ok, err := repo.CheckDemandVisibility(t.Context(), 0, "user"); err != nil || ok {
		t.Errorf("zero id should return false,nil; got %v,%v", ok, err)
	}
	if ok, err := repo.CheckDemandVisibility(t.Context(), 1, ""); err != nil || ok {
		t.Errorf("empty account should return false,nil; got %v,%v", ok, err)
	}
}

// -- Handler 出口映射（401/403/404/500） --

func setupDemandDetailHandler(t *testing.T) (*Handler, sqlmock.Sqlmock) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	_ = NewDetailService(repo)
	handler := NewHandler(NewService(NewRepo(gormDB, gormDB), nil, nil, zap.NewNop()), zap.NewNop())
	return handler, mock
}

func runDemandDetail(t *testing.T, handler *Handler, actor *model.User, demandID string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("currentUser", actor)
	req, _ := http.NewRequest(http.MethodGet, "/demands/"+demandID+"/detail", nil)
	c.Request = req
	c.Params = gin.Params{gin.Param{Key: "id", Value: demandID}}
	handler.DemandDetail(c)
	return w
}

func TestDemandDetailHandler_403ForUnrelatedActor(t *testing.T) {
	handler, mock := setupDemandDetailHandler(t)

	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demand d[\s\S]*WHERE d\.id = \?`).
		WithArgs(uint(1500)).
		WillReturnRows(newDemandDetailMockRow(1500, 0))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_demand d`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	w := runDemandDetail(t, handler, &model.User{Account: "user_outsider"}, "1500")

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected HTTP 403, got %d, body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "无权") {
		t.Errorf("expected body to mention 无权, got: %s", w.Body.String())
	}
}

func TestDemandDetailHandler_404ForMissing(t *testing.T) {
	handler, mock := setupDemandDetailHandler(t)

	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demand d[\s\S]*WHERE d\.id = \?`).
		WithArgs(uint(1600)).
		WillReturnError(gorm.ErrRecordNotFound)

	w := runDemandDetail(t, handler, &model.User{Account: "admin", IsSuperAdmin: true}, "1600")

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected HTTP 404, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestDemandDetailHandler_500ForDBFault(t *testing.T) {
	handler, mock := setupDemandDetailHandler(t)

	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demand d[\s\S]*WHERE d\.id = \?`).
		WithArgs(uint(1700)).
		WillReturnError(errors.New("connection refused"))

	w := runDemandDetail(t, handler, &model.User{Account: "admin", IsSuperAdmin: true}, "1700")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected HTTP 500 for DB error, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestDemandDetailHandler_401ForNilActor(t *testing.T) {
	handler, _ := setupDemandDetailHandler(t)

	w := runDemandDetail(t, handler, nil, "1800")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401 for nil actor, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestDemandDetailView_InvalidIDReturnsBadRequest(t *testing.T) {
	handler, _ := setupDemandDetailHandler(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/demands/not-a-number", nil)
	c.Request = req
	c.Params = gin.Params{gin.Param{Key: "id", Value: "not-a-number"}}

	handler.DemandDetailView(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected HTTP 400, got %d", w.Code)
	}
}
