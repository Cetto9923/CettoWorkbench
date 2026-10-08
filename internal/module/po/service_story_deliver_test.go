// =============================================================================
// 文件: internal/module/po/service_story_deliver_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证独立研发需求发起交付按纯编号加载，且不存在时不回落到业务需求。
// =============================================================================

package po

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"workbench/internal/config"
	"workbench/internal/pkg/zentao"

	"github.com/DATA-DOG/go-sqlmock"

	"workbench/internal/model"
)

func TestGetStoryDeliverMeta_DisplayIDHasNoUSPrefix(t *testing.T) {
	s, mock := newServiceForDeliverTest(t)
	mock.ExpectQuery(`(?s)SELECT.*FROM zt_story.*WHERE id = \?`).
		WithArgs(uint(68485)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "status", "deleted", "assignedTo", "productReqM", "deliverDate",
			"isCarReview", "verifyDate", "verifyPlan", "veriFier", "delivered", "windowBound",
		}).AddRow(68485, "工时填报", "active", "0", "alice", "alice", "2026-10-20", "0", "1", "计划", "alice", 0, 1))
	mock.ExpectQuery(`(?s)SELECT.*FROM zt_bug`).
		WithArgs(uint(68485)).
		WillReturnRows(sqlmock.NewRows([]string{"severe_count", "open_count"}).AddRow(0, 0))
	mock.ExpectQuery(`(?s)SELECT.*FROM zt_planstory`).
		WithArgs(uint(68485)).
		WillReturnRows(sqlmock.NewRows([]string{"window_id", "window_name", "release_date"}).AddRow(9, "十月窗口", "2026-10-20"))
	mock.ExpectQuery(`(?s)SELECT.*FROM zt_versionwindow`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "releaseDate"}).AddRow(9, "十月窗口", "2026-10-20"))
	mock.ExpectQuery(`(?s)SELECT.*FROM zt_user`).
		WillReturnRows(sqlmock.NewRows([]string{"value", "label", "pinyin"}).AddRow("alice", "Alice (alice)", "alice"))

	meta, err := s.GetStoryDeliverMeta(context.Background(), &model.User{Account: "alice"}, 68485)
	if err != nil {
		t.Fatalf("GetStoryDeliverMeta: %v", err)
	}
	if meta.DisplayID != "68485" {
		t.Fatalf("displayId = %q", meta.DisplayID)
	}
	if !meta.Precheck.CanSubmit {
		t.Fatalf("precheck = %+v", meta.Precheck)
	}
}

func TestGetStoryDeliverMeta_MissingStory(t *testing.T) {
	s, mock := newServiceForDeliverTest(t)
	mock.ExpectQuery(`(?s)SELECT.*FROM zt_story.*WHERE id = \?`).
		WithArgs(uint(68485)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "status", "deleted", "assignedTo", "productReqM", "deliverDate",
			"isCarReview", "verifyDate", "verifyPlan", "veriFier", "delivered", "windowBound",
		}))
	_, err := s.GetStoryDeliverMeta(context.Background(), &model.User{Account: "alice"}, 68485)
	if !errors.Is(err, errStoryNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func expectDeliverStory(mock sqlmock.Sqlmock, assigned string, delivered, bound int) {
	mock.ExpectQuery(`(?s)SELECT.*FROM zt_story.*WHERE id = \? AND deleted = '0' AND fromDemand = 0 AND type = 'story'.*isParent = '0'.*sourceType`).WithArgs(uint(68485)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "assignedTo", "productReqM", "delivered", "windowBound"}).AddRow(68485, "active", assigned, "pm", delivered, bound))
}

func TestDeliverStoryGuards(t *testing.T) {
	cases := []struct {
		name, account            string
		delivered, bound, severe int
		want                     error
	}{
		{"anonymous", "", 0, 1, 0, errHomeActionForbidden},
		{"other user", "other", 0, 1, 0, errHomeActionForbidden},
		{"already delivered", "alice", 1, 1, 0, errHomeActionConflict},
		{"unscheduled", "alice", 0, 0, 0, errHomeActionConflict},
		{"severe bugs", "alice", 0, 1, 2, errDeliverBlocked},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, mock := newServiceForDeliverTest(t)
			if tc.account != "" {
				expectDeliverStory(mock, "alice", tc.delivered, tc.bound)
			}
			if tc.severe > 0 {
				mock.ExpectQuery(`(?s)SELECT.*FROM zt_bug.*WHERE b.story = \?`).WithArgs(uint(68485)).WillReturnRows(sqlmock.NewRows([]string{"severe_count", "open_count"}).AddRow(tc.severe, tc.severe))
			}
			err := s.DeliverStory(context.Background(), &model.User{Account: tc.account}, DemandDeliverReq{ID: 68485})
			if !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDeliverStoryNativePayload(t *testing.T) {
	s, mock := newServiceForDeliverTest(t)
	expectDeliverStory(mock, "someone", 0, 1)
	mock.ExpectQuery(`(?s)SELECT.*FROM zt_bug.*WHERE b.story = \?`).WithArgs(uint(68485)).WillReturnRows(sqlmock.NewRows([]string{"severe_count", "open_count"}).AddRow(0, 3))
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/tokens" {
			_, _ = w.Write([]byte(`{"token":"synthetic-test-token"}`))
			return
		}
		calls++
		if r.Method != http.MethodPost || r.URL.Path != "/stories/68485/deliver" {
			t.Errorf("unexpected action %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if payload["veriFier"] != "pm" || payload["verifyPlan"] != "验证计划" || payload["comment"] != "工作台发起交付" {
			t.Errorf("payload = %v", payload)
		}
		_, _ = w.Write([]byte(`{"result":"fail","message":"模拟原生拒绝"}`))
	}))
	defer upstream.Close()
	s.ztAPI = zentao.NewClient(config.ZentaoConfig{API: upstream.URL})
	err := s.DeliverStory(zentao.WithAccount(context.Background(), "pm"), &model.User{Account: "pm"}, DemandDeliverReq{ID: 68485, DeliverDate: "2026-10-20", VerifyDate: "1", VerifyPlan: "验证计划", Verifier: "pm"})
	if err == nil || calls != 1 {
		t.Fatalf("native rejection lost: calls=%d err=%v", calls, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
