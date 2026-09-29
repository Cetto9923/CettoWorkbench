// =============================================================================
// 文件: internal/module/testtask/service_write_authz_test.go
// 模块: 提测办理
// 类型: test
// 职责: 提测两个写入口的对象级鉴权回归：
//   - 无业需写权限：403，且不读 zt_build、不发起任何禅道请求
//   - 有写权限：放行至禅道调用（禅道走 httptest 假服务，禁止真实外部请求）
//   - actor 为空：403
// =============================================================================

package testtask

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"workbench/internal/config"
	"workbench/internal/model"
	"workbench/internal/module/demandauthz"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/zentao"
)

const (
	ttPMOQuery     = `SELECT COUNT\(\*\) FROM zt_gf_user_roles ur JOIN zt_roles r ON r\.id = ur\.roleId WHERE ur\.userId = \? AND ur\.deleted = '0' AND r\.deleted = '0' AND r\.isActive = 1 AND r\.code = 'pmo'`
	ttRelatedQuery = `SELECT COUNT\(\*\) FROM zt_demand d`
	ttManagedQuery = `SELECT d\.id, COALESCE\(d\.path, ''\) AS path FROM zt_dept d`
)

func newTesttaskMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open stub database: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm database: %v", err)
	}
	return db, mock
}

// newZentaoStub 返回指向 httptest 假禅道的客户端与命中计数器。
// 任何真实外部请求都不可能发生：apiBase 只指向本地测试服务器。
func newZentaoStub(t *testing.T, handler http.HandlerFunc) (*zentao.Client, *atomic.Int64) {
	t.Helper()
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return zentao.NewClient(config.ZentaoConfig{API: srv.URL}), &hits
}

func expectDemandAuthzDenied(mock sqlmock.Sqlmock, userID int64) {
	mock.ExpectQuery(ttPMOQuery).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(ttRelatedQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(ttManagedQuery).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}))
}

func expectDemandAuthzAllowed(mock sqlmock.Sqlmock, userID int64) {
	mock.ExpectQuery(ttPMOQuery).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(ttRelatedQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
}

func assertTTForbidden(t *testing.T, err error, wantMsg string) {
	t.Helper()
	bizErr, ok := errorx.IsBizError(err)
	if !ok {
		t.Fatalf("expected BizError, got %v", err)
	}
	if bizErr.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected %s, got %s", errorx.ErrCodeForbidden, bizErr.Code)
	}
	if bizErr.Msg != wantMsg {
		t.Fatalf("message = %q, want %q", bizErr.Msg, wantMsg)
	}
}

// 1. 创建测试单：无写权限 → 403，且一次禅道请求都没发出。
func TestCreateTesttasks_NoWritePermissionReturnsForbiddenWithoutZentaoCall(t *testing.T) {
	db, mock := newTesttaskMockDB(t)
	client, hits := newZentaoStub(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected zentao call: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	})
	svc := NewService(NewRepo(db), nil, client, nil)

	expectDemandAuthzDenied(mock, 15865)
	actor := &model.User{ID: 15865, Account: "demo_leader"}

	_, err := svc.CreateTesttasks(t.Context(), actor, 63450, CreateTesttasksReq{
		Tasks: []CreateTesttaskItem{{BuildID: 1, ProductID: 2, Name: "T"}},
	})
	assertTTForbidden(t, err, demandauthz.WriteDenialMessage)

	// 关键断言：zt_build 读取与禅道请求全部未发生。
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("forbidden create must not touch the database beyond authz reads: %v", err)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("forbidden create must not call zentao, got %d call(s)", n)
	}
}

// 2. 创建版本：无写权限 → 403，且零禅道请求。
func TestCreateBuilds_NoWritePermissionReturnsForbiddenWithoutZentaoCall(t *testing.T) {
	db, mock := newTesttaskMockDB(t)
	client, hits := newZentaoStub(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected zentao call: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	})
	svc := NewService(NewRepo(db), nil, client, nil)

	expectDemandAuthzDenied(mock, 15865)
	actor := &model.User{ID: 15865, Account: "demo_leader"}

	_, err := svc.CreateBuilds(t.Context(), actor, 63450, CreateBuildsReq{
		Builds: []CreateBuildItem{{ProjectID: 9, ExecutionID: 10, ProductID: 2, Name: "B"}},
	})
	assertTTForbidden(t, err, demandauthz.WriteDenialMessage)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("forbidden create must not touch the database beyond authz reads: %v", err)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("forbidden create must not call zentao, got %d call(s)", n)
	}
}

