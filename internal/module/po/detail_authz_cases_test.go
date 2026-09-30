// =============================================================================
// 文件: internal/module/po/detail_authz_cases_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 业务需求详情放开只读授权及边界测试：
//   - 团队长看本团队需求 = 可见 (200)
//   - 团队长看外团队需求：有 ScheduleList 可见，无该权限为 403
//   - PMO 角色 = 可见 (200)
//   - 需求查询权限用户 = 可见 (200)
//   - 无关普通用户 = 403
//   - 只读角色调用写接口 = 403
//   - DemandDetailView 渲染标准 HTML 403 页面
//   - 逾期天数脏数据兜底 (< 2000-01-01)
// =============================================================================

package po

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/perm"
	"workbench/internal/pkg/workbenchroles"
)

// 1. 团队长看本团队需求 = 可见 (200)
func TestDemandDetailAuthZ_LeaderDeptDemandVisible(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	// mock demand row
	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demand d[\s\S]*WHERE d\.id = \?`).
		WithArgs(uint(1001)).
		WillReturnRows(newDemandDetailMockRow(1001, 0))

	// PMO 角色检查：未命中
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_gf_user_roles ur JOIN zt_roles r ON r\.id = ur\.roleId WHERE ur\.userId = \? AND ur\.deleted = '0' AND r\.deleted = '0' AND r\.isActive = 1 AND r\.code = \?`).
		WithArgs(int64(10), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// 个人干系人检查：未命中
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_demand d`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// 部门经理检查：查询账号负责的部门（部门 14）
	mock.ExpectQuery(`SELECT d\.id, COALESCE\(d\.path, ''\) AS path FROM zt_dept d`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}).AddRow(14, ",1,14,"))

	// 部门树展开
	mock.ExpectQuery(`SELECT DISTINCT id FROM zt_dept WHERE id IN \(\?\) OR path LIKE \?`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(14))

	// 团队长需求可见性检查：命中本部门干系人
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_demand d[\s\S]*WHERE d\.id = \?`).
		WithArgs(uint(1001), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	actor := &model.User{ID: 10, Account: "demo_leader", IsSuperAdmin: false}
	row, err := svc.GetDemandDetailAuthZ(t.Context(), actor, 1001)
	if err != nil {
		t.Fatalf("expected visible, got error: %v", err)
	}
	if row == nil || row.ID != 1001 {
		t.Fatalf("expected demand 1001, got %+v", row)
	}
}

// 2. 团队长看外团队需求 = 403
func TestDemandDetailAuthZ_LeaderOtherDeptDemandForbidden(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demand d[\s\S]*WHERE d\.id = \?`).
		WithArgs(uint(1002)).
		WillReturnRows(newDemandDetailMockRow(1002, 0))

	// PMO 角色检查：未命中
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_gf_user_roles ur JOIN zt_roles r ON r\.id = ur\.roleId WHERE ur\.userId = \? AND ur\.deleted = '0' AND r\.deleted = '0' AND r\.isActive = 1 AND r\.code = \?`).
		WithArgs(int64(10), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// 个人干系人检查：未命中
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_demand d`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// 部门经理检查：命中部门 14
	mock.ExpectQuery(`SELECT d\.id, COALESCE\(d\.path, ''\) AS path FROM zt_dept d`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}).AddRow(14, ",1,14,"))
	mock.ExpectQuery(`SELECT DISTINCT id FROM zt_dept WHERE id IN \(\?\) OR path LIKE \?`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(14))

	// 团队长需求可见性检查：未命中（外部门干系人）
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_demand d[\s\S]*WHERE d\.id = \?`).
		WithArgs(uint(1002), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	actor := &model.User{ID: 10, Account: "demo_leader", IsSuperAdmin: false}
	row, err := svc.GetDemandDetailAuthZ(t.Context(), actor, 1002)
	if row != nil {
		t.Fatalf("expected nil row, got %+v", row)
	}
	biz, ok := errorx.IsBizError(err)
	if !ok || biz.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected forbidden error, got: %v", err)
	}
}

func TestDemandDetailAuthZ_LeaderOtherDeptQueryPermVisible(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demand d[\s\S]*WHERE d\.id = \?`).
		WithArgs(uint(1002)).
		WillReturnRows(newDemandDetailMockRow(1002, 0))

	// PMO 角色检查：未命中
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_gf_user_roles ur JOIN zt_roles r ON r\.id = ur\.roleId WHERE ur\.userId = \? AND ur\.deleted = '0' AND r\.deleted = '0' AND r\.isActive = 1 AND r\.code = \?`).
		WithArgs(int64(10), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// 个人干系人检查：未命中
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_demand d`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// 部门经理检查：命中部门 14
	mock.ExpectQuery(`SELECT d\.id, COALESCE\(d\.path, ''\) AS path FROM zt_dept d`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}).AddRow(14, ",1,14,"))
	mock.ExpectQuery(`SELECT DISTINCT id FROM zt_dept WHERE id IN \(\?\) OR path LIKE \?`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(14))

	// 团队长需求可见性检查：未命中（外部门干系人）
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_demand d[\s\S]*WHERE d\.id = \?`).
		WithArgs(uint(1002), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	actor := &model.User{ID: 10, Account: "demo_leader", IsSuperAdmin: false}
	ctx := perm.WithGranted(t.Context(), map[string]bool{perm.ScheduleList.String(): true})
	row, err := svc.GetDemandDetailAuthZ(ctx, actor, 1002)
	if err != nil || row == nil || row.ID != 1002 {
		t.Fatalf("expected visible with ScheduleList, row=%+v err=%v", row, err)
	}
}

