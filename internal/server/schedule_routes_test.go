package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"workbench/internal/middleware"
	"workbench/internal/model"
	"workbench/internal/module/schedule"
	"workbench/internal/pkg/perm"
)

type scheduleRouteCase struct {
	name         string
	method       string
	path         string
	requiredPerm perm.Permission
	otherPerm    perm.Permission
	isHTML       bool
	body         string
}

var scheduleTestRoutes = []scheduleRouteCase{
	{name: "Index", method: http.MethodGet, path: "/schedule", requiredPerm: perm.ScheduleList, otherPerm: perm.ScheduleUpdate, isHTML: true},
	{name: "MatchingPlans", method: http.MethodGet, path: "/schedule/matching-plans?product_id=1&end_date=2026-12-31", requiredPerm: perm.ScheduleList, otherPerm: perm.ScheduleUpdate},
	{name: "DemandScheduling", method: http.MethodGet, path: "/schedule/demands/1/scheduling", requiredPerm: perm.ScheduleList, otherPerm: perm.ScheduleUpdate},
	{name: "SaveDemandScheduling", method: http.MethodPost, path: "/schedule/demands/1/save-scheduling", requiredPerm: perm.ScheduleUpdate, otherPerm: perm.ScheduleList, body: "{}"},
	{name: "StoryScheduling", method: http.MethodGet, path: "/schedule/stories/1/scheduling", requiredPerm: perm.ScheduleList, otherPerm: perm.ScheduleUpdate},
	{name: "SaveStoryScheduling", method: http.MethodPost, path: "/schedule/stories/1/save-scheduling", requiredPerm: perm.ScheduleUpdate, otherPerm: perm.ScheduleList, body: "{}"},
	{name: "ProductProjects", method: http.MethodGet, path: "/schedule/products/1/projects", requiredPerm: perm.ScheduleList, otherPerm: perm.ScheduleUpdate},
	{name: "ProjectExecutions", method: http.MethodGet, path: "/schedule/projects/1/executions", requiredPerm: perm.ScheduleList, otherPerm: perm.ScheduleUpdate},
	{name: "StoryTasks", method: http.MethodGet, path: "/schedule/stories/1/tasks", requiredPerm: perm.ScheduleList, otherPerm: perm.ScheduleUpdate},
	{name: "SaveStoryTasks", method: http.MethodPost, path: "/schedule/stories/1/save-tasks", requiredPerm: perm.ScheduleUpdate, otherPerm: perm.ScheduleList, body: "{}"},
	{name: "CreateWindow", method: http.MethodPost, path: "/schedule/windows", requiredPerm: perm.ScheduleCreate, otherPerm: perm.ScheduleList, body: "{}"},
	{name: "ListWindows", method: http.MethodGet, path: "/schedule/windows", requiredPerm: perm.ScheduleList, otherPerm: perm.ScheduleUpdate},
	{name: "GetWindow", method: http.MethodGet, path: "/schedule/windows/1", requiredPerm: perm.ScheduleList, otherPerm: perm.ScheduleUpdate},
	{name: "UpdateWindow", method: http.MethodPut, path: "/schedule/windows/1", requiredPerm: perm.ScheduleUpdate, otherPerm: perm.ScheduleList, body: "{}"},
	{name: "DeleteWindow", method: http.MethodDelete, path: "/schedule/windows/1", requiredPerm: perm.ScheduleDelete, otherPerm: perm.ScheduleList},
}

func newScheduleTestRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()

	_, source, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(source), "../..")
	if tmpl, err := loadTemplates(filepath.Join(repoRoot, "web/templates")); err == nil {
		r.SetHTMLTemplate(tmpl)
	}

	po := r.Group("")
	po.Use(func(c *gin.Context) {
		role := c.GetHeader("X-Test-Role")
		if role == "" {
			if strings.Contains(c.GetHeader("Accept"), "application/json") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "error": "未登录或会话已过期"})
				return
			}
			c.Redirect(http.StatusSeeOther, "/login?redirect="+url.QueryEscape(c.Request.URL.RequestURI()))
			c.Abort()
			return
		}
		if role == "superadmin" {
			c.Set("currentUser", &model.User{ID: 1, Account: "superadmin", IsSuperAdmin: true})
			c.Next()
			return
		}
		c.Set("currentUser", &model.User{ID: 2, Account: "regular_user", IsSuperAdmin: false})
		perms := make(map[string]bool)
		if permsHeader := c.GetHeader("X-Test-Perms"); permsHeader != "" {
			for _, p := range strings.Split(permsHeader, ",") {
				if p = strings.TrimSpace(p); p != "" {
					perms[p] = true
				}
			}
		}
		c.Set("userPerms", perms)
		c.Next()
	})

	repo := schedule.NewRepo(db)
	svc := schedule.NewService(repo, zap.NewNop())
	h := schedule.NewHandler(nil, zap.NewNop(), svc, "")
	h.RegisterRoutes(po)
	return r
}

func TestScheduleRoutes(t *testing.T) {
	t.Run("ScheduleRouteMatrix", func(t *testing.T) { testScheduleRouteMatrix(t) })
	t.Run("NoListImplied", func(t *testing.T) { testNoListImplied(t) })
	t.Run("DownstreamNotCalled", func(t *testing.T) { testDownstreamNotCalled(t) })
}

