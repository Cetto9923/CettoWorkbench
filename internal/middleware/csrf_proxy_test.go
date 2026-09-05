package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/justinas/nosurf"
)

func TestCSRFProxyHeadersCannotDowngradeOriginCheck(t *testing.T) {
	var token string
	handler := CSRF(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token = nosurf.Token(r)
		w.WriteHeader(http.StatusOK)
	}))
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "http://workbench.test/", nil))
	for _, referer := range []string{"https://workbench.test/form", "https://evil.test/form", "http://workbench.test/form"} {
		req := httptest.NewRequest(http.MethodPost, "http://workbench.test/action", nil)
		req.Header.Set("X-CSRF-Token", token)
		req.Header.Set("Referer", referer)
		req.Header.Set("X-Forwarded-Proto", "http")
		req.Header.Set("Forwarded", "proto=http;host=evil.test")
		for _, cookie := range get.Result().Cookies() {
			if !cookie.Secure {
				t.Fatal("production CSRF cookie lost Secure")
			}
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		want := http.StatusForbidden
		if referer == "https://workbench.test/form" {
			want = http.StatusOK
		}
		if rec.Code != want {
			t.Fatalf("referer %s status=%d want=%d", referer, rec.Code, want)
		}
	}
}
