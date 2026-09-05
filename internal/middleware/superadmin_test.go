package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"workbench/internal/model"
)

func setupSuperAdminTestRouter(user *model.User) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set("currentUser", user)
		}
		c.Next()
	})
	r.GET("/debug-test", RequireSuperAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    "superadmin_data",
		})
	})
	return r
}

func TestRequireSuperAdmin_DebugAnonymous(t *testing.T) {
	r := setupSuperAdminTestRouter(nil)

	// JSON request
	reqJSON := httptest.NewRequest(http.MethodGet, "/debug-test", nil)
	reqJSON.Header.Set("Accept", "application/json")
	rrJSON := httptest.NewRecorder()
	r.ServeHTTP(rrJSON, reqJSON)

	if rrJSON.Code != http.StatusUnauthorized {
		t.Fatalf("anon JSON: status = %d, want %d", rrJSON.Code, http.StatusUnauthorized)
	}

	// HTML request
	reqHTML := httptest.NewRequest(http.MethodGet, "/debug-test", nil)
	reqHTML.Header.Set("Accept", "text/html")
	rrHTML := httptest.NewRecorder()
	r.ServeHTTP(rrHTML, reqHTML)

	if rrHTML.Code != http.StatusSeeOther {
		t.Fatalf("anon HTML: status = %d, want %d", rrHTML.Code, http.StatusSeeOther)
	}
}

func TestRequireSuperAdmin_DebugRegularDenied(t *testing.T) {
	regularUser := &model.User{ID: 2, Account: "regular_user", IsSuperAdmin: false}
	r := setupSuperAdminTestRouter(regularUser)

	// JSON request
	reqJSON := httptest.NewRequest(http.MethodGet, "/debug-test", nil)
	reqJSON.Header.Set("Accept", "application/json")
	rrJSON := httptest.NewRecorder()
	r.ServeHTTP(rrJSON, reqJSON)

	if rrJSON.Code != http.StatusForbidden {
		t.Fatalf("regular user JSON: status = %d, want %d", rrJSON.Code, http.StatusForbidden)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rrJSON.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if resp["success"] != false || resp["error"] != "无权限访问" {
		t.Fatalf("unexpected JSON response: %v", resp)
	}

	// HTML request
	reqHTML := httptest.NewRequest(http.MethodGet, "/debug-test", nil)
	reqHTML.Header.Set("Accept", "text/html")
	rrHTML := httptest.NewRecorder()
	r.ServeHTTP(rrHTML, reqHTML)

	if rrHTML.Code != http.StatusForbidden {
		t.Fatalf("regular user HTML: status = %d, want %d", rrHTML.Code, http.StatusForbidden)
	}
	if !strings.Contains(rrHTML.Body.String(), "无权限访问") {
		t.Fatalf("body missing '无权限访问': %s", rrHTML.Body.String())
	}
}

func TestRequireSuperAdmin_DebugSuperAllowed(t *testing.T) {
	superUser := &model.User{ID: 1, Account: "superadmin_user", IsSuperAdmin: true}
	r := setupSuperAdminTestRouter(superUser)

	req := httptest.NewRequest(http.MethodGet, "/debug-test", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("superadmin status = %d, want %d", rr.Code, http.StatusOK)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if resp["success"] != true || resp["data"] != "superadmin_data" {
		t.Fatalf("unexpected response: %v", resp)
	}
}

func TestRequireSuperAdmin_DebugForgedFlag(t *testing.T) {
	regularUser := &model.User{ID: 3, Account: "attacker", IsSuperAdmin: false}
	r := setupSuperAdminTestRouter(regularUser)

	req := httptest.NewRequest(http.MethodGet, "/debug-test?superadmin=true&is_super_admin=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("forged flag status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}
