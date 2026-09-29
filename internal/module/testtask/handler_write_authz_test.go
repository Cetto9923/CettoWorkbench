// =============================================================================
// 文件: internal/module/testtask/handler_write_authz_test.go
// 模块: 提测办理
// 类型: test
// 职责: 提测两个写入口的 HTTP 403 出口回归：
//   - 无写权限时返回 403，响应体 {"success":false,"message":"…"}
//   - 403 优先于 400/422：即使提交缺参/非法 body 也返回 403
//   - 无权时零禅道请求
// =============================================================================

package testtask

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"workbench/internal/model"
	"workbench/internal/module/demandauthz"
)

func newTesttaskAuthzRouter(svc *Service, actor *model.User) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewHandler(svc, nil)
	router.Use(func(c *gin.Context) {
		c.Set("currentUser", actor)
		c.Next()
	})
	router.POST("/demands/:id/testtask/tasks", h.CreateTesttasks)
	router.POST("/demands/:id/testtask/builds", h.CreateBuilds)
	return router
}

func decodeTTForbidden(t *testing.T, rec *httptest.ResponseRecorder) string {
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

// 1. POST /demands/:id/testtask/tasks：无写权限 → 403（body 非法也不返回 400）。
func TestCreateTesttasksHandler_NoWritePermissionReturns403BeforeParsing(t *testing.T) {
	db, mock := newTesttaskMockDB(t)
	client, hits := newZentaoStub(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected zentao call: %s %s", r.Method, r.URL.Path)
	})
	svc := NewService(NewRepo(db), nil, client, nil)
	expectDemandAuthzDenied(mock, 15865)

	router := newTesttaskAuthzRouter(svc, &model.User{ID: 15865, Account: "demo_leader"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/demands/63450/testtask/tasks", strings.NewReader("{bad")))

	if msg := decodeTTForbidden(t, rec); msg != demandauthz.WriteDenialMessage {
		t.Fatalf("message = %q, want %q", msg, demandauthz.WriteDenialMessage)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("forbidden create must not call zentao, got %d call(s)", n)
	}
}

// 2. POST /demands/:id/testtask/builds：无写权限 → 403（body 非法也不返回 400）。
func TestCreateBuildsHandler_NoWritePermissionReturns403BeforeParsing(t *testing.T) {
	db, mock := newTesttaskMockDB(t)
	client, hits := newZentaoStub(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected zentao call: %s %s", r.Method, r.URL.Path)
	})
	svc := NewService(NewRepo(db), nil, client, nil)
	expectDemandAuthzDenied(mock, 15865)

	router := newTesttaskAuthzRouter(svc, &model.User{ID: 15865, Account: "demo_leader"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/demands/63450/testtask/builds", strings.NewReader("{bad")))

	if msg := decodeTTForbidden(t, rec); msg != demandauthz.WriteDenialMessage {
		t.Fatalf("message = %q, want %q", msg, demandauthz.WriteDenialMessage)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("forbidden create must not call zentao, got %d call(s)", n)
	}
}
