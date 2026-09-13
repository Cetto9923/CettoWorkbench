// 验证 CSRF 中间件行为合同 (Phase X1)。
package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/justinas/nosurf"
)

func newTestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK:" + r.Method))
	})
}

// TestCSRF_AllUnsafeMethods 验证所有不安全方法（POST/PUT/PATCH/DELETE）缺失 token 时均被拦截为 403，
// 安全方法（GET/HEAD）正常放行。
func TestCSRF_AllUnsafeMethods(t *testing.T) {
	csrfMW := CSRF(false)
	handler := csrfMW(newTestHandler())

	unsafeMethods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for _, method := range unsafeMethods {
		t.Run(method+"_MissingToken_403", func(t *testing.T) {
			req := httptest.NewRequest(method, "/submit", strings.NewReader("key=val"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("expected status 403 for %s without token, got %d", method, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), "CSRF 校验失败") {
				t.Fatalf("expected CSRF error body, got %s", rec.Body.String())
			}
		})
	}

	safeMethods := []string{http.MethodGet, http.MethodHead}
	for _, method := range safeMethods {
		t.Run(method+"_SafeMethod_200", func(t *testing.T) {
			req := httptest.NewRequest(method, "/page", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200 for %s, got %d", method, rec.Code)
			}
		})
	}
}

// TestCSRF_CrossOriginOrJSONFailure 验证 JSON/AJAX 请求在 CSRF 失败时返回 403 JSON，不混淆为 401。
func TestCSRF_CrossOriginOrJSONFailure(t *testing.T) {
	csrfMW := CSRF(false)
	handler := csrfMW(newTestHandler())

	cases := []struct {
		name        string
		headers     map[string]string
		contentType string
	}{
		{
			name:    "Accept_JSON",
			headers: map[string]string{"Accept": "application/json"},
		},
		{
			name:    "XRequestedWith_AJAX",
			headers: map[string]string{"X-Requested-With": "XMLHttpRequest"},
		},
		{
			name:    "Content_Type_JSON",
			headers: map[string]string{"Content-Type": "application/json"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/mutation", strings.NewReader(`{"field":"val"}`))
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("expected status 403, got %d", rec.Code)
			}
			ct := rec.Header().Get("Content-Type")
			if !strings.Contains(ct, "application/json") {
				t.Fatalf("expected application/json Content-Type, got %q", ct)
			}
			body := rec.Body.String()
			if !strings.Contains(body, `"success":false`) || !strings.Contains(body, `"code":403`) {
				t.Fatalf("expected JSON 403 error envelope, got %s", body)
			}
		})
	}
}

// TestCSRF_ValidTokenRoundtrip 验证获取合法 token/cookie 后的正常提交与错误 token 拒绝。
func TestCSRF_ValidTokenRoundtrip(t *testing.T) {
	csrfMW := CSRF(false)

	var capturedToken string
	appHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			capturedToken = nosurf.Token(r)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("TOKEN_PAGE"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK:" + r.Method))
	})

	handler := csrfMW(appHandler)

	// Step 1: GET to obtain cookie & token
	getReq := httptest.NewRequest(http.MethodGet, "/form", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("GET /form status = %d, want 200", getRec.Code)
	}

	cookie := getRec.Header().Get("Set-Cookie")
	if cookie == "" || !strings.Contains(cookie, "csrf_token=") {
		t.Fatalf("expected Set-Cookie with csrf_token, got %q", cookie)
	}
	rawCookie := getRec.Result().Cookies()

	if capturedToken == "" {
		t.Fatalf("expected non-empty captured nosurf token")
	}

	// Step 2: POST with valid Header token
	postReq := httptest.NewRequest(http.MethodPost, "/form", strings.NewReader("foo=bar"))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("Sec-Fetch-Site", "same-origin")
	postReq.Header.Set("X-CSRF-Token", capturedToken)
	for _, c := range rawCookie {
		t.Logf("Cookie from GET: %s = %s", c.Name, c.Value)
		postReq.AddCookie(c)
	}
	t.Logf("Captured token: %s", capturedToken)
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("POST with valid header token status = %d (body: %s), want 200", postRec.Code, postRec.Body.String())
	}

	// Step 3: POST with valid Form token
	formValues := url.Values{}
	formValues.Set("csrf_token", capturedToken)
	formValues.Set("foo", "bar")
	formReq := httptest.NewRequest(http.MethodPost, "/form", strings.NewReader(formValues.Encode()))
	formReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	formReq.Header.Set("Sec-Fetch-Site", "same-origin")
	for _, c := range rawCookie {
		formReq.AddCookie(c)
	}
	formRec := httptest.NewRecorder()
	handler.ServeHTTP(formRec, formReq)

	if formRec.Code != http.StatusOK {
		t.Fatalf("POST with valid form token status = %d, want 200", formRec.Code)
	}

	// Step 4: POST with wrong token -> 403
	badReq := httptest.NewRequest(http.MethodPost, "/form", strings.NewReader("foo=bar"))
	badReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	badReq.Header.Set("Sec-Fetch-Site", "same-origin")
	badReq.Header.Set("X-CSRF-Token", "invalid-token-12345678901234567890123456789012")
	for _, c := range rawCookie {
		badReq.AddCookie(c)
	}
	badRec := httptest.NewRecorder()
	handler.ServeHTTP(badRec, badReq)

	if badRec.Code != http.StatusForbidden {
		t.Fatalf("POST with invalid token status = %d, want 403", badRec.Code)
	}
}

// TestCSRF_SecureCookie 验证 cookieSecure 控制 Set-Cookie 中的 Secure 属性。
func TestCSRF_SecureCookie(t *testing.T) {
	// Secure = true
	handlerSecure := CSRF(true)(newTestHandler())
	rec1 := httptest.NewRecorder()
	handlerSecure.ServeHTTP(rec1, httptest.NewRequest(http.MethodGet, "/", nil))
	cookies1 := rec1.Result().Cookies()
	var csrfCookie1 *http.Cookie
	for _, c := range cookies1 {
		if c.Name == "csrf_token" {
			csrfCookie1 = c
		}
	}
	if csrfCookie1 == nil || !csrfCookie1.Secure {
		t.Fatalf("expected Secure=true on csrf_token cookie, got %+v", csrfCookie1)
	}

	// Secure = false
	handlerInsecure := CSRF(false)(newTestHandler())
	rec2 := httptest.NewRecorder()
	handlerInsecure.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/", nil))
	cookies2 := rec2.Result().Cookies()
	var csrfCookie2 *http.Cookie
	for _, c := range cookies2 {
		if c.Name == "csrf_token" {
			csrfCookie2 = c
		}
	}
	if csrfCookie2 == nil || csrfCookie2.Secure {
		t.Fatalf("expected Secure=false on csrf_token cookie, got %+v", csrfCookie2)
	}
}
