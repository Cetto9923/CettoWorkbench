// =============================================================================
// 文件: internal/module/po/blocker_test.go
// 模块: PO 工作台
// 类型: action
// 职责: 「卡点快速响应」纯函数单测：等级优先级、限长、契约，以及从真实日期字段算
//       等级 / dueLabel / owner 的工具函数。
// 依赖: 无
// =============================================================================

package po

import (
	"reflect"
	"testing"
	"time"
)

func TestLevelOrder(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   BlockerLevel
		want int
	}{
		{in: BlockerLevelBlocked, want: 0},
		{in: BlockerLevelOverdue, want: 1},
		{in: BlockerLevelRisk, want: 2},
		{in: BlockerLevelCoord, want: 3},
		{in: BlockerLevel("unknown"), want: 99},
	}
	for _, tc := range cases {
		if got := levelOrder(tc.in); got != tc.want {
			t.Errorf("levelOrder(%q)=%d want %d", tc.in, got, tc.want)
		}
	}
}

func TestLevelLabel(t *testing.T) {
	t.Parallel()

	cases := map[BlockerLevel]string{
		BlockerLevelBlocked: "阻塞",
		BlockerLevelOverdue: "超期",
		BlockerLevelRisk:    "风险",
		BlockerLevelCoord:   "协调",
		BlockerLevel("foo"): "foo",
	}
	for in, want := range cases {
		if got := levelLabel(in); got != want {
			t.Errorf("levelLabel(%q)=%q want %q", in, got, want)
		}
	}
}

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestDueMetrics(t *testing.T) {
	t.Parallel()

	today := day("2026-09-02")
	cases := []struct {
		name        string
		due         *time.Time
		wantDueAt   string
		wantLabel   string
		wantOverdue bool
	}{
		{name: "nil", due: nil, wantDueAt: "", wantLabel: "", wantOverdue: false},
		{name: "today", due: ptrTime(today), wantDueAt: "2026-09-02", wantLabel: "今日", wantOverdue: false},
		{name: "future 5", due: ptrTime(day("2026-09-07")), wantDueAt: "2026-09-07", wantLabel: "距今 5 天", wantOverdue: false},
		{name: "past 3", due: ptrTime(day("2026-08-30")), wantDueAt: "2026-08-30", wantLabel: "今日已超 3 天", wantOverdue: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dueAt, label, overdue := dueMetrics(tc.due, today)
			if dueAt != tc.wantDueAt || label != tc.wantLabel || overdue != tc.wantOverdue {
				t.Errorf("dueMetrics = (%q,%q,%v) want (%q,%q,%v)",
					dueAt, label, overdue, tc.wantDueAt, tc.wantLabel, tc.wantOverdue)
			}
		})
	}
}

func TestClassifyDateBased(t *testing.T) {
	t.Parallel()

	today := day("2026-09-02")
	cases := []struct {
		name      string
		due       *time.Time
		ownAction bool
		want      BlockerLevel
	}{
		{name: "nil default risk", due: nil, ownAction: false, want: BlockerLevelRisk},
		{name: "overdue not own", due: ptrTime(day("2026-08-30")), ownAction: false, want: BlockerLevelBlocked},
		{name: "overdue own", due: ptrTime(day("2026-08-30")), ownAction: true, want: BlockerLevelOverdue},
		{name: "close range risk", due: ptrTime(day("2026-09-04")), ownAction: false, want: BlockerLevelRisk},
		{name: "far own coord", due: ptrTime(day("2026-12-01")), ownAction: true, want: BlockerLevelCoord},
		{name: "far not own risk", due: ptrTime(day("2026-12-01")), ownAction: false, want: BlockerLevelRisk},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := classifyDateBased(tc.due, today, tc.ownAction); got != tc.want {
				t.Errorf("classifyDateBased=%q want %q", got, tc.want)
			}
		})
	}
}

