package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alexedwards/scs/v2"
	"go.uber.org/zap"
	mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"

	"workbench/internal/config"
	"workbench/internal/module/debug"
)

// Use real session loading, real GORM model hooks, real authorization and the
// registered debug handler. Only database transport is mocked; no header sets actor.
func TestDebugProductionChainTrustedIdentity(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	t.Chdir(filepath.Join(filepath.Dir(source), "../.."))
	for _, actor := range []string{"anonymous", "regular", "admin"} {
		t.Run(actor, func(t *testing.T) {
			conn, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			db, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			mgr := scs.New()
			cfg := &config.Config{App: config.App{Env: "test"}, RateLimit: config.RateLimit{GlobalRPS: 1000}}
			deps := RouteDeps{DB: db, SessionMgr: mgr, SqlPerfHandler: debug.NewHandler(debug.NewService(debug.NewRepo(t.TempDir())))}
			srv := New(cfg, zap.NewNop(), db, mgr, nil, nil, deps)
			handler := srv.BuildHandler()
			req := httptest.NewRequest(http.MethodGet, "/debug/sqlperf/requests?superadmin=true&is_super_admin=1", nil)
			req.Header.Set("Accept", "application/json")
			req.Header.Set("X-Test-SuperAdmin", "1")
			req.Header.Set("X-SuperAdmin", "true")
			want := http.StatusUnauthorized
			if actor != "anonymous" {
				// Seed only the server-side session ID; role is loaded from database rows.
				seed := httptest.NewRecorder()
				mgr.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mgr.Put(r.Context(), "userID", int64(7))
					w.WriteHeader(http.StatusNoContent)
				})).ServeHTTP(seed, httptest.NewRequest(http.MethodGet, "/", nil))
				for _, cookie := range seed.Result().Cookies() {
					req.AddCookie(cookie)
				}
				mock.ExpectQuery("SELECT .* FROM `zt_user`").WithArgs(int64(7), "0", 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "account", "role", "deleted"}).AddRow(7, actor, "", "0"))
				want = http.StatusOK
				if actor == "regular" {
					mock.ExpectQuery("SELECT DISTINCT .*zt_role_permissions").WithArgs(int64(7), 0, 0, true).
						WillReturnRows(sqlmock.NewRows([]string{"permCode"}))
					want = http.StatusForbidden
				}
				mock.ExpectQuery("SELECT .* FROM `zt_menus`").WillReturnRows(sqlmock.NewRows([]string{"id"}))
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != want {
				t.Fatalf("%s status=%d want=%d", actor, rec.Code, want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
