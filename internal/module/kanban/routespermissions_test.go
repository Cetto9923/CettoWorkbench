// =============================================================================
// 文件: internal/module/kanban/routespermissions_test.go
// 模块: 工作看板
// 类型: test
// 职责: 生产路由三态及登录但无关联成员任务的拒绝回归。
// =============================================================================
package kanban

import (
	"encoding/json"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"workbench/internal/config"
	"workbench/internal/model"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/zentao"
)

func TestKanbanWritePermissionStates(t *testing.T) {
	for _, tc := range []struct {
		actor   *model.User
		granted bool
		want    int
	}{
		{nil, false, 401}, {&model.User{ID: 3, Account: "tester"}, false, 403}, {&model.User{ID: 3, Account: "tester"}, true, 400},
	} {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set("currentUser", tc.actor)
			c.Set("userPerms", map[string]bool{perm.KanbanStory.String(): tc.granted})
			c.Next()
		})
		NewHandler(nil, nil).RegisterRoutes(r.Group(""))
		out := httptest.NewRecorder()
		r.ServeHTTP(out, httptest.NewRequest("PUT", "/kanban/tasks/x", nil))
		if out.Code != tc.want {
			t.Fatalf("status=%d want=%d", out.Code, tc.want)
		}
	}
}

func TestKanbanRouteOutsideTaskScopeDenied(t *testing.T) {
	db, mock := kanbanMockDB(t)
	mock.ExpectQuery("SELECT id, status, assignedTo FROM `zt_task`").WillReturnRows(sqlmock.NewRows([]string{"id", "status", "assignedTo"}).AddRow(5, "wait", "outsider"))
	mock.ExpectQuery("SELECT g.id, g.name").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "PO", "manager"}))
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: 3, Account: "tester"})
		c.Set("userPerms", map[string]bool{perm.KanbanStory.String(): true})
		c.Next()
	})
	NewHandler(NewService(NewRepo(db), nil, nil, nil), nil).RegisterRoutes(r.Group(""))
	req := httptest.NewRequest("PUT", "/kanban/tasks/5", strings.NewReader(`{"status":"doing"}`))
	req.Header.Set("Content-Type", "application/json")
	out := httptest.NewRecorder()
	r.ServeHTTP(out, req)
	if out.Code != 403 {
		t.Fatalf("unrelated task allowed: %d %s", out.Code, out.Body.String())
	}
}

func TestKanbanMemberTaskAllowedWithActualEditor(t *testing.T) {
	db, mock := kanbanMockDB(t)
	mock.ExpectQuery("SELECT id, status, assignedTo FROM `zt_task`").WillReturnRows(sqlmock.NewRows([]string{"id", "status", "assignedTo"}).AddRow(5, "wait", "member"))
	mock.ExpectQuery("SELECT g.id, g.name").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "PO", "manager"}).AddRow(7, "team", "tester", ""))
	mock.ExpectQuery("SELECT root, account").WillReturnRows(sqlmock.NewRows([]string{"root", "account", "order"}).AddRow(7, "member", 1))
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tokens" {
			_, _ = w.Write([]byte(`{"token":"stub"}`))
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["lastEditedBy"] != "tester" {
			t.Errorf("wrong editor: %v", body)
		}
		hits.Add(1)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	t.Cleanup(server.Close)
	svc := NewService(NewRepo(db), nil, nil, zentao.NewClient(config.ZentaoConfig{API: server.URL}))
	actor := &model.User{ID: 3, Account: "tester"}
	err := svc.UpdateTaskStatus(zentao.WithAccount(t.Context(), actor.Account), actor, UpdateTaskStatusReq{ID: 5, Status: "doing"})
	if err != nil || hits.Load() != 1 {
		t.Fatalf("err=%v hits=%d", err, hits.Load())
	}
}