func TestValidDate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   *time.Time
		want bool
	}{
		{name: "nil", in: nil, want: false},
		{name: "year 1999 dirty", in: ptrTime(day("1999-12-31")), want: false},
		{name: "year 0001 mysql zero", in: ptrTime(day("0001-01-01")), want: false},
		{name: "year 2000 boundary", in: ptrTime(day("2000-01-01")), want: true},
		{name: "year 2025 fresh", in: ptrTime(day("2025-08-28")), want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := validDate(tc.in); got != tc.want {
				t.Errorf("validDate(%v)=%v want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestChooseDeadline(t *testing.T) {
	t.Parallel()

	dev := ptrTime(day("2026-09-10"))
	tf := ptrTime(day("2026-09-12"))
	vf := ptrTime(day("2026-09-14"))
	dd := ptrTime(day("2026-09-20"))
	dirty := ptrTime(day("0001-01-01")) // MySQL '0000-00-00' 解析后值

	cases := []struct {
		name   string
		status string
		d      *time.Time
		tf     *time.Time
		vf     *time.Time
		dd     *time.Time
		want   *time.Time
	}{
		{name: "testing picks testFinish", status: "testing", d: nil, tf: tf, vf: vf, dd: dd, want: tf},
		{name: "waitacceptance picks verifyFinish", status: "waitacceptance", d: nil, tf: tf, vf: vf, dd: dd, want: vf},
		{name: "acceptanced picks deliverDate", status: "acceptanced", d: nil, tf: tf, vf: vf, dd: dd, want: dd},
		{name: "schedule fallback developFinish", status: "schedule", d: dev, tf: nil, vf: nil, dd: nil, want: dev},
		{name: "all nil returns nil", status: "schedule", d: nil, tf: nil, vf: nil, dd: nil, want: nil},
		// 零日期防护：acceptanced 阶段 deliverDate 是 0000-00-00 应回退。
		{name: "acceptanced dirty deliverDate falls back", status: "acceptanced", d: dev, tf: tf, vf: vf, dd: dirty, want: dev},
		{name: "testing dirty testFinish falls back", status: "testing", d: dev, tf: dirty, vf: vf, dd: dd, want: dev},
		{name: "waitacceptance dirty verifyFinish falls back", status: "waitacceptance", d: dev, tf: tf, vf: dirty, dd: dd, want: dev},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := chooseDeadline(tc.status, tc.d, tc.tf, tc.vf, tc.dd)
			if (got == nil) != (tc.want == nil) {
				t.Fatalf("nil mismatch: got nil=%v want nil=%v", got == nil, tc.want == nil)
			}
			if got != nil && !got.Equal(*tc.want) {
				t.Errorf("chooseDeadline=%v want %v", got.Format("2006-01-02"), tc.want.Format("2006-01-02"))
			}
		})
	}
}

func TestPickOwner(t *testing.T) {
	t.Parallel()

	if got := pickOwner("alice", "", "bob", ""); got != "alice" {
		t.Errorf("alice first → got %q", got)
	}
	if got := pickOwner("", "bob", "alice", ""); got != "bob" {
		t.Errorf("bob second → got %q", got)
	}
	if got := pickOwner("", "", "", ""); got != "" {
		t.Errorf("all empty → got %q", got)
	}
}

func TestIsOwnAction(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                              string
		who, rd, qd, bra, assigned        string
		want                              bool
	}{
		{name: "empty who never own", who: "", rd: "alice", qd: "alice", bra: "alice", assigned: "alice", want: false},
		{name: "match rd", who: "alice", rd: "alice", want: true},
		{name: "match assignedTo", who: "alice", assigned: "alice", want: true},
		{name: "no match", who: "alice", rd: "bob", qd: "carol", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isOwnAction(tc.who, tc.rd, tc.qd, tc.bra, tc.assigned)
			if got != tc.want {
				t.Errorf("isOwnAction=%v want %v", got, tc.want)
			}
		})
	}
}

func TestSortBlockers(t *testing.T) {
	t.Parallel()

	in := []BlockerDetail{
		{ID: "1", Kind: "demand", Level: BlockerLevelRisk, Stage: "排期"},
		{ID: "2", Kind: "demand", Level: BlockerLevelBlocked, IsOwnAction: true, Stage: "提测"},
		{ID: "3", Kind: "story", Level: BlockerLevelBlocked, Stage: "提测"},
		{ID: "4", Kind: "demand", Level: BlockerLevelOverdue, IsOwnAction: true, Stage: "联调测试"},
		{ID: "5", Kind: "demand", Level: BlockerLevelCoord, Stage: "澄清"},
	}
	got := append([]BlockerDetail(nil), in...)
	sortBlockers(got)

	wantIDs := []string{"2", "3", "4", "1", "5"}
	gotIDs := make([]string, 0, len(got))
	for _, it := range got {
		gotIDs = append(gotIDs, it.ID)
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Errorf("sort result mismatch\n got: %v\nwant: %v", gotIDs, wantIDs)
	}
}

func TestBlockerLimits(t *testing.T) {
	if BlockerStageLimit <= 0 || BlockerStageLimit >= 100 {
		t.Fatalf("BlockerStageLimit 离群：%d", BlockerStageLimit)
	}
	if BlockerOverallLimit <= BlockerStageLimit {
		t.Fatalf("BlockerOverallLimit(%d) 必须大于 BlockerStageLimit(%d)", BlockerOverallLimit, BlockerStageLimit)
	}
}

func TestBlockerStatusesContainHotStages(t *testing.T) {
	want := map[string]bool{
		"clarify":        true,
		"schedule":       true,
		"developing":     true,
		"testing":        true,
		"waitacceptance": true,
		"acceptanced":    true,
	}
	got := make(map[string]bool, len(blockerStatuses))
	for _, s := range blockerStatuses {
		got[s] = true
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("blockerStatuses 集合已偏离合同\n got: %v\nwant: %v", got, want)
	}
}

func TestBlockerReqValidate(t *testing.T) {
	t.Parallel()
	var req BlockerReq
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("BlockerReq.Validate 必须通过（无入参），got errors: %+v", errs)
	}
}
