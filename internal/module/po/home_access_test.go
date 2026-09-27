package po

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"workbench/internal/model"
)

func TestHomePageAccessAllowsTeamManagersOnlyInTeamView(t *testing.T) {
	gin.SetMode(gin.TestMode)

	called := 0
	h := &Handler{teamViewAccess: func(context.Context, *model.User) (bool, error) {
		called++
		return true, nil
	}}
	r := gin.New()
	r.GET("/home", setHomeTestUser(nil), h.homePageAccess(), func(c *gin.Context) {
		teamOnly, _ := c.Get("teamHomeOnly")
		if teamOnly != true {
			c.Status(http.StatusNoContent)
			return
		}
		c.Status(http.StatusAccepted)
	})

	teamReq := httptest.NewRequest(http.MethodGet, "/home?view=team", nil)
	teamReq.Header.Set("Accept", "text/html")
	teamRec := httptest.NewRecorder()
	r.ServeHTTP(teamRec, teamReq)
	if teamRec.Code != http.StatusAccepted || called != 1 {
		t.Fatalf("authorized team view status/calls = %d/%d, want 202/1", teamRec.Code, called)
	}

	demandReq := httptest.NewRequest(http.MethodGet, "/home?view=demand", nil)
	demandReq.Header.Set("Accept", "text/html")
	demandRec := httptest.NewRecorder()
	r.ServeHTTP(demandRec, demandReq)
	if demandRec.Code != http.StatusForbidden || called != 1 {
		t.Fatalf("team-only demand view status/calls = %d/%d, want 403/1", demandRec.Code, called)
	}
}

func TestHomePageAccessKeepsDemandPermissionAndRequiresTeamScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := 0
	h := &Handler{teamViewAccess: func(context.Context, *model.User) (bool, error) {
		called++
		return false, nil
	}}
	r := gin.New()
	r.GET("/home", setHomeTestUser(map[string]bool{"po:home": true}), h.homePageAccess(), func(c *gin.Context) { c.Status(http.StatusNoContent) })

	poReq := httptest.NewRequest(http.MethodGet, "/home", nil)
	poRec := httptest.NewRecorder()
	r.ServeHTTP(poRec, poReq)
	if poRec.Code != http.StatusNoContent {
		t.Fatalf("PO home permission status = %d, want 204", poRec.Code)
	}
	if called != 0 {
		t.Fatalf("team access callback called for demand request: %d", called)
	}
}

func setHomeTestUser(perms map[string]bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("currentUser", &model.User{Account: "user1"})
		if perms != nil {
			c.Set("userPerms", perms)
		}
		c.Next()
	}
}
