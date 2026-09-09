package schedule

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetDemandScheduling_DirectBrowserRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{}

	r := gin.New()
	r.GET("/schedule/demands/:id/scheduling", h.GetDemandScheduling)

	// Case 1: Browser direct navigation (Accept: text/html)
	req := httptest.NewRequest(http.MethodGet, "/schedule/demands/63406/scheduling", nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected status 302 Found, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "/schedule?openDemand=63406" {
		t.Fatalf("expected redirect to /schedule?openDemand=63406, got %s", loc)
	}
}

func TestGetStoryScheduling_DirectBrowserRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{}

	r := gin.New()
	r.GET("/schedule/stories/:id/scheduling", h.GetStoryScheduling)

	// Case 1: Browser direct navigation (Accept: text/html)
	req := httptest.NewRequest(http.MethodGet, "/schedule/stories/12345/scheduling", nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected status 302 Found, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "/schedule?openStory=12345" {
		t.Fatalf("expected redirect to /schedule?openStory=12345, got %s", loc)
	}
}
