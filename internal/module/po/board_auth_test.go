// =============================================================================
// 文件: internal/module/po/board_auth_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证故事任务抽屉对象级权限边界与反向越权防护。
// =============================================================================

package po

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

// TestBoardTask_UnauthorizedStoryAccess 验证无权用户尝试通过故事 ID 查看任务抽屉时，Service 返回 403 Forbidden 业务错误。
func TestBoardTask_UnauthorizedStoryAccess(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	actor := &model.User{
		Account:      "user_stranger",
		Role:         "user",
		IsSuperAdmin: false,
	}

	// 1. Mock 查询 zt_story 元数据：归属于需求 50，负责人为 dev_other，创建人为 po_other
	mock.ExpectQuery(`SELECT id, fromDemand, assignedTo, openedBy FROM .zt_story. WHERE`).
		WithArgs(200, "0", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "fromDemand", "assignedTo", "openedBy"}).
			AddRow(200, 50, "dev_other", "po_other"))

	// 2. Mock 查询 zt_demand 关联权限：user_stranger 无任何角色且不在小组中，返回 count = 0
	mock.ExpectQuery(`SELECT count\(\*\) FROM zt_demand AS d WHERE`).
		WithArgs(50, "0", "user_stranger", "user_stranger", "user_stranger", "user_stranger", "user_stranger", "user_stranger").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// 3. Mock 查询 zt_task 用户在故事下是否有任务：返回 count = 0
	mock.ExpectQuery(`SELECT count\(\*\) FROM .zt_task. WHERE \(story = \? AND deleted = \?\) AND \(assignedTo = \? OR openedBy = \?\)`).
		WithArgs(200, "0", "user_stranger", "user_stranger").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	req := BoardTaskReq{
		StoryID:  200,
		PageSize: 200,
	}

	resp, err := svc.BoardTask(t.Context(), actor, req)
	if err == nil {
		t.Fatal("expected forbidden error for unauthorized actor accessing story drawer, got nil")
	}
	if resp != nil {
		t.Fatalf("expected nil resp on error, got %+v", resp)
	}

	bizErr, ok := errorx.IsBizError(err)
	if !ok {
		t.Fatalf("expected BizError, got: %v", err)
	}
	if bizErr.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected error code %q, got %q", errorx.ErrCodeForbidden, bizErr.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

// TestBoardTask_AuthorizedDirectAssignee 验证需求/任务负责人可正常访问抽屉。
func TestBoardTask_AuthorizedDirectAssignee(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	actor := &model.User{
		Account:      "dev_owner",
		Role:         "dev",
		IsSuperAdmin: false,
	}

	// 故事负责人为 dev_owner，直接命中第一规则，无需深入 demand / task 聚合
	mock.ExpectQuery(`SELECT id, fromDemand, assignedTo, openedBy FROM .zt_story. WHERE`).
		WithArgs(300, "0", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "fromDemand", "assignedTo", "openedBy"}).
			AddRow(300, 60, "dev_owner", "po_other"))

	// 后续正常加载看板小组与任务
	mock.ExpectQuery(`SELECT DISTINCT tg\.id, tg\.name FROM zt_teamgroup AS tg`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "敏捷一组"))

	mock.ExpectQuery(`SELECT zt_task\.id.*FROM .zt_task.`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "type", "status", "pri", "story", "assignedTo", "deadline"}).
			AddRow(1001, "前端实现", "devel", "doing", 2, 300, "dev_owner", nil))

	mock.ExpectQuery(`SELECT id, title FROM .zt_story. WHERE id IN`).
		WithArgs(300, "0").
		WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).AddRow(300, "故事标题"))

	mock.ExpectQuery(`SELECT DISTINCT account FROM .zt_team.`).
		WillReturnRows(sqlmock.NewRows([]string{"account"}).AddRow("dev_owner"))

	mock.ExpectQuery(`SELECT assignedTo, COUNT\(\*\) AS cnt FROM .zt_task.`).
		WillReturnRows(sqlmock.NewRows([]string{"assignedTo", "cnt"}).AddRow("dev_owner", 1))

	req := BoardTaskReq{
		StoryID:  300,
		PageSize: 200,
	}

	resp, err := svc.BoardTask(t.Context(), actor, req)
	if err != nil {
		t.Fatalf("expected success for direct assignee, got error: %v", err)
	}
	if resp == nil || len(resp.Columns) == 0 {
		t.Fatal("expected non-empty columns response for authorized user")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

// TestBoardTask_SuperAdminBypass 验证超级管理员跳过对象级鉴权。
func TestBoardTask_SuperAdminBypass(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())

	actor := &model.User{
		Account:      "admin",
		Role:         "admin",
		IsSuperAdmin: true,
	}

	// 超级管理员跳过 CheckStoryAccess，直接进入小组和任务查询
	mock.ExpectQuery(`SELECT DISTINCT tg\.id, tg\.name FROM zt_teamgroup AS tg`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "敏捷一组"))

	mock.ExpectQuery(`SELECT zt_task\.id.*FROM .zt_task.`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "type", "status", "pri", "story", "assignedTo", "deadline"}))

	mock.ExpectQuery(`SELECT DISTINCT account FROM .zt_team.`).
		WillReturnRows(sqlmock.NewRows([]string{"account"}))

	mock.ExpectQuery(`SELECT assignedTo, COUNT\(\*\) AS cnt FROM .zt_task.`).
		WillReturnRows(sqlmock.NewRows([]string{"assignedTo", "cnt"}))

	req := BoardTaskReq{
		StoryID:  500,
		PageSize: 200,
	}

	resp, err := svc.BoardTask(t.Context(), actor, req)
	if err != nil {
		t.Fatalf("expected success for superadmin, got error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response for superadmin")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

// TestBoardHandler_BoardTaskItems_ForbiddenHTTPStatus 验证 Handler 在遭遇对象级鉴权失败时正确返回 HTTP 403。
func TestBoardHandler_BoardTaskItems_ForbiddenHTTPStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB)
	svc := NewService(repo, nil, nil, zap.NewNop())
	handler := NewBoardHandler(svc, zap.NewNop())

	// 模拟越权查询：返回不匹配
	mock.ExpectQuery(`SELECT id, fromDemand, assignedTo, openedBy FROM .zt_story. WHERE`).
		WithArgs(400, "0", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "fromDemand", "assignedTo", "openedBy"}).
			AddRow(400, 80, "other_dev", "other_po"))

	mock.ExpectQuery(`SELECT count\(\*\) FROM zt_demand AS d WHERE`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`SELECT count\(\*\) FROM .zt_task. WHERE \(story = \? AND deleted = \?\) AND \(assignedTo = \? OR openedBy = \?\)`).
		WithArgs(400, "0", "unauthorized_user", "unauthorized_user").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	actor := &model.User{
		Account:      "unauthorized_user",
		Role:         "guest",
		IsSuperAdmin: false,
	}
	c.Set("currentUser", actor)

	req, _ := http.NewRequest(http.MethodGet, "/board/task/items?storyId=400", nil)
	c.Request = req

	handler.BoardTaskItems(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected HTTP status 403 Forbidden, got: %d, body: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response json: %v", err)
	}
	if body["message"] != "无权访问该研发需求任务" {
		t.Fatalf("expected error message %q, got: %v", "无权访问该研发需求任务", body["message"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
