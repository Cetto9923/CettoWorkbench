// =============================================================================
// 文件: internal/module/role/handler_form_test.go
// 模块: 角色管理
// 类型: test
// 职责: 用模拟数据库验证角色 JSON 写入与 422 字段错误。
// =============================================================================
package role

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestRoleFormJSON(t *testing.T) {
	for _, method := range []string{"POST", "PUT"} {
		t.Run(method, func(t *testing.T) { testRoleFormJSON(t, method) })
	}
}

func testRoleFormJSON(t *testing.T, method string) {
	t.Helper()

	db, mock := newRoleTestDB(t)
	h := NewHandler(nil, nil, NewService(NewRepo(db)))
	router := gin.New()
	path := "/roles"
	if method == "PUT" {
		path += "/8"
		router.PUT("/roles/:id", h.Update)
		mock.ExpectQuery("SELECT .*zt_roles").WillReturnRows(sqlmock.NewRows([]string{"id", "isBuiltin"}).AddRow(8, false))
	} else {
		router.POST(path, h.Create)
	}
	mock.ExpectQuery("SELECT count.*zt_roles").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectBegin()
	if method == "POST" {
		mock.ExpectExec("INSERT INTO `zt_roles`").WillReturnResult(sqlmock.NewResult(8, 1))
	} else {
		mock.ExpectExec("UPDATE `zt_roles`").WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()
	req := httptest.NewRequest(method, path, strings.NewReader(`{"name":"运营","remark":"说明"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var response struct {
		Success     bool
		RedirectURL string `json:"redirectUrl"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || !response.Success || response.RedirectURL != "/admin/roles" {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRoleFormFieldErrors(t *testing.T) {
	h := NewHandler(nil, nil, nil)
	for _, method := range []string{"POST", "PUT"} {
		router := gin.New()
		router.POST("/roles", h.Create)
		router.PUT("/roles/:id", h.Update)
		path := "/roles"
		if method == "PUT" {
			path += "/8"
		}
		req := httptest.NewRequest(method, path, strings.NewReader(`{"name":"","remark":"保留输入"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 422 || !strings.Contains(w.Body.String(), `"field":"name"`) {
			t.Fatalf("%d %s", w.Code, w.Body)
		}
	}
}

func TestRoleBuiltinCannotUpdate(t *testing.T) {
	db, mock := newRoleTestDB(t)
	h := NewHandler(nil, nil, NewService(NewRepo(db)))
	mock.ExpectQuery("SELECT .*zt_roles").WillReturnRows(sqlmock.NewRows([]string{"id", "isBuiltin"}).AddRow(8, true))
	router := gin.New()
	router.PUT("/roles/:id", h.Update)
	req := httptest.NewRequest("PUT", "/roles/8", strings.NewReader(`{"name":"运营"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 422 || !strings.Contains(w.Body.String(), "内置角色不可修改") {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
