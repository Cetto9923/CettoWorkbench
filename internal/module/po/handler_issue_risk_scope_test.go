package po

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
)

func TestIssueRiskTeamScopeRequiresLeadAccessAndKeepsScopeID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(nil, zap.NewNop())
	var gotScope string
	var gotID uint
	h.SetTeamViewAccess(func(context.Context, *model.User) (bool, error) { return true, nil })
	h.SetTeamScopeAccounts(func(_ context.Context, _ *model.User, scope string, scopeID uint) ([]string, error) {
		gotScope, gotID = scope, scopeID
		return []string{"dev1"}, nil
	})
	router := gin.New()
	router.GET("/issues/risk", func(c *gin.Context) {
		c.Set("currentUser", &model.User{Account: "coach1"})
		c.Next()
	}, h.issueRiskPageAccess(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/issues/risk?scope=team&scopeId=11", nil))
	if resp.Code != http.StatusNoContent || gotScope != "team" || gotID != 11 {
		t.Fatalf("team route status/scope = %d/%s/%d", resp.Code, gotScope, gotID)
	}
}

func TestIssueRiskTeamScopeRejectsUnauthorizedSelection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(nil, zap.NewNop())
	h.SetTeamViewAccess(func(context.Context, *model.User) (bool, error) { return true, nil })
	h.SetTeamScopeAccounts(func(context.Context, *model.User, string, uint) ([]string, error) {
		return nil, errorx.New(errorx.ErrCodeForbidden, "所选团队不在你的授权范围内")
	})
	router := gin.New()
	router.GET("/issues/risk", func(c *gin.Context) {
		c.Set("currentUser", &model.User{Account: "coach1"})
		c.Next()
	}, h.issueRiskPageAccess(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/issues/risk?scope=team&scopeId=1", nil))
	if resp.Code != http.StatusForbidden {
		t.Fatalf("unauthorized team selection status = %d, want 403", resp.Code)
	}
}
