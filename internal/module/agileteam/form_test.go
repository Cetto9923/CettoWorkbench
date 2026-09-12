// =============================================================================
// 文件: internal/module/agileteam/form_test.go
// 模块: 敏捷小组治理
// 类型: action
// 职责: 调整请求校验单测。
// =============================================================================

package agileteam

import (
	"strings"
	"testing"
)

func TestSubmitAdjustmentReqValidate(t *testing.T) {
	t.Parallel()
	req := SubmitAdjustmentReq{TeamgroupID: 1, Items: []AdjustItemReq{
		{Account: "alice", ActionType: ActionAdd, Role: "研发", AvailableHours: 7},
	}}
	if errs := req.Validate(); len(errs) != 0 {
		t.Fatalf("expected valid, got %v", errs)
	}

	bad := SubmitAdjustmentReq{TeamgroupID: 0, Items: nil}
	if errs := bad.Validate(); len(errs) < 2 {
		t.Fatalf("expected teamgroup + items errors, got %v", errs)
	}

	dup := SubmitAdjustmentReq{TeamgroupID: 2, Items: []AdjustItemReq{
		{Account: "a", ActionType: ActionAdd},
		{Account: "a", ActionType: ActionRemove},
	}}
	if errs := dup.Validate(); len(errs) == 0 {
		t.Fatal("expected duplicate account error")
	}

	badType := SubmitAdjustmentReq{TeamgroupID: 2, Items: []AdjustItemReq{
		{Account: "b", ActionType: "move"},
	}}
	if errs := badType.Validate(); len(errs) == 0 {
		t.Fatal("expected invalid actionType error")
	}
}

func TestRejectReqValidate(t *testing.T) {
	t.Parallel()
	if errs := (&RejectReq{AdjustmentID: 1, Reason: "不合规"}).Validate(); len(errs) != 0 {
		t.Fatalf("expected ok, got %v", errs)
	}
	if errs := (&RejectReq{AdjustmentID: 1}).Validate(); len(errs) == 0 {
		t.Fatal("expected reason required")
	}
	long := strings.Repeat("驳", 501)
	if errs := (&RejectReq{AdjustmentID: 1, Reason: long}).Validate(); len(errs) == 0 {
		t.Fatal("expected reason length limit")
	}
}

func TestUpdateBasicReqValidate(t *testing.T) {
	t.Parallel()
	ok := UpdateBasicReq{ID: 1, Name: "组织变革团队"}
	if errs := ok.Validate(); len(errs) != 0 {
		t.Fatalf("expected valid, got %v", errs)
	}
	empty := UpdateBasicReq{ID: 1, Name: "  "}
	if errs := empty.Validate(); len(errs) == 0 {
		t.Fatal("expected empty name error")
	}
	jsLogo := UpdateBasicReq{ID: 1, Name: "团队", Logo: "javascript:alert(1)"}
	if errs := jsLogo.Validate(); len(errs) == 0 {
		t.Fatal("expected javascript logo rejection")
	}
}

func TestBuildPendingSummaryCounts(t *testing.T) {
	t.Parallel()
	adj := &Adjustment{ID: 9, AdjustNo: "ADJ-001", Status: StatusPending, SubmittedBy: "po1", Reason: "扩编"}
	items := []AdjustmentItem{
		{ActionType: ActionAdd},
		{ActionType: ActionAdd},
		{ActionType: ActionRemove},
		{ActionType: ActionRoleChange},
	}
	ps := buildPendingSummary(adj, items)
	if ps.AddCount != 2 || ps.RemoveCount != 1 || ps.ChangeCount != 1 {
		t.Fatalf("counts mismatch: %+v", ps)
	}
	if ps.AdjustNo != "ADJ-001" || ps.AdjustmentID != 9 {
		t.Fatalf("header mismatch: %+v", ps)
	}
}

func TestListReqNormalizePageSize(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   int
		want int
	}{
		{"zero defaults 20", 0, 20},
		{"negative defaults 20", -10, 20},
		{"preset 20 stays 20", 20, 20},
		{"preset 50 stays 50", 50, 50},
		{"preset 100 stays 100", 100, 100},
		{"custom 15 stays 15", 15, 15},
		{"custom 30 stays 30", 30, 30},
		{"custom 86 stays 86", 86, 86},
		{"below floor clamps to 5", 3, 5},
		{"way above ceiling clamps to 200", 9999, 200},
		{"at floor 5 stays 5", 5, 5},
		{"at ceiling 200 stays 200", 200, 200},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := ListReq{PageSize: c.in}
			r.Normalize()
			if r.PageSize != c.want {
				t.Fatalf("in=%d want=%d got=%d", c.in, c.want, r.PageSize)
			}
		})
	}
}
