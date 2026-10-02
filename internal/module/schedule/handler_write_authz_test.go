// =============================================================================
// 文件: internal/module/schedule/handler_write_authz_test.go
// 模块: 排期工作台
// 类型: test
// 职责: 研发需求写入口与版本窗口写入口的 HTTP 403 出口回归：
//   - 无写权限时返回 403，且响应体为 {"success":false,"message":"…"}
//   - 403 优先于 400/422：即使提交缺参/非法 body 也返回 403
// =============================================================================

package schedule

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"

	"workbench/internal/model"
	"workbench/internal/pkg/workbenchroles"
)

// newAuthzRouter 造一个只挂目标写路由的 gin 引擎，并注入当前用户。
func newAuthzRouter(svc *Service, actor *model.User) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewHandler(nil, nil, svc, "")
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", actor)
		c.Next()
	})
	router.POST("/schedule/stories/:id/save-scheduling", h.SaveStoryScheduling)
	router.POST("/schedule/stories/:id/save-tasks", h.SaveStoryTasks)
	router.PUT("/schedule/windows/:id", h.UpdateWindow)
	router.DELETE("/schedule/windows/:id", h.DeleteWindow)
	return router
}

func decodeForbidden(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v; raw = %s", err, rec.Body.String())
	}
	if body.Success {
		t.Fatalf("success must be false, got %s", rec.Body.String())
	}
	if body.Message == "" {
		t.Fatalf("message must not be empty, got %s", rec.Body.String())
	}
	return body.Message
}

// 1. save-scheduling：无写权限 → 403（body 非法也不返回 400）。
func TestSaveStorySchedulingHandler_NoWritePermissionReturns403BeforeParsing(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)
	expectStoryAuthzDenied(mock, 15865, 7001, 900, 901)

	router := newAuthzRouter(svc, &model.User{ID: 15865, Account: "demo_outsider"})
	rec := httptest.NewRecorder()
	// 故意提交非法 JSON：鉴权必须先于参数解析生效。
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/schedule/stories/7001/save-scheduling", strings.NewReader("{bad")))

	if msg := decodeForbidden(t, rec); msg != StoryWriteDenialMessage {
		t.Fatalf("message = %q, want %q", msg, StoryWriteDenialMessage)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

// 2. save-tasks：无写权限 → 403（body 非法也不返回 400）。
func TestSaveStoryTasksHandler_NoWritePermissionReturns403BeforeParsing(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)
	expectStoryAuthzDenied(mock, 15865, 7002, 900, 901)

	router := newAuthzRouter(svc, &model.User{ID: 15865, Account: "demo_outsider"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/schedule/stories/7002/save-tasks", strings.NewReader("{bad")))

	if msg := decodeForbidden(t, rec); msg != StoryWriteDenialMessage {
		t.Fatalf("message = %q, want %q", msg, StoryWriteDenialMessage)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

// 3. PUT windows/:id：他人 → 403（body 非法也不返回 400）。
func TestUpdateWindowHandler_OtherUserReturns403BeforeParsing(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)
	expectWindowFind(mock, 501, "demo_po")
	mock.ExpectQuery(authzPMOQuery).
		WithArgs(int64(2), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	router := newAuthzRouter(svc, &model.User{ID: 2, Account: "demo_other"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/schedule/windows/501", strings.NewReader("{bad")))

	if msg := decodeForbidden(t, rec); msg != WindowWriteDenialMessage {
		t.Fatalf("message = %q, want %q", msg, WindowWriteDenialMessage)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("forbidden update must not write the database: %v", err)
	}
}

// 4. DELETE windows/:id：他人 → 403。
func TestDeleteWindowHandler_OtherUserReturns403(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)
	expectWindowFind(mock, 501, "demo_po")
	mock.ExpectQuery(authzPMOQuery).
		WithArgs(int64(2), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	router := newAuthzRouter(svc, &model.User{ID: 2, Account: "demo_other"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/schedule/windows/501", nil))

	if msg := decodeForbidden(t, rec); msg != WindowWriteDenialMessage {
		t.Fatalf("message = %q, want %q", msg, WindowWriteDenialMessage)
	}
	// 关键断言：软删除未发生。
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("forbidden delete must not write the database: %v", err)
	}
}

// 5. DELETE windows/:id：未登录 → 403（与 UpdateWindow 同一出口，不再放行成 200）。
func TestDeleteWindowHandler_NilActorReturns403(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)
	expectWindowFind(mock, 501, "demo_po")

	router := newAuthzRouter(svc, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/schedule/windows/501", nil))

	if msg := decodeForbidden(t, rec); msg != WindowWriteDenialMessage {
		t.Fatalf("message = %q, want %q", msg, WindowWriteDenialMessage)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unauthenticated delete must not write the database: %v", err)
	}
}

func TestSaveStoryTasksHandler_MissingStoryReturns404(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)
	mock.ExpectQuery(`FROM zt_story s`).
		WithArgs(uint(999999)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	router := newAuthzRouter(svc, &model.User{ID: 1, Account: "admin", IsSuperAdmin: true})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/schedule/stories/999999/save-tasks", strings.NewReader(`{"tasks":[]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusInternalServerError {
		t.Fatalf("missing story must not be 500; body = %s", rec.Body.String())
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v; raw = %s", err, rec.Body.String())
	}
	if body.Success || body.Message != "研发需求不存在" {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("missing story must not write: %v", err)
	}
}
