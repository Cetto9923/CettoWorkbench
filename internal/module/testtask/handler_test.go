package testtask

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"workbench/internal/model"
)

func setupMockTesttaskDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	dialector := mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}
	return db, mock
}

func TestGetContext_Success(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil, nil, nil)

	mock.ExpectQuery("(?s)SELECT d\\.id, d\\.name.*FROM zt_demand AS d.*WHERE d\\.id = \\? AND d\\.deleted = '0' LIMIT \\?").
		WithArgs(63411, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status", "stage", "BRA", "RD", "QD", "product_name", "estimate_launch"}).
			AddRow(63411, "测试需求", "developing", "wait", "003030", "771349", "004481", "主系统A", "2026-10-01"))
	mock.ExpectQuery("(?s)SELECT DISTINCT p\\.id, p\\.name.*FROM zt_demandclarify").WithArgs(63411).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))
	mock.ExpectQuery("(?s)SELECT account, realname.*FROM zt_user").
		WillReturnRows(sqlmock.NewRows([]string{"account", "realname"}))

	resp, err := svc.GetContext(context.Background(), &model.User{Account: "admin", DisplayName: "管理员"}, 63411)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.DemandID != 63411 {
		t.Errorf("expected demandID 63411, got %d", resp.DemandID)
	}
	if resp.Title != "测试需求" {
		t.Errorf("expected title '测试需求', got %s", resp.Title)
	}
	if resp.RawStatus != "开发中" {
		t.Errorf("expected RawStatus '开发中', got %s", resp.RawStatus)
	}
	if resp.MainSystemName != "主系统A" {
		t.Errorf("expected MainSystemName '主系统A', got %s", resp.MainSystemName)
	}
}

func TestGetContext_DemandNotFound(t *testing.T) {
	db, mock := setupMockTesttaskDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil, nil, nil)

	mock.ExpectQuery("(?s)SELECT d\\.id, d\\.name.*FROM zt_demand AS d.*WHERE d\\.id = \\? AND d\\.deleted = '0' LIMIT \\?").
		WithArgs(99999, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status", "stage", "BRA", "RD", "QD", "product_name", "estimate_launch"}))

	_, err := svc.GetContext(context.Background(), nil, 99999)
	if err == nil {
		t.Fatal("expected error for not found demand, got nil")
	}
}

func TestHandler_GetContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock := setupMockTesttaskDB(t)
	repo := NewRepo(db)
	svc := NewService(repo, nil, nil, nil)
	h := NewHandler(svc, nil)

	r := gin.New()
	h.RegisterRoutes(r.Group(""))

	mock.ExpectQuery("(?s)SELECT d\\.id, d\\.name.*FROM zt_demand AS d.*WHERE d\\.id = \\? AND d\\.deleted = '0' LIMIT \\?").
		WithArgs(63411, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "status", "stage", "BRA", "RD", "QD", "product_name", "estimate_launch"}).
			AddRow(63411, "测试需求", "developing", "wait", "003030", "771349", "004481", "系统2", "2026-10-01"))
	mock.ExpectQuery("(?s)SELECT DISTINCT p\\.id, p\\.name.*FROM zt_demandclarify").WithArgs(63411).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))
	mock.ExpectQuery("(?s)SELECT account, realname.*FROM zt_user").
		WillReturnRows(sqlmock.NewRows([]string{"account", "realname"}))

	req := httptest.NewRequest(http.MethodGet, "/demands/63411/testtask", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
