package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"workbench/internal/model"
	"workbench/internal/pkg/perm"
)

func setupTestRouter(p perm.Permission, user *model.User, perms map[string]bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set("currentUser", user)
		}
		if perms != nil {
			c.Set("userPerms", perms)
		}
		c.Next()
	})
	r.GET("/protected", RequirePerm(p), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    "ok_payload",
		})
	})
	return r
}

func TestRequirePerm_PermissionJSONDenied(t *testing.T) {
	user := &model.User{ID: 1, Account: "user1", IsSuperAdmin: false}
	perms := map[string]bool{}
	r := setupTestRouter(perm.Permission("test:action"), user, perms)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}

	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v, body: %s", err, rr.Body.String())
	}
	if resp["success"] != false {
		t.Fatalf("success = %v, want false", resp["success"])
	}
	if resp["error"] != "无权限访问" {
		t.Fatalf("error = %v, want '无权限访问'", resp["error"])
	}
}

func TestRequirePerm_HTMLDenied(t *testing.T) {
	user := &model.User{ID: 1, Account: "user1", IsSuperAdmin: false}
	perms := map[string]bool{}
	r := setupTestRouter(perm.Permission("test:action"), user, perms)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Accept", "text/html")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}

	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}

	if !strings.Contains(rr.Body.String(), "无权限访问") {
		t.Fatalf("body missing '无权限访问': %s", rr.Body.String())
	}
}

func TestRequirePerm_AllowedResponsePreserved(t *testing.T) {
	user := &model.User{ID: 1, Account: "user1", IsSuperAdmin: false}
	perms := map[string]bool{"test:action": true}
	r := setupTestRouter(perm.Permission("test:action"), user, perms)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if resp["success"] != true || resp["data"] != "ok_payload" {
		t.Fatalf("unexpected response payload: %v", resp)
	}
}

func TestRequirePerm_SuperAdminAllowed(t *testing.T) {
	user := &model.User{ID: 99, Account: "admin", IsSuperAdmin: true}
	r := setupTestRouter(perm.Permission("test:action"), user, nil)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}