// 3. PMO 角色 = 全量只读放行 (200)
func TestDemandDetailAuthZ_PMOGlobalVisible(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demand d[\s\S]*WHERE d\.id = \?`).
		WithArgs(uint(1003)).
		WillReturnRows(newDemandDetailMockRow(1003, 0))

	// PMO 角色查询：命中 pmo
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_gf_user_roles ur JOIN zt_roles r ON r\.id = ur\.roleId WHERE ur\.userId = \? AND ur\.deleted = '0' AND r\.deleted = '0' AND r\.isActive = 1 AND r\.code = \?`).
		WithArgs(int64(88), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	actor := &model.User{ID: 88, Account: "user_pmo", IsSuperAdmin: false}
	row, err := svc.GetDemandDetailAuthZ(t.Context(), actor, 1003)
	if err != nil {
		t.Fatalf("expected visible for PMO, got error: %v", err)
	}
	if row == nil || row.ID != 1003 {
		t.Fatalf("expected demand 1003, got %+v", row)
	}
}

// 4. 产品经理持需求查询权限看他人需求 = 可见 (200)
func TestDemandDetailAuthZ_ProductManagerQueryPermGlobalVisible(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demand d[\s\S]*WHERE d\.id = \?`).
		WithArgs(uint(1004)).
		WillReturnRows(newDemandDetailMockRow(1004, 0))

	// PMO 角色查询：未命中
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_gf_user_roles ur JOIN zt_roles r ON r\.id = ur\.roleId WHERE ur\.userId = \? AND ur\.deleted = '0' AND r\.deleted = '0' AND r\.isActive = 1 AND r\.code = \?`).
		WithArgs(int64(99), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// 个人干系人检查：未命中
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_demand d`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// 部门经理检查：非部门经理
	mock.ExpectQuery(`SELECT d\.id, COALESCE\(d\.path, ''\) AS path FROM zt_dept d`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}))

	// 注入 schedule:list 权限上下文
	ctx := perm.WithGranted(context.Background(), map[string]bool{perm.ScheduleList.String(): true})
	actor := &model.User{ID: 99, Account: "demo_po", IsSuperAdmin: false}

	row, err := svc.GetDemandDetailAuthZ(ctx, actor, 1004)
	if err != nil {
		t.Fatalf("expected visible for query user, got error: %v", err)
	}
	if row == nil || row.ID != 1004 {
		t.Fatalf("expected demand 1004, got %+v", row)
	}
}

// 5. 无关普通用户 = 403
func TestDemandDetailAuthZ_NormalUserNoRelationForbidden(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	svc := NewDetailService(repo)

	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demand d[\s\S]*WHERE d\.id = \?`).
		WithArgs(uint(1005)).
		WillReturnRows(newDemandDetailMockRow(1005, 0))

	// PMO 角色查询：未命中
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_gf_user_roles ur JOIN zt_roles r ON r\.id = ur\.roleId WHERE ur\.userId = \? AND ur\.deleted = '0' AND r\.deleted = '0' AND r\.isActive = 1 AND r\.code = \?`).
		WithArgs(int64(100), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// 个人干系人检查：未命中
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_demand d`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// 部门经理检查：非部门经理
	mock.ExpectQuery(`SELECT d\.id, COALESCE\(d\.path, ''\) AS path FROM zt_dept d`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}))

	actor := &model.User{ID: 100, Account: "user_normal", IsSuperAdmin: false}
	row, err := svc.GetDemandDetailAuthZ(t.Context(), actor, 1005)
	if row != nil {
		t.Fatalf("expected nil row, got %+v", row)
	}
	biz, ok := errorx.IsBizError(err)
	if !ok || biz.Code != errorx.ErrCodeForbidden {
		t.Fatalf("expected forbidden error, got: %v", err)
	}
}