// 3. 有写权限（干系人）→ 放行，确实会打到假禅道（防止误拦）。
func TestCreateBuilds_StakeholderPassesAuthzAndCallsZentao(t *testing.T) {
	db, mock := newTesttaskMockDB(t)
	client, hits := newZentaoStub(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tokens" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"stub-token"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":77,"name":"B"}`))
	})
	svc := NewService(NewRepo(db), nil, client, nil)

	expectDemandAuthzAllowed(mock, 15864)
	actor := &model.User{ID: 15864, Account: "demo_po"}

	// ctx 需要带禅道账号，否则 Do 会因取不到账号而失败。
	ctx := zentao.WithAccount(t.Context(), "demo_po")
	resp, err := svc.CreateBuilds(ctx, actor, 63450, CreateBuildsReq{
		Builds: []CreateBuildItem{{ProjectID: 9, ExecutionID: 10, ProductID: 2, Name: "B"}},
	})
	if err != nil {
		t.Fatalf("stakeholder should pass authz and reach zentao, got %v", err)
	}
	if len(resp.Builds) != 1 || resp.Builds[0].BuildID != 77 {
		t.Fatalf("unexpected response: %+v", resp.Builds)
	}
	if hits.Load() == 0 {
		t.Fatal("stakeholder path should have called the stubbed zentao")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected mock state: %v", err)
	}
}

// 4. 有写权限的创建测试单同样放行到禅道。
func TestCreateTesttasks_StakeholderPassesAuthz(t *testing.T) {
	db, mock := newTesttaskMockDB(t)
	client, hits := newZentaoStub(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tokens" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"stub-token"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":88,"name":"T"}`))
	})
	svc := NewService(NewRepo(db), nil, client, nil)

	expectDemandAuthzAllowed(mock, 15864)
	mock.ExpectQuery("SELECT id, product, project, execution, name FROM `zt_build`").
		WillReturnRows(sqlmock.NewRows([]string{"id", "product", "project", "execution", "name"}).
			AddRow(1, 2, 9, 10, "build-1"))

	actor := &model.User{ID: 15864, Account: "demo_po"}
	ctx := zentao.WithAccount(t.Context(), "demo_po")
	resp, err := svc.CreateTesttasks(ctx, actor, 63450, CreateTesttasksReq{
		Tasks: []CreateTesttaskItem{{BuildID: 1, ProductID: 2, Name: "T", Owner: "demo_po"}},
	})
	if err != nil {
		t.Fatalf("stakeholder should pass authz and reach zentao, got %v", err)
	}
	if len(resp.Tasks) != 1 || resp.Tasks[0].TesttaskID != 88 {
		t.Fatalf("unexpected response: %+v", resp.Tasks)
	}
	if hits.Load() == 0 {
		t.Fatal("stakeholder path should have called the stubbed zentao")
	}
}

// 5. actor 为 nil → 403，且不查询关系、不调用禅道。
func TestRequireDemandWriteAccess_NilActorForbidden(t *testing.T) {
	db, mock := newTesttaskMockDB(t)
	client, hits := newZentaoStub(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected zentao call: %s %s", r.Method, r.URL.Path)
	})
	svc := NewService(NewRepo(db), nil, client, nil)

	assertTTForbidden(t, svc.RequireDemandWriteAccess(t.Context(), nil, 63450), "请先登录")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("nil actor must not touch the database: %v", err)
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("nil actor must not call zentao, got %d call(s)", n)
	}
}

// 6. 超管免关系查询直接放行。
func TestRequireDemandWriteAccess_SuperAdminAllowed(t *testing.T) {
	db, mock := newTesttaskMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil)

	if err := svc.RequireDemandWriteAccess(t.Context(), &model.User{ID: 7, Account: "003030", IsSuperAdmin: true}, 63450); err != nil {
		t.Fatalf("super admin should pass authz, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("super admin must not trigger relation queries: %v", err)
	}
}

// 7. 非法 demandID → 参数错误而非 403。
func TestRequireDemandWriteAccess_InvalidDemandID(t *testing.T) {
	db, _ := newTesttaskMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil)

	err := svc.RequireDemandWriteAccess(t.Context(), &model.User{ID: 1, Account: "demo_po", IsSuperAdmin: true}, 0)
	bizErr, ok := errorx.IsBizError(err)
	if !ok || bizErr.Code != errorx.ErrCodeInvalidParam {
		t.Fatalf("expected invalidparam, got %v", err)
	}
}
