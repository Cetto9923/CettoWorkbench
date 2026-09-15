package testtask

import (
	"context"
	"database/sql/driver"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

// ---- 业务校验：先于所有 DB / 远程写入 ----

func TestCreateTesttasksRequiresAuthenticatedActor(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)
	_, err := svc.CreateTesttasks(context.Background(), nil, 1, CreateTesttasksReq{
		Tasks: []CreateTesttaskItem{{ProductID: 1, BuildID: 2, Name: "n", Begin: "2026-09-11", End: "2026-09-12", Owner: "x", Type: "feature", Pri: 1}},
	})
	assertForbidden(t, err)
}

func TestCreateTesttasksRejectsEmptyAccount(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)
	_, err := svc.CreateTesttasks(context.Background(), &model.User{Account: "   "}, 1, CreateTesttasksReq{
		Tasks: []CreateTesttaskItem{{ProductID: 1, BuildID: 2, Name: "n", Begin: "2026-09-11", End: "2026-09-12", Owner: "x", Type: "feature", Pri: 1}},
	})
	assertForbidden(t, err)
}

func TestCreateTesttasksRejectsZeroDemandBeforeAnyDBOrRemote(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil)
	_, err := svc.CreateTesttasks(context.Background(), &model.User{Account: "tester"}, 0, CreateTesttasksReq{
		Tasks: []CreateTesttaskItem{{ProductID: 1, BuildID: 2, Name: "n", Begin: "2026-09-11", End: "2026-09-12", Owner: "x", Type: "feature", Pri: 1}},
	})
	bizErr, ok := errorx.IsBizError(err)
	if !ok || bizErr.Code != errorx.ErrCodeInvalidParam {
		t.Fatalf("expected invalidparam, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("DB should not be touched: %v", err)
	}
}

func TestCreateTesttasksRejectsIncompleteJointRequest(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil)
	_, err := svc.CreateTesttasks(context.Background(), &model.User{Account: "tester"}, 123, CreateTesttasksReq{
		Joint: 1,
		Tasks: []CreateTesttaskItem{{ProductID: 1, BuildID: 2, Name: "n", Begin: "2026-09-11", End: "2026-09-12", Owner: "x", Type: "feature", Pri: 1}},
	})
	bizErr, ok := errorx.IsBizError(err)
	if !ok || bizErr.Code != errorx.ErrCodeInvalidParam {
		t.Fatalf("expected invalidparam for joint=1, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("DB should not be touched for joint=1: %v", err)
	}
}

func TestCreateTesttasksRejectsEmptyBatchBeforeAnyDB(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil)
	_, err := svc.CreateTesttasks(context.Background(), &model.User{Account: "tester"}, 123, CreateTesttasksReq{})
	bizErr, ok := errorx.IsBizError(err)
	if !ok || bizErr.Code != errorx.ErrCodeInvalidParam {
		t.Fatalf("expected invalidparam for empty batch, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("DB should not be touched for empty batch: %v", err)
	}
}

// ---- 需求预检：FindDemandContext 调用一次即停；不发起远程写入 ----