// 6. 只读角色调用写接口（业务评审接口） = 403
func TestDemandDetailAuthZ_ReadOnlyRoleWriteInterfaceForbidden(t *testing.T) {
	for _, account := range []string{"demo_leader", "demo_po", "user_pmo"} {
		t.Run(account, func(t *testing.T) {
			gormDB, mock := setupMockDB(t)
			repo := NewRepo(gormDB, gormDB)
			svc := NewService(repo, nil, nil, zap.NewNop())

			// Mock 查询需求处于 wait 状态
			mock.ExpectQuery(`SELECT id, status, deleted, createdBy, assignedTo, reviewedBy, reviewer, mailto, isNeedFocus, product FROM `+"`?zt_demand`?"+` WHERE id = \? LIMIT \?`).
				WithArgs(int64(1001), 1).
				WillReturnRows(sqlmock.NewRows([]string{
					"id", "status", "deleted", "createdBy", "assignedTo", "reviewedBy", "reviewer", "mailto", "isNeedFocus", "product",
				}).AddRow(1001, "wait", "0", "alice", "bob", "bob", "bob", "", 0, 1))

			// Mock 查询当前账号在 zt_demandreview 中不存在评审记录 (found = false)
			mock.ExpectQuery(`SELECT[\s\S]*FROM[\s\S]*zt_demandreview[\s\S]*WHERE demand = \? AND reviewer = \? LIMIT \?`).
				WithArgs(int64(1001), account, 1).
				WillReturnRows(sqlmock.NewRows([]string{"result"}))

			actor := &model.User{ID: 10, Account: account}
			// 团队长调用业务评审接口：非指定评审人，返回 403 Forbidden
			_, err := svc.ReviewDemand(perm.WithGranted(context.Background(), map[string]bool{perm.ScheduleList.String(): true}), actor, ReviewDemandReq{ID: 1001, Result: "pass"})
			if err == nil {
				t.Fatal("expected error for write action by non-assigned leader, got nil")
			}
			biz, ok := errorx.IsBizError(err)
			if !ok || biz.Code != errorx.ErrCodeForbidden {
				t.Fatalf("expected ErrCodeForbidden, got: %v", err)
			}
		})
	}
}

// 7. DemandDetailView 渲染标准 HTML 403 页面
func TestDemandDetailView_LeaderWithoutQueryPerm403RendersHTMLErrorPage(t *testing.T) {
	initTestRenderer(t)
	gin.SetMode(gin.TestMode)

	gormDB, mock := setupMockDB(t)
	repo := NewDemandDetailRepo(gormDB)
	detailSvc := NewDetailService(repo)

	poRepo := NewRepo(gormDB, gormDB)
	poSvc := NewService(poRepo, nil, nil, zap.NewNop())
	handler := &Handler{
		svc:       poSvc,
		detailSvc: detailSvc,
		logger:    zap.NewNop(),
	}

	// Mock demand 存在但无访问权限
	mock.ExpectQuery(`SELECT[\s\S]*FROM zt_demand d[\s\S]*WHERE d\.id = \?`).
		WithArgs(uint(2001)).
		WillReturnRows(newDemandDetailMockRow(2001, 0))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_gf_user_roles ur JOIN zt_roles r ON r\.id = ur\.roleId WHERE ur\.userId = \? AND ur\.deleted = '0' AND r\.deleted = '0' AND r\.isActive = 1 AND r\.code = \?`).
		WithArgs(int64(200), workbenchroles.RolePMO).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_demand d`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT d\.id, COALESCE\(d\.path, ''\) AS path FROM zt_dept d`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "path"}).AddRow(14, ",1,14,"))
	mock.ExpectQuery(`SELECT DISTINCT id FROM zt_dept WHERE id IN \(\?\) OR path LIKE \?`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(14))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_demand d`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	actor := &model.User{ID: 200, Account: "demo_leader", IsSuperAdmin: false}
	c.Set("currentUser", actor)
	req, _ := http.NewRequest(http.MethodGet, "/demands/2001", nil)
	c.Request = req
	c.Params = gin.Params{gin.Param{Key: "id", Value: "2001"}}

	handler.DemandDetailView(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected HTTP 403, got %d", w.Code)
	}
	body := w.Body.String()
	// 验证渲染的是 HTML 页面而非裸 JSON
	if !strings.Contains(body, "<!DOCTYPE html>") && !strings.Contains(body, "<html") {
		t.Fatalf("expected HTML response, got:\n%s", body)
	}
	if !strings.Contains(body, "无权查看该业务需求") {
		t.Fatalf("expected '无权查看该业务需求' in error page, got:\n%s", body)
	}
}

// 8. 逾期天数脏数据兜底测试（0000-00-00, 0001-01-01, < 2000-01-01）
func TestCalcOverdue_DirtyDataFallback(t *testing.T) {
	dirtyDates := []string{
		"",
		" ",
		"0000-00-00",
		"0000-00-00 00:00:00",
		"0001-01-01",
		"0001-01-01T00:00:00Z",
		"1970-01-01",
		"1999-12-31",
	}
	for _, raw := range dirtyDates {
		deadline, overdue, days := calcOverdue(raw)
		if deadline != "" || overdue || days != 0 {
			t.Errorf("calcOverdue(%q) = (%q, %v, %d); want ('', false, 0)", raw, deadline, overdue, days)
		}
	}

	// 正常未来日期：非逾期
	future := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	d, overdue, days := calcOverdue(future)
	if overdue || days != 0 || d != future {
		t.Errorf("calcOverdue(%q) = (%q, %v, %d); want (%q, false, 0)", future, d, overdue, days, future)
	}

	// 正常历史日期：逾期
	past := "2023-01-01"
	d2, overdue2, days2 := calcOverdue(past)
	if !overdue2 || days2 <= 0 || d2 != past {
		t.Errorf("calcOverdue(%q) = (%q, %v, %d); want overdue=true, days > 0", past, d2, overdue2, days2)
	}
}
