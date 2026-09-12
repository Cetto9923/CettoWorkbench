// =============================================================================
// 文件: internal/module/po/home_test.go
// 模块: PO 工作台
// 类型: action
// 职责: 固化首页阶段与前端真实性契约。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"workbench/internal/config"
	"workbench/internal/model"
	"workbench/internal/pkg/render"
)

func TestValueStreamStagesRemainOrdered(t *testing.T) {
	want := []string{
		"all",
		"accept",
		"clarify",
		"schedule",
		"developing",
		"testing",
		"waitacceptance",
		"acceptanced",
		"publish",
		"released",
	}

	got := make([]string, 0, len(valueStreamStages))
	for _, stage := range valueStreamStages {
		got = append(got, stage.status)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("value stream stage order = %v, want %v", got, want)
	}
}

func TestDemandsReqValidate(t *testing.T) {
	req := DemandsReq{Status: " schedule "}
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("valid status returned errors: %v", errs)
	}
	if req.Status != "schedule" {
		t.Fatalf("trimmed status = %q, want schedule", req.Status)
	}
	if req.Page != 1 || req.PageSize != 15 {
		t.Fatalf("default pagination: got page=%d pageSize=%d, want page=1 pageSize=15", req.Page, req.PageSize)
	}

	reqCustom := DemandsReq{Status: "all", Page: 3, PageSize: 50}
	if errs := reqCustom.Validate(); len(errs) != 0 {
		t.Fatalf("valid custom pagination returned errors: %v", errs)
	}
	if reqCustom.Page != 3 || reqCustom.PageSize != 50 {
		t.Fatalf("custom pagination: got page=%d pageSize=%d, want 3, 50", reqCustom.Page, reqCustom.PageSize)
	}

	reqClamp := DemandsReq{Status: "all", Page: -2, PageSize: 500}
	if errs := reqClamp.Validate(); len(errs) != 0 {
		t.Fatalf("clamp pagination returned errors: %v", errs)
	}
	if reqClamp.Page != 1 || reqClamp.PageSize != 100 {
		t.Fatalf("clamped pagination: got page=%d pageSize=%d, want 1, 100", reqClamp.Page, reqClamp.PageSize)
	}

	invalid := DemandsReq{Status: "unknown"}
	if errs := invalid.Validate(); len(errs) != 1 || errs[0].Field != "status" {
		t.Fatalf("invalid status errors = %v", errs)
	}
}

func TestBuildDemandWorkItemCarriesCurrentRiskFacts(t *testing.T) {
	item := buildDemandWorkItem(DemandRow{
		ID:     2511,
		Pri:    "1",
		Name:   "挂起且驳回的需求",
		Status: "refuse",
		Hang:   "1",
	}, "澄清", nil)

	if !item.Suspended {
		t.Fatal("hang='1' must render as a current suspension")
	}
	if !item.Blocked {
		t.Fatal("status='refuse' must render as a current blocker")
	}
	if item.Pri != "P1" {
		t.Fatalf("priority = %q, want P1", item.Pri)
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "web", "templates")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("cannot find repo root containing web/templates")
	return ""
}

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	dialector := mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening gorm database", err)
	}
	return gormDB, mock
}

func initTestRenderer(t *testing.T) {
	t.Helper()
	root := findRepoRoot(t)
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}
	cfg := &config.Config{
		App:    config.App{Name: "Workbench", Env: "dev"},
		Layout: config.Layout{Nav: "sidebar"},
	}
	r, err := render.New(cfg, true)
	if err != nil {
		t.Fatalf("init test renderer: %v", err)
	}
	render.SetDefault(r)
}

func TestEmptyValueStreamStages(t *testing.T) {
	stages := emptyValueStreamStages()
	if len(stages) != len(valueStreamStages) {
		t.Fatalf("emptyValueStreamStages length = %d, want %d", len(stages), len(valueStreamStages))
	}
	for i, stage := range stages {
		if stage.Valid {
			t.Errorf("stage[%d] (%s) Valid should be false in empty stages", i, stage.Status)
		}
		if stage.Count != 0 || stage.DemandCount != 0 || stage.StoryCount != 0 {
			t.Errorf("stage[%d] counts should be 0, got %#v", i, stage)
		}
		if stage.Status != valueStreamStages[i].status {
			t.Errorf("stage[%d].Status = %q, want %q", i, stage.Status, valueStreamStages[i].status)
		}
	}
}

func TestServiceHome_RepoNilReturnsError(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)
	resp, err := svc.Home(context.Background(), &model.User{Account: "alice"})
	if err == nil {
		t.Fatal("expected error when repo is nil, got nil")
	}
	if resp != nil {
		t.Fatalf("expected nil resp on error, got %#v", resp)
	}
}

func TestServiceHome_DBErrorPropagated(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	repo := NewRepo(gormDB, gormDB)
	svc := NewService(repo, nil, nil, nil)

	mock.ExpectQuery(".+").WillReturnError(errors.New("db disconnect"))

	resp, err := svc.Home(context.Background(), &model.User{Account: "alice"})
	if err == nil {
		t.Fatal("expected DB error to be returned, got nil")
	}
	if resp != nil {
		t.Fatalf("expected nil resp on DB error, got %#v", resp)
	}
}

func TestHomeHandler_ServiceErrorRendersPageError(t *testing.T) {
	initTestRenderer(t)
	gin.SetMode(gin.TestMode)

	svc := NewService(nil, nil, nil, zap.NewNop())
	h := NewHandler(svc, zap.NewNop())

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/home", nil)
	c.Request = req

	h.Home(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	// 验证错误警示信息已渲染
	if !strings.Contains(body, "数据统计暂不可用，统计数据暂不可用") {
		t.Fatalf("expected PageError alert in response body, got:\n%s", body)
	}
	// 验证阶段数字为“暂不可用”，绝非显示为 0 (ERROR ≠ ZERO)
	if !strings.Contains(body, "暂不可用") {
		t.Fatalf("expected '暂不可用' in stage counts, got:\n%s", body)
	}
	// 验证工作视角 Perspective Tabs 已渲染且顶部全局切换器 poRoleSwitcher 已移除
	if !strings.Contains(body, "po-perspective-tabs") {
		t.Fatalf("expected po-perspective-tabs in response body, got:\n%s", body)
	}
	if strings.Contains(body, "poRoleSwitcher") {
		t.Fatalf("expected poRoleSwitcher to be removed from response body, got:\n%s", body)
	}
	if !strings.Contains(body, "个人工作") || !strings.Contains(body, "需求交付") || !strings.Contains(body, "治理分析") {
		t.Fatalf("expected updated sidebar sections in response body, got:\n%s", body)
	}
}
