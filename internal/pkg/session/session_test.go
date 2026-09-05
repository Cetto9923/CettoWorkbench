package session

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"workbench/internal/config"
)

// TestSession_ModesAgree 验证 session cookie Secure 属性在不同运行模式下保持一致。
func TestSession_ModesAgree(t *testing.T) {
	// Case A: prod 模式强制 Secure
	prodCfg := &config.Config{
		App: config.App{
			Env: "prod",
		},
		Session: config.Session{
			CookieName:    "test_sid",
			LifetimeHours: 2,
			CookieSecure:  true,
		},
	}
	prodMgr := New(prodCfg)
	if !prodMgr.Cookie.Secure {
		t.Fatal("prod session manager must have Cookie.Secure = true")
	}

	// Case B: local/dev 模式且 cookieSecure=false 时 Secure 为 false
	localCfg := &config.Config{
		App: config.App{
			Env: "local",
		},
		Session: config.Session{
			CookieName:    "test_sid",
			LifetimeHours: 2,
			CookieSecure:  false,
		},
	}
	localMgr := New(localCfg)
	if localMgr.Cookie.Secure {
		t.Fatal("local session manager must have Cookie.Secure = false")
	}
}

// TestSession_LocalCookieRoundtrip 验证在 local HTTP 模式下 cookie 可正常发送与回传并保持登录态。
func TestSession_LocalCookieRoundtrip(t *testing.T) {
	localCfg := &config.Config{
		App: config.App{
			Env: "local",
		},
		Session: config.Session{
			CookieName:    "workbench_sid",
			LifetimeHours: 2,
			CookieSecure:  false,
		},
	}
	mgr := New(localCfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		PutUserID(r.Context(), mgr, 10086)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("logged_in"))
	})
	mux.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
		uid := GetUserID(r.Context(), mgr)
		if uid == 10086 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok_user"))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
		}
	})

	handler := mgr.LoadAndSave(mux)

	// Step 1: 模拟登录请求
	loginReq := httptest.NewRequest(http.MethodPost, "/login", nil)
	loginRR := httptest.NewRecorder()
	handler.ServeHTTP(loginRR, loginReq)

	if loginRR.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200", loginRR.Code)
	}

	cookieHeader := loginRR.Header().Get("Set-Cookie")
	if !strings.Contains(cookieHeader, "workbench_sid=") {
		t.Fatalf("Set-Cookie missing session cookie: %s", cookieHeader)
	}
	// 验证在 local HTTP 模式下不含 Secure 属性，保证在本地 HTTP 下浏览器可正常回送
	if strings.Contains(strings.ToLower(cookieHeader), "secure") {
		t.Fatalf("local HTTP session cookie should not have Secure flag: %s", cookieHeader)
	}

	// 提取 cookie 值
	parts := strings.Split(cookieHeader, ";")
	rawCookie := strings.TrimSpace(parts[0])

	// Step 2: 携带该 cookie 访问需登录接口
	profileReq := httptest.NewRequest(http.MethodGet, "/profile", nil)
	profileReq.Header.Set("Cookie", rawCookie)
	profileRR := httptest.NewRecorder()
	handler.ServeHTTP(profileRR, profileReq)

	if profileRR.Code != http.StatusOK {
		t.Fatalf("profile status = %d, want 200 (cookie roundtrip failed)", profileRR.Code)
	}
	if profileRR.Body.String() != "ok_user" {
		t.Fatalf("profile body = %q, want 'ok_user'", profileRR.Body.String())
	}
}

// TestSession_UserIDLifecycle 验证 Put, Get, Clear, Renew 的基本生命周期。
func TestSession_UserIDLifecycle(t *testing.T) {
	cfg := &config.Config{
		App: config.App{Env: "dev"},
		Session: config.Session{
			CookieName:    "test_sid",
			LifetimeHours: 1,
		},
	}
	mgr := New(cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if GetUserID(ctx, mgr) != 0 {
			t.Fatal("initial user ID should be 0")
		}

		PutUserID(ctx, mgr, 42)
		if GetUserID(ctx, mgr) != 42 {
			t.Fatalf("user ID after Put = %d, want 42", GetUserID(ctx, mgr))
		}

		if err := Renew(ctx, mgr); err != nil {
			t.Fatalf("Renew error: %v", err)
		}
		if GetUserID(ctx, mgr) != 42 {
			t.Fatalf("user ID after Renew = %d, want 42", GetUserID(ctx, mgr))
		}

		if err := Clear(ctx, mgr); err != nil {
			t.Fatalf("Clear error: %v", err)
		}
		if GetUserID(ctx, mgr) != 0 {
			t.Fatalf("user ID after Clear = %d, want 0", GetUserID(ctx, mgr))
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := mgr.LoadAndSave(mux)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("test handler failed with status: %d", rr.Code)
	}
}
