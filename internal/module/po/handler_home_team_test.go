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

func TestTeamHomeVersionWindowsRejectsUnauthorizedScopeBeforeLoadingData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(&Service{}, zap.NewNop())
	h.SetTeamViewAccess(func(context.Context, *model.User) (bool, error) { return true, nil })
	var gotScope string
	var gotID uint
	h.SetTeamScopeGroupIDs(func(_ context.Context, _ *model.User, scope string, scopeID uint) ([]uint, error) {
		gotScope, gotID = scope, scopeID
		return nil, errorx.New(errorx.ErrCodeForbidden, "所选团队不在你的授权范围内")
	})
	router := gin.New()
	router.GET("/home/team/version-windows", func(c *gin.Context) {
		c.Set("currentUser", &model.User{Account: "coach1"})
		c.Next()
	}, h.TeamHomeVersionWindows)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/home/team/version-windows?scope=team&scopeId=99", nil))
	if rec.Code != http.StatusForbidden || gotScope != "team" || gotID != 99 {
		t.Fatalf("unauthorized version-window scope status/scope = %d/%s/%d", rec.Code, gotScope, gotID)
	}
}
