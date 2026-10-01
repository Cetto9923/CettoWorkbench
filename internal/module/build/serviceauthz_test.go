// =============================================================================
// 文件: internal/module/build/serviceauthz_test.go
// 模块: 版本管理
// 类型: test
// 职责: 验证来源业务需求全部授权后才调用假禅道服务。
// =============================================================================
package build

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"workbench/internal/config"
	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/zentao"
)

func buildMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})
	return db, mock
}

func expectBuildDemand(mock sqlmock.Sqlmock, allowed bool) {
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_gf_user_roles`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	count := 0
	if allowed {
		count = 1
	}
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM zt_demand`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(count))
	if !allowed {
		mock.ExpectQuery(`SELECT d\.id, COALESCE`).WillReturnRows(sqlmock.NewRows([]string{"id", "path"}))
	}
}

type buildWriteCase struct {
	name    string
	origins []uint
	denied  bool
}

func TestBuildWritesAuthorizeAllOrigins(t *testing.T) {
	for _, unlink := range []bool{false, true} {
		for _, tc := range []buildWriteCase{
			{"independent", []uint{0}, false}, {"related", []uint{41}, false},
			{"unrelated", []uint{42}, true}, {"mixed", []uint{41, 42}, true},
		} {
			t.Run(fmt.Sprintf("%s/unlink=%t", tc.name, unlink), func(t *testing.T) { checkBuildWrite(t, tc, unlink) })
		}
	}
}

func setupBuildWrite(t *testing.T, tc buildWriteCase) (*Service, *atomic.Int32, LinkStoriesReq) {
	t.Helper()
	db, mock := buildMockDB(t)
	mock.ExpectQuery("SELECT .* FROM `zt_build`").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(9))
	rows := sqlmock.NewRows([]string{"id", "fromDemand"})
	ids := []string{}
	for i, origin := range tc.origins {
		rows.AddRow(i+1, origin)
		ids = append(ids, strconv.Itoa(i+1))
	}
	mock.ExpectQuery("SELECT id, fromDemand FROM `zt_story`").WillReturnRows(rows)
	for _, origin := range tc.origins {
		if origin > 0 {
			expectBuildDemand(mock, origin == 41)
		}
	}
	hits := new(atomic.Int32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tokens" {
			_, _ = w.Write([]byte(`{"token":"stub"}`))
			return
		}
		hits.Add(1)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	t.Cleanup(server.Close)
	return NewService(NewRepo(db), nil, zentao.NewClient(config.ZentaoConfig{API: server.URL}), nil), hits, LinkStoriesReq{Stories: strings.Join(ids, ",")}
}

func checkBuildWrite(t *testing.T, tc buildWriteCase, unlink bool) {
	t.Helper()
	svc, hits, req := setupBuildWrite(t, tc)
	actor := &model.User{ID: 3, Account: "tester"}
	ctx := zentao.WithAccount(t.Context(), "tester")
	var err error
	if unlink {
		err = svc.UnlinkStories(ctx, actor, 9, req)
	} else {
		err = svc.LinkStories(ctx, actor, 9, req)
	}
	if tc.denied {
		biz, ok := errorx.IsBizError(err)
		if !ok || biz.Code != errorx.ErrCodeForbidden || !strings.Contains(biz.Msg, "US42") || hits.Load() != 0 {
			t.Fatalf("err=%v hits=%d", err, hits.Load())
		}
		return
	}
	if err != nil || hits.Load() != 1 {
		t.Fatalf("err=%v hits=%d", err, hits.Load())
	}
}
