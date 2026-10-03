// =============================================================================
// 文件: internal/module/schedule/safety_test.go
// 模块: 排期工作台
// 类型: test
// 职责: 验证子对象越权在写入前拒绝，以及任务编辑保留原生剩余工时。
// =============================================================================

package schedule

import (
	"context"
	"encoding/json"
	"github.com/DATA-DOG/go-sqlmock"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"workbench/internal/config"
	"workbench/internal/model"
	"workbench/internal/pkg/zentao"
)

func TestSchedulingRejectsForeignStoryBeforeWrites(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	mock.ExpectQuery(`SELECT id, product, fromDemand FROM zt_story`).WithArgs(uint(100), uint(100), uint(100), uint(0)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "product", "fromDemand"}).AddRow(11, 2, 100))
	svc := NewService(NewRepo(db), nil, nil, nil, nil)
	err := svc.validateSchedulingObjects(t.Context(), &model.User{Account: "fixture", IsSuperAdmin: true}, 100, 0,
		&SaveSchedulingReq{Stories: []SaveSchedulingStory{{Action: "delete", ID: 999}}})
	assertForbidden(t, err, "研发需求不属于当前排期对象")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSchedulingRejectsForeignTaskBeforeWrites(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	mock.ExpectQuery(`SELECT id, product, fromDemand FROM zt_story`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "product", "fromDemand"}).AddRow(11, 2, 100))
	mock.ExpectQuery(`SELECT id, story FROM zt_task`).WithArgs(uint(999)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "story"}).AddRow(999, 12))
	svc := NewService(NewRepo(db), nil, nil, nil, nil)
	err := svc.validateSchedulingObjects(t.Context(), &model.User{Account: "fixture", IsSuperAdmin: true}, 100, 0,
		&SaveSchedulingReq{Stories: []SaveSchedulingStory{{Action: "edit", ID: 11, ProductID: 2, Tasks: []SaveSchedulingTask{{Action: "delete", ID: 999}}}}})
	assertForbidden(t, err, "任务不属于当前研发需求")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTaskEditDoesNotSendLeftOrConsumed(t *testing.T) {
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tokens" {
			_, _ = w.Write([]byte(`{"token":"synthetic-test-token"}`))
			return
		}
		calls++
		if r.URL.Path != "/tasks/7" || r.Method != http.MethodPut {
			t.Errorf("wrong task action: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		for _, key := range []string{"left", "consumed", "status"} {
			if _, ok := body[key]; ok {
				t.Errorf("task edit overrides %s", key)
			}
		}
		if body["estimate"] != float64(8) || body["execution"] != float64(3) {
			t.Errorf("lost edited fields: %v", body)
		}
		_, _ = w.Write([]byte(`{"result":"success"}`))
	}))
	defer upstream.Close()
	svc := &Service{ztAPI: zentao.NewClient(config.ZentaoConfig{API: upstream.URL})}
	err := svc.saveEditedTask(zentao.WithAccount(context.Background(), "fixture"), SaveSchedulingTask{ID: 7, Estimate: 8, ExecutionID: 3})
	if err != nil || calls != 1 {
		t.Fatalf("task edit = %v, native calls = %d", err, calls)
	}
}

func TestWindowStatsUsesThreeQueriesForMultipleWindows(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	mock.ExpectQuery(`(?s)SELECT window_id, item_kind, item_id`).WithArgs(1, 2, 1, 2, 1, 2).
		WillReturnRows(sqlmock.NewRows([]string{"window_id", "item_kind", "item_id"}).AddRow(1, "demand", 9).AddRow(2, "story", 10))
	mock.ExpectQuery(`(?s)SELECT linked.windowID, COALESCE`).WithArgs(1, 2).
		WillReturnRows(sqlmock.NewRows([]string{"windowID", "total"}).AddRow(1, 12.5).AddRow(2, 7))
	mock.ExpectQuery(`(?s)SELECT linked.windowID,.*SUM`).WithArgs(1, 2).
		WillReturnRows(sqlmock.NewRows([]string{"windowID", "devCount", "testCount", "deliverCount"}).AddRow(1, 2, 3, 4).AddRow(2, 0, 1, 0))
	stats, err := NewRepo(db).windowsStats(t.Context(), []uint64{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	if stats[1].DemandCount != 1 || stats[1].Consumed != 12.5 || stats[1].DevCount != 2 || stats[1].TestCount != 3 || stats[1].DeliverCount != 4 || stats[2].Consumed != 7 {
		t.Fatalf("batch metrics lost columns: %+v", stats)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestNativeDeletesStopAfterFailureWithoutReplay(t *testing.T) {
	for _, kind := range []string{"tasks", "stories"} {
		t.Run(kind, func(t *testing.T) {
			var calls []string
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/tokens" {
					_, _ = w.Write([]byte(`{"token":"synthetic-test-token"}`))
					return
				}
				calls = append(calls, r.Method+" "+r.URL.Path)
				if r.URL.Path == "/"+kind+"/8" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				_, _ = w.Write([]byte(`{"result":"success"}`))
			}))
			defer upstream.Close()
			client := zentao.NewClient(config.ZentaoConfig{API: upstream.URL})
			err := deleteSchedulingObjects(zentao.WithAccount(t.Context(), "fixture"), client, kind, []uint{7, 8, 9})
			if err == nil || !strings.Contains(err.Error(), "此前 1 项已提交") {
				t.Fatalf("partial failure = %v", err)
			}
			want := "DELETE /" + kind + "/7,DELETE /" + kind + "/8"
			if got := strings.Join(calls, ","); got != want {
				t.Fatalf("native calls = %s, want %s", got, want)
			}
		})
	}
}