func TestScheduleRouteMatrix(t *testing.T) { testScheduleRouteMatrix(t) }
func TestNoListImplied(t *testing.T)       { testNoListImplied(t) }
func TestDownstreamNotCalled(t *testing.T) { testDownstreamNotCalled(t) }

func testScheduleRouteMatrix(t *testing.T) {
	t.Helper()
	sqlDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer sqlDB.Close()

	gdb, _ := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	router := newScheduleTestRouter(t, gdb)

	scenarios := []struct {
		name       string
		role       string
		permKind   string
		wantReject bool
	}{
		{name: "Anonymous", role: "", permKind: "", wantReject: true},
		{name: "NoCap", role: "user", permKind: "none", wantReject: true},
		{name: "OtherCap", role: "user", permKind: "other", wantReject: true},
		{name: "CorrectCap", role: "user", permKind: "correct", wantReject: false},
		{name: "SuperAdmin", role: "superadmin", permKind: "", wantReject: false},
	}

	for _, route := range scheduleTestRoutes {
		route := route
		for _, sc := range scenarios {
			sc := sc
			t.Run(route.name+"_"+sc.name, func(t *testing.T) {
				req := httptest.NewRequest(route.method, route.path, bytes.NewBufferString(route.body))
				if route.isHTML {
					req.Header.Set("Accept", "text/html")
				} else {
					req.Header.Set("Accept", "application/json")
					req.Header.Set("Content-Type", "application/json")
				}

				if sc.role != "" {
					req.Header.Set("X-Test-Role", sc.role)
					if sc.permKind == "other" {
						req.Header.Set("X-Test-Perms", route.otherPerm.String())
					} else if sc.permKind == "correct" {
						req.Header.Set("X-Test-Perms", route.requiredPerm.String())
					}
				}

				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)

				if sc.role == "" {
					if route.isHTML {
						if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "/login") {
							t.Fatalf("expected 303 redirect to login, got code=%d loc=%s", rec.Code, rec.Header().Get("Location"))
						}
					} else {
						if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "未登录或会话已过期") {
							t.Fatalf("expected 401 auth error, got code=%d body=%s", rec.Code, rec.Body.String())
						}
					}
					return
				}

				if sc.wantReject {
					if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "无权限访问") {
						t.Fatalf("expected 403 无权限访问, got code=%d body=%s", rec.Code, rec.Body.String())
					}
					return
				}

				if rec.Code == http.StatusForbidden || strings.Contains(rec.Body.String(), "无权限访问") {
					t.Fatalf("expected downstream access, got forbidden: code=%d body=%s", rec.Code, rec.Body.String())
				}
			})
		}
	}
}

func testNoListImplied(t *testing.T) {
	t.Helper()
	sqlDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer sqlDB.Close()

	gdb, _ := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	router := newScheduleTestRouter(t, gdb)

	req := httptest.NewRequest(http.MethodGet, "/schedule/windows", nil)
	req.Header.Set("X-Test-Role", "user")
	req.Header.Set("X-Test-Perms", perm.ScheduleUpdate.String())
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "无权限访问") {
		t.Fatalf("expected 403 无权限访问 when holding only schedule:update, got code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func testDownstreamNotCalled(t *testing.T) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer sqlDB.Close()

	gdb, _ := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	router := newScheduleTestRouter(t, gdb)

	req := httptest.NewRequest(http.MethodPost, "/schedule/stories/1/save-tasks", bytes.NewBufferString(`{"tasks":[{"name":"task"}]}`))
	req.Header.Set("X-Test-Role", "user")
	req.Header.Set("X-Test-Perms", perm.ScheduleList.String())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "无权限访问") {
		t.Fatalf("expected 403 无权限访问, got code=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected no SQL executed, got: %v", err)
	}
}

func TestScheduleRoutes_SuperAdminObjectAuthEnforced(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer sqlDB.Close()

	gdb, _ := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	router := newScheduleTestRouter(t, gdb)

	rows := sqlmock.NewRows([]string{"id", "name", "releaseDate", "createdBy", "deletedAt"}).
		AddRow(1, "26-0701窗口", time.Now(), "other_user", nil)
	mock.ExpectQuery("SELECT \\* FROM `zt_versionwindow` WHERE").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodDelete, "/schedule/windows/1", nil)
	req.Header.Set("X-Test-Role", "superadmin")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusForbidden {
		t.Fatalf("superadmin capability check should not return 403, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "只有创建人可以删除") {
		t.Fatalf("expected service object auth rejection, got body: %s", rec.Body.String())
	}
}

func TestScheduleRoutes_RealRequireLoginChain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mgr := scs.New()
	r := gin.New()
	po := r.Group("")
	po.Use(middleware.RequireLogin(mgr, nil))

	sqlDB, _, _ := sqlmock.New()
	defer sqlDB.Close()
	gdb, _ := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	repo := schedule.NewRepo(gdb)
	svc := schedule.NewService(repo, zap.NewNop())
	h := schedule.NewHandler(nil, zap.NewNop(), svc, "")
	h.RegisterRoutes(po)

	handler := mgr.LoadAndSave(r)

	req := httptest.NewRequest(http.MethodGet, "/schedule", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/schedule/windows", nil)
	req.Header.Set("Accept", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 unauthorized, got %d", rec.Code)
	}
}
