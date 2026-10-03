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
	"reflect"
	"strings"
	"testing"
	"workbench/internal/config"
	"workbench/internal/model"
	"workbench/internal/pkg/zentao"
)

func TestToStoryPreservesMainContract(t *testing.T) {
	const expected = `{
	 "estimateLaunch":"2026-10-31","developFinish":"2026-10-15",
	 "testFinish":"2026-10-20","verifyFinish":"2026-10-25","QD":"qa",
	 "product":[2,3],"module":[0,0],"plan":["11","12"],
	 "title":["主研需","配研需"],"spec":["主描述","配描述"],
	 "source":["",""],"sourceNote":["",""],"verify":["",""],
	 "assignedTo":["dev1","dev2"],"category":["feature","feature"],
	 "pri":["3","3"],"estimate":["8.5","0"],"keywords":["",""],"color":["",""]
	}`
	var want map[string]any
	if err := json.Unmarshal([]byte(expected), &want); err != nil {
		t.Fatal(err)
	}
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tokens" {
			_, _ = w.Write([]byte(`{"token":"synthetic-test-token"}`))
			return
		}
		calls++
		var got map[string]any
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Error(err)
		}
		if r.Method != http.MethodPost || r.URL.Path != "/demand/100/tostory" || !reflect.DeepEqual(got, want) {
			t.Errorf("Main native contract changed: %s %s, body=%v", r.Method, r.URL.Path, got)
		}
		_, _ = w.Write([]byte(`{"message":"success","storyIds":[21,22]}`))
	}))
	defer upstream.Close()
	client := zentao.NewClient(config.ZentaoConfig{API: upstream.URL})
	ids, err := toStory(zentao.WithAccount(t.Context(), "fixture"), client, 100, toStoryBodyInput{
		EstimateLaunch: "2026-10-31", DevelopFinish: "2026-10-15", TestFinish: "2026-10-20",
		VerifyFinish: "2026-10-25", QD: "qa",
		Stories: []toStoryStoryInput{
			{ProductID: 2, PlanID: 11, Title: " 主研需 ", Spec: "主描述", AssignedTo: "dev1", Estimate: 8.5},
			{ProductID: 3, PlanID: 12, Title: "配研需", Spec: "配描述", AssignedTo: "dev2", Estimate: 0},
		},
	})
	if err != nil || calls != 1 || !reflect.DeepEqual(ids, []uint{21, 22}) {
		t.Fatalf("native creation = %v, %v; calls=%d", ids, err, calls)
	}
}

func TestToStoryFailureNeverUsesAlternateWriteRoute(t *testing.T) {
	for _, body := range []string{`{"storyIds":[21]}`, `{"storyIds":[21,0]}`, `{"result":"fail","message":"not found"}`} {
		calls := 0
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/tokens" {
				_, _ = w.Write([]byte(`{"token":"synthetic-test-token"}`))
				return
			}
			calls++
			_, _ = w.Write([]byte(body))
		}))
		client := zentao.NewClient(config.ZentaoConfig{API: upstream.URL})
		_, err := toStory(zentao.WithAccount(t.Context(), "fixture"), client, 100,
			toStoryBodyInput{Stories: []toStoryStoryInput{{ProductID: 2}, {ProductID: 3}}})
		upstream.Close()
		if err == nil || calls != 1 {
			t.Fatalf("invalid native response accepted or creation replayed: %v, calls=%d", err, calls)
		}
	}
}

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

func TestNativeDeletesPreserveMainContractWithoutReplay(t *testing.T) {
	for kind, field := range map[string]string{"tasks": "taskIdList", "stories": "storyIdList"} {
		calls := 0
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/tokens" {
				_, _ = w.Write([]byte(`{"token":"synthetic-test-token"}`))
				return
			}
			calls++
			var body map[string][]uint
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if r.Method != http.MethodPost || r.URL.Path != "/delete"+kind || !reflect.DeepEqual(body, map[string][]uint{field: {7, 8, 9}}) {
				t.Errorf("Main batch contract: %s %s %+v", r.Method, r.URL.Path, body)
			}
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer upstream.Close()
		client := zentao.NewClient(config.ZentaoConfig{API: upstream.URL})
		err := deleteSchedulingObjects(zentao.WithAccount(t.Context(), "fixture"), client, kind, []uint{7, 8, 9})
		if err == nil || !strings.Contains(err.Error(), "部分操作可能已提交") || calls != 1 {
			t.Fatalf("failure = %v, calls = %d", err, calls)
		}
	}
}
