package po

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"workbench/internal/model"
)

func TestUrgeHomeDemandUnknownTypeReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(NewService(NewRepo(nil, nil), nil, nil), nil)
	router := gin.New()
	router.POST("/demands/:id/urge", func(c *gin.Context) {
		c.Set("currentUser", &model.User{ID: 1, Account: "admin"})
		h.UrgeHomeDemand(c)
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/demands/999999/urge", strings.NewReader(`{"urgeType":"zzz","channels":["inapp"]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusInternalServerError {
		t.Fatalf("illegal urgeType must not be 500; body = %s", rec.Body.String())
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v; raw = %s", err, rec.Body.String())
	}
	if body.Success || body.Message != "催办类型无效" {
		t.Fatalf("body = %s", rec.Body.String())
	}
}
