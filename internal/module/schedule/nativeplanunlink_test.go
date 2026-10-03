// =============================================================================
// 文件: internal/module/schedule/nativeplanunlink_test.go
// 模块: 排期工作台
// 类型: test
// 职责: 原生计划摘除契约、部分失败停止及零本地写入。
// =============================================================================

package schedule

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"workbench/internal/config"
	"workbench/internal/pkg/zentao"
)

func TestNativePlanUnlinkContractAndFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) { testNativePlanUnlink(t, fail) })
	}
}

func testNativePlanUnlink(t *testing.T, fail bool) {
	db, mock := newAuthzMockDB(t)
	mock.ExpectQuery(`SELECT plan FROM zt_planstory`).WithArgs(uint(7), uint(20)).WillReturnRows(sqlmock.NewRows([]string{"plan"}).AddRow(11).AddRow(12))
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tokens" {
			_, _ = w.Write([]byte(`{"token":"synthetic-test-token"}`))
			return
		}
		calls++
		if r.Method != http.MethodPost || r.URL.Path != fmt.Sprintf("/productplans/%d/unlinkstories", 10+calls) {
			t.Errorf("wrong action: %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			Stories []uint `json:"stories"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !reflect.DeepEqual(body.Stories, []uint{7}) {
			t.Errorf("wrong story body: %+v, %v", body, err)
		}
		if fail {
			_, _ = w.Write([]byte(`{"result":"fail","message":"fixture failure"}`))
			return
		}
		_, _ = fmt.Fprintf(w, `{"id":%d}`, 10+calls)
	}))
	defer upstream.Close()
	svc := &Service{repo: NewRepo(db), ztAPI: zentao.NewClient(config.ZentaoConfig{API: upstream.URL})}
	err := svc.removeStoryFromOtherPlans(zentao.WithAccount(t.Context(), "fixture"), 7, 20)
	wantCalls := 2
	if fail {
		wantCalls = 1
	}
	if (err != nil) != fail || calls != wantCalls {
		t.Fatalf("calls=%d, err=%v", calls, err)
	}
	if err != nil && !strings.Contains(err.Error(), "计划 11") {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
