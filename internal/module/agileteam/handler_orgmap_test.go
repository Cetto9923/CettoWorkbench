package agileteam

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"

	"workbench/internal/model"
	"workbench/internal/pkg/perm"
)

func TestUpdateOrgTeamMappingMissingTeamReturns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, mock := newTestService(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM zt_teamgroup WHERE id = \\? AND deleted = '0' FOR UPDATE").
		WithArgs(uint(999999)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectRollback()

	h := NewHandler(svc, nil)
	router := gin.New()
	router.PUT("/workbench/api/agile-teams/:id/org-team", func(c *gin.Context) {
		c.Set("currentUser", &model.User{Account: "003030"})
		c.Set("userPerms", map[string]bool{perm.AgileTeamConfirm.String(): true})
		h.UpdateOrgTeamMapping(c)
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/workbench/api/agile-teams/999999/org-team", strings.NewReader(`{"deptId":1}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusInternalServerError {
		t.Fatalf("missing team must not be 500; body = %s", rec.Body.String())
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v; raw = %s", err, rec.Body.String())
	}
	if body.Success || body.Message != "敏捷小组不存在" {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("missing team must not write: %v", err)
	}
}