func TestCreateTesttasksDemandNotFoundStopsBeforeRemoteWrite(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	zt := newFakeZentao(t)
	svc := NewService(NewRepo(db), nil, zt.client(), nil)

	mock.ExpectQuery("(?s)SELECT d\\.id, d\\.name.*FROM zt_demand AS d.*WHERE d\\.id = \\? AND d\\.deleted = '0' LIMIT \\?").
		WithArgs(99999, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status", "stage", "BRA", "RD", "QD", "main_system_id", "product_name", "estimate_launch"}))

	_, err := svc.CreateTesttasks(context.Background(), &model.User{Account: "tester"}, 99999, CreateTesttasksReq{
		Tasks: []CreateTesttaskItem{{ProductID: 1, BuildID: 2, Name: "n", Begin: "2026-09-11", End: "2026-09-12", Owner: "x", Type: "feature", Pri: 1}},
	})
	bizErr, ok := errorx.IsBizError(err)
	if !ok || bizErr.Code != errorx.ErrCodeNotFound {
		t.Fatalf("expected notfound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected DB calls: %v", err)
	}
	if got := zt.createCalls.Load(); got != 0 {
		t.Fatalf("no remote create should fire, got %d", got)
	}
}

// ---- 归属校验：失败时不发起远程写入 ----

func TestCreateTesttasksRejectsMissingBuildBeforeRemoteWrite(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	zt := newFakeZentao(t)
	svc := NewService(NewRepo(db), nil, zt.client(), nil)

	expectDemandOK(mock, 123)
	expectBuildsByIDs(mock, 1, sqlmock.NewRows([]string{"id", "product", "project", "execution", "name"}))
	// 注意：build 不存在时直接 invalidparam，不调 VerifyExecutionBelongsToProject

	_, err := svc.CreateTesttasks(context.Background(), &model.User{Account: "tester"}, 123, CreateTesttasksReq{
		Tasks: []CreateTesttaskItem{{ProductID: 1, BuildID: 9999, Name: "n", Begin: "2026-09-11", End: "2026-09-12", Owner: "x", Type: "feature", Pri: 1}},
	})
	bizErr, ok := errorx.IsBizError(err)
	if !ok || bizErr.Code != errorx.ErrCodeInvalidParam {
		t.Fatalf("expected invalidparam for missing build, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected DB calls: %v", err)
	}
	if got := zt.createCalls.Load(); got != 0 {
		t.Fatalf("no remote create should fire, got %d", got)
	}
}

func TestCreateTesttasksRejectsBuildProductMismatch(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	zt := newFakeZentao(t)
	svc := NewService(NewRepo(db), nil, zt.client(), nil)

	expectDemandOK(mock, 123)
	expectBuildsByIDs(mock, 1, sqlmock.NewRows([]string{"id", "product", "project", "execution", "name"}).
		AddRow(101, 1, 11, 22, "buildA"))

	_, err := svc.CreateTesttasks(context.Background(), &model.User{Account: "tester"}, 123, CreateTesttasksReq{
		Tasks: []CreateTesttaskItem{{ProductID: 2 /* 与 build.product=1 不一致 */, BuildID: 101, Name: "n", Begin: "2026-09-11", End: "2026-09-12", Owner: "x", Type: "feature", Pri: 1}},
	})
	bizErr, ok := errorx.IsBizError(err)
	if !ok || bizErr.Code != errorx.ErrCodeInvalidParam {
		t.Fatalf("expected invalidparam for product mismatch, got %v", err)
	}
	if !strings.Contains(bizErr.Msg, "不属于所选产品") {
		t.Fatalf("expected '不属于所选产品' message, got %q", bizErr.Msg)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected DB calls: %v", err)
	}
	if got := zt.createCalls.Load(); got != 0 {
		t.Fatalf("no remote create should fire, got %d", got)
	}
}

func TestCreateTesttasksRejectsBuildWithoutExecution(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	zt := newFakeZentao(t)
	svc := NewService(NewRepo(db), nil, zt.client(), nil)

	expectDemandOK(mock, 123)
	expectBuildsByIDs(mock, 1, sqlmock.NewRows([]string{"id", "product", "project", "execution", "name"}).
		AddRow(101, 1, 11, 0, "buildA")) // execution=0

	_, err := svc.CreateTesttasks(context.Background(), &model.User{Account: "tester"}, 123, CreateTesttasksReq{
		Tasks: []CreateTesttaskItem{{ProductID: 1, BuildID: 101, Name: "n", Begin: "2026-09-11", End: "2026-09-12", Owner: "x", Type: "feature", Pri: 1}},
	})
	bizErr, ok := errorx.IsBizError(err)
	if !ok || bizErr.Code != errorx.ErrCodeInvalidParam {
		t.Fatalf("expected invalidparam for build without execution, got %v", err)
	}
	if !strings.Contains(bizErr.Msg, "缺少所属执行") {
		t.Fatalf("expected '缺少所属执行' message, got %q", bizErr.Msg)
	}
	if got := zt.createCalls.Load(); got != 0 {
		t.Fatalf("no remote create should fire, got %d", got)
	}
}

func TestCreateTesttasksRejectsExecutionNotBelongsToProject(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	zt := newFakeZentao(t)
	svc := NewService(NewRepo(db), nil, zt.client(), nil)

	expectDemandOK(mock, 123)
	expectBuildsByIDs(mock, 1, sqlmock.NewRows([]string{"id", "product", "project", "execution", "name"}).
		AddRow(101, 1, 11, 22, "buildA"))
	expectProjectBelongs(mock, 22, 11, 0)

	_, err := svc.CreateTesttasks(context.Background(), &model.User{Account: "tester"}, 123, CreateTesttasksReq{
		Tasks: []CreateTesttaskItem{{ProductID: 1, BuildID: 101, Name: "n", Begin: "2026-09-11", End: "2026-09-12", Owner: "x", Type: "feature", Pri: 1}},
	})
	bizErr, ok := errorx.IsBizError(err)
	if !ok || bizErr.Code != errorx.ErrCodeInvalidParam {
		t.Fatalf("expected invalidparam for execution-not-belongs-to-project, got %v", err)
	}
	if !strings.Contains(bizErr.Msg, "不属于项目") {
		t.Fatalf("expected '不属于项目' message, got %q", bizErr.Msg)
	}
	if got := zt.createCalls.Load(); got != 0 {
		t.Fatalf("no remote create should fire, got %d", got)
	}
}

// ---- 同批 (buildID, name) 去重 ----

func TestCreateTesttasksDedupesSameBuildAndName(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	zt := newFakeZentao(t)
	svc := NewService(NewRepo(db), nil, zt.client(), nil)

	expectDemandOK(mock, 123)
	expectBuildsByIDs(mock, 1, sqlmock.NewRows([]string{"id", "product", "project", "execution", "name"}).
		AddRow(101, 1, 11, 22, "buildA"))
	expectProjectBelongs(mock, 22, 11, 1)

	resp, err := svc.CreateTesttasks(context.Background(), &model.User{Account: "tester"}, 123, CreateTesttasksReq{
		Tasks: []CreateTesttaskItem{
			{ProductID: 1, BuildID: 101, Name: "tt-1", Begin: "2026-09-11", End: "2026-09-12", Owner: "x", Type: "feature", Pri: 1},
			{ProductID: 1, BuildID: 101, Name: "tt-1", Begin: "2026-09-11", End: "2026-09-12", Owner: "x", Type: "feature", Pri: 1}, // 重复
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Tasks) != 1 {
		t.Fatalf("expected 1 task after dedup, got %d", len(resp.Tasks))
	}
	if got := zt.createCalls.Load(); got != 1 {
		t.Fatalf("expected exactly 1 remote create after dedup, got %d", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected DB calls: %v", err)
	}
}

// ---- 部分成功：返回 *PartialTesttaskError ----

func TestCreateTesttasksPartialSuccessWrapsPartialError(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	zt := newFakeZentao(t)
	svc := NewService(NewRepo(db), nil, zt.client(), nil)

	expectDemandOK(mock, 123)
	// 一次性返回 3 个 build
	expectBuildsByIDs(mock, 3, sqlmock.NewRows([]string{"id", "product", "project", "execution", "name"}).
		AddRow(101, 1, 11, 22, "buildA").
		AddRow(102, 1, 11, 22, "buildB").
		AddRow(103, 1, 11, 22, "buildC"))
	// 全批归属校验通过后才开始远程写入。
	expectProjectBelongs(mock, 22, 11, 1)
	expectProjectBelongs(mock, 22, 11, 1)
	expectProjectBelongs(mock, 22, 11, 1)

	// 第 2 条请求失败：第 1 条已建立后第 2 条请求失败 → 期望 1 条 succeeded
	zt.failOnCall.Store(2)

	resp, err := svc.CreateTesttasks(context.Background(), &model.User{Account: "tester"}, 123, CreateTesttasksReq{
		Tasks: []CreateTesttaskItem{
			{ProductID: 1, BuildID: 101, Name: "tt-1", Begin: "2026-09-11", End: "2026-09-12", Owner: "x", Type: "feature", Pri: 1},
			{ProductID: 1, BuildID: 102, Name: "tt-2", Begin: "2026-09-11", End: "2026-09-12", Owner: "x", Type: "feature", Pri: 1},
			{ProductID: 1, BuildID: 103, Name: "tt-3", Begin: "2026-09-11", End: "2026-09-12", Owner: "x", Type: "feature", Pri: 1},
		},
	})
	if resp == nil {
		t.Fatal("expected partial resp to be non-nil")
	}
	if len(resp.Tasks) != 1 {
		t.Fatalf("expected 1 succeeded in partial resp, got %d", len(resp.Tasks))
	}
	var partial *PartialTesttaskError
	if !errors.As(err, &partial) {
		t.Fatalf("expected *PartialTesttaskError, got %T %v", err, err)
	}
	if len(partial.Succeeded) != 1 {
		t.Fatalf("expected 1 in partial.Succeeded, got %d", len(partial.Succeeded))
	}
	if !strings.Contains(partial.Cause, "第 2 项创建失败") {
		t.Fatalf("expected cause to mention 第 2 项, got %q", partial.Cause)
	}
	if !strings.Contains(partial.Cause, "已成功 1 条") {
		t.Fatalf("expected cause to mention 已成功 1 条, got %q", partial.Cause)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected DB calls: %v", err)
	}
}

// ---- 单条超时：标记「结果未知」 ----

func TestCreateTesttasksTimeoutMarksResultUnknown(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	zt := newFakeZentao(t)
	// 让服务端 sleep 超出 10s 单条超时；测试时长控制在合理范围：通过派生更短超时的 ctx。
	zt.delayCreate = 200 * time.Millisecond
	svc := NewService(NewRepo(db), nil, zt.client(), nil)

	expectDemandOK(mock, 123)
	expectBuildsByIDs(mock, 1, sqlmock.NewRows([]string{"id", "product", "project", "execution", "name"}).
		AddRow(101, 1, 11, 22, "buildA"))
	expectProjectBelongs(mock, 22, 11, 1)

	// 用父 ctx 派生 50ms 的子 ctx；服务内 withTesttaskTimeout 会基于父 ctx 再派生 10s，
	// 所以这里需要通过一个包装：把 testtaskCallTimeout 截断到 50ms。
	// 由于 service 内硬编码 10s，本测试用 context.WithCancel 让 cancel 在 50ms 后调用以模拟超时更直接。
	parentCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	resp, err := svc.CreateTesttasks(parentCtx, &model.User{Account: "tester"}, 123, CreateTesttasksReq{
		Tasks: []CreateTesttaskItem{{ProductID: 1, BuildID: 101, Name: "tt-1", Begin: "2026-09-11", End: "2026-09-12", Owner: "x", Type: "feature", Pri: 1}},
	})
	if resp == nil {
		t.Fatal("expected resp to be non-nil")
	}
	var partial *PartialTesttaskError
	if !errors.As(err, &partial) {
		// 远端被 cancel 时返回的可能是普通错误（非 DeadlineExceeded），但本测试只校验 cause 含「结果未知」或「失败」字样。
		t.Fatalf("expected *PartialTesttaskError, got %T %v", err, err)
	}
	if !strings.Contains(partial.Cause, "结果未知") {
		t.Fatalf("expected '结果未知' in cause for timeout, got %q", partial.Cause)
	}
}

// ---- Form Validate ----

func TestCreateTesttasksValidateFieldErrors(t *testing.T) {
	req := CreateTesttasksReq{Tasks: []CreateTesttaskItem{
		{ProductID: 0, BuildID: 0, Name: " ", Begin: "bad", End: "2026-09-11", Owner: "", Type: "", Pri: 9},
	}}
	errs := req.Validate()
	if len(errs) == 0 {
		t.Fatal("expected field errors, got none")
	}
	// 必须命中：productId、buildId、name、begin、end、owner、type、pri
	fields := map[string]bool{}
	for _, e := range errs {
		fields[e.Field] = true
	}
	for _, want := range []string{
		"tasks.0.productId", "tasks.0.buildId", "tasks.0.name", "tasks.0.begin",
		"tasks.0.owner", "tasks.0.type", "tasks.0.pri",
	} {
		if !fields[want] {
			t.Errorf("expected field error for %s, got %+v", want, errs)
		}
	}
}

func TestCreateTesttasksValidateRejectsEndBeforeBegin(t *testing.T) {
	req := CreateTesttasksReq{Tasks: []CreateTesttaskItem{
		{ProductID: 1, BuildID: 2, Name: "tt", Begin: "2026-09-12", End: "2026-09-11", Owner: "x", Type: "feature", Pri: 1},
	}}
	errs := req.Validate()
	hit := false
	for _, e := range errs {
		if e.Field == "tasks.0.end" && strings.Contains(e.Message, "结束日期不能早于") {
			hit = true
		}
	}
	if !hit {
		t.Fatalf("expected end-before-begin error, got %+v", errs)
	}
}

// ---- 辅助：mock 期望需求预检通过 ----

func expectDemandOK(mock sqlmock.Sqlmock, demandID uint) {
	mock.ExpectQuery("(?s)SELECT d\\.id, d\\.name.*FROM zt_demand AS d.*WHERE d\\.id = \\? AND d\\.deleted = '0' LIMIT \\?").
		WithArgs(demandID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status", "stage", "BRA", "RD", "QD", "main_system_id", "product_name", "estimate_launch"}).
			AddRow(demandID, "测试需求", "developing", "wait", "", "", "", 0, "主系统A", ""))
}

func anyBuildArgs(idCount int) []driver.Value {
	args := make([]driver.Value, 0, idCount+1)
	for i := 0; i < idCount; i++ {
		args = append(args, sqlmock.AnyArg())
	}
	args = append(args, "0")
	return args
}

func expectBuildsByIDs(mock sqlmock.Sqlmock, idCount int, rows *sqlmock.Rows) {
	mock.ExpectQuery("(?s)SELECT id, product, project, execution, name FROM `zt_build` WHERE id IN \\([^)]*\\) AND deleted").
		WithArgs(anyBuildArgs(idCount)...).
		WillReturnRows(rows)
}

func expectProjectBelongs(mock sqlmock.Sqlmock, executionID, projectID uint, count int) {
	mock.ExpectQuery("(?s)SELECT count.*FROM `zt_project` WHERE id = \\? AND project = \\? AND deleted").
		WithArgs(executionID, projectID, "0").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(count))
}

// ---- Handler：207 路径 ----

func TestHandler_CreateTesttasksReturns207OnPartial(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock := setupMockTesttaskDB(t)
	zt := newFakeZentao(t)
	svc := NewService(NewRepo(db), nil, zt.client(), nil)
	h := NewHandler(svc, nil)
	r := gin.New()
	// 注入当前用户：业务校验放在 Service / Handler，但 middleware.CurrentUser 需从 ctx 取
	r.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{Account: "tester", IsSuperAdmin: true})
		c.Next()
	})
	h.RegisterRoutes(r.Group(""))

	expectDemandOK(mock, 123)
	expectBuildsByIDs(mock, 2, sqlmock.NewRows([]string{"id", "product", "project", "execution", "name"}).
		AddRow(101, 1, 11, 22, "buildA").
		AddRow(102, 1, 11, 22, "buildB"))
	for i := 0; i < 2; i++ {
		expectProjectBelongs(mock, 22, 11, 1)
	}
	zt.failOnCall.Store(2)

	body := `{"joint":0,"tasks":[
		{"productId":1,"buildId":101,"name":"tt-1","begin":"2026-09-11","end":"2026-09-12","owner":"x","type":"feature","pri":1},
		{"productId":1,"buildId":102,"name":"tt-2","begin":"2026-09-11","end":"2026-09-12","owner":"x","type":"feature","pri":1}
	]}`
	req := httptest.NewRequest(http.MethodPost, "/demands/123/testtask/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "succeeded") {
		t.Errorf("expected 'succeeded' in body, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "第 2 项创建失败") {
		t.Errorf("expected '第 2 项创建失败' in body, got %s", w.Body.String())
	}
}

func TestHandler_CreateTesttasksReturnsValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, _ := setupMockTesttaskDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil)
	h := NewHandler(svc, nil)
	r := gin.New()
	// 注入当前用户：业务校验放在 Service / Handler，但 middleware.CurrentUser 需从 ctx 取
	r.Use(func(c *gin.Context) {
		c.Set("currentUser", &model.User{Account: "tester", IsSuperAdmin: true})
		c.Next()
	})
	h.RegisterRoutes(r.Group(""))

	body := `{"joint":1,"tasks":[{"productId":1,"buildId":2,"name":"n","begin":"2026-09-11","end":"2026-09-12","owner":"x","type":"feature","pri":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/demands/123/testtask/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}
