// =============================================================================
// 文件: internal/module/schedule/iteration_test.go
// 模块: 排期工作台
// 类型: test
// 职责: 迭代周期、计划归属与原生调用前权限回归。
// 依赖: 无
// =============================================================================
package schedule

import (
	"reflect"
	"testing"
	"time"

	"workbench/internal/model"
	"workbench/internal/pkg/perm"
)

func TestIterationDatesAndNativePlanArray(t *testing.T) {
	now := time.Date(2026, 10, 3, 17, 0, 0, 0, time.UTC)
	for period, end := range map[string]string{"2w": "2026-10-17", "4w": "2026-10-31"} {
		body, err := iterationBody(CreateIterationReq{Period: period}, 2, "admin", nil, now)
		if err != nil || body["begin"] != "2026-10-04" || body["end"] != end || body["name"] != "2026-10-04 - "+end {
			t.Fatalf("period %s: %v, %v", period, body, err)
		}
	}
	plan := MatchingPlanItem{ID: 7, Begin: "2026-10-05", End: "2026-10-28"}
	body, err := iterationBody(CreateIterationReq{Period: "plan", PlanID: 7}, 2, "admin", []MatchingPlanItem{plan}, now)
	if err != nil || body["begin"] != plan.Begin || body["end"] != plan.End || body["name"] != plan.Begin+" - "+plan.End {
		t.Fatalf("plan dates: %v, %v", body, err)
	}
	if !reflect.DeepEqual(body["plans"], [][]uint{{}, {}, {7}}) {
		t.Fatalf("native plan contract: %v", body["plans"])
	}
	_, err = iterationBody(CreateIterationReq{Period: "plan", PlanID: 8}, 2, "admin", []MatchingPlanItem{plan}, now)
	assertForbidden(t, err, "计划不属于该产品或已关闭")
}

func TestCreateIterationDenialPrecedesNativeCall(t *testing.T) {
	db, mock := newAuthzMockDB(t)
	svc := NewService(NewRepo(db), nil, nil, nil, nil)
	expectStoryAuthzDenied(mock, 5, 100, 2, 3)
	ctx := perm.WithGranted(t.Context(), map[string]bool{perm.ScheduleList.String(): true})
	_, err := svc.CreateIteration(ctx, &model.User{ID: 5, Account: "demo_outsider"}, CreateIterationReq{StoryID: 100})
	assertForbidden(t, err, StoryWriteDenialMessage)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
