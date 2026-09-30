// =============================================================================
// 文件: internal/module/po/service_detail_action_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证详情页 B1 状态标志（Flags）、原因文案（FlagNotice）与验收主操作改判逻辑。
// =============================================================================

package po

import (
	"context"
	"testing"

	"workbench/internal/model"
	"workbench/internal/module/po/primaryaction"
	"workbench/internal/pkg/perm"
)

// 1. 标志解析与原因文案。
func TestDeriveDetailFlags(t *testing.T) {
	tests := []struct {
		name string
		row  *DemandDetailRow
		want DemandFlags
		msg  string
	}{
		{"nil_row", nil, DemandFlags{}, ""},
		{"normal", &DemandDetailRow{Hang: "0", IsChange: "", IsReturned: "0"}, DemandFlags{}, ""},
		{"hang", &DemandDetailRow{Hang: "1", IsChange: "", IsReturned: "0"}, DemandFlags{Hang: true}, FlagNoticePaused},
		{"changing", &DemandDetailRow{Hang: "0", IsChange: "changing", IsReturned: "0"}, DemandFlags{Changing: true}, FlagNoticePaused},
		{"returning", &DemandDetailRow{Hang: "0", IsChange: "", IsReturned: "1"}, DemandFlags{Returning: true}, FlagNoticePaused},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flags, notice := deriveDetailFlags(tc.row)
			if flags != tc.want {
				t.Fatalf("flags = %+v, want %+v", flags, tc.want)
			}
			if notice != tc.msg {
				t.Fatalf("notice = %q, want %q", notice, tc.msg)
			}
		})
	}
}

// 2. 标志存在时详情主操作置空（buildPrimaryActionForDetail 返回 None）。
func TestBuildPrimaryActionForDetail_FlagsYieldsNone(t *testing.T) {
	svc := &DetailService{}
	actor := &model.User{Account: "alice"}

	// 挂起需求即便处于 waitacceptance 也无主操作
	rowHang := &DemandDetailRow{ID: 101, Status: "waitacceptance", AssignedTo: "alice", Hang: "1"}
	paHang := svc.buildPrimaryActionForDetail(context.Background(), actor, rowHang)
	if paHang.Key != "" || paHang.Enabled {
		t.Fatalf("hang demand should yield None, got %+v", paHang)
	}

	// 变更中需求无主操作
	rowChanging := &DemandDetailRow{ID: 102, Status: "waitacceptance", AssignedTo: "alice", IsChange: "changing"}
	paChanging := svc.buildPrimaryActionForDetail(context.Background(), actor, rowChanging)
	if paChanging.Key != "" || paChanging.Enabled {
		t.Fatalf("changing demand should yield None, got %+v", paChanging)
	}

	// 退回中需求无主操作
	rowReturning := &DemandDetailRow{ID: 103, Status: "waitacceptance", AssignedTo: "alice", IsReturned: "1"}
	paReturning := svc.buildPrimaryActionForDetail(context.Background(), actor, rowReturning)
	if paReturning.Key != "" || paReturning.Enabled {
		t.Fatalf("returning demand should yield None, got %+v", paReturning)
	}
}

// 3. 验收主操作改判：assignedTo == 当前用户 派生「验收」。
func TestBuildPrimaryActionForDetail_WaitAcceptance_AssignedTo(t *testing.T) {
	svc := &DetailService{}
	actor := &model.User{Account: "alice"}
	row := &DemandDetailRow{
		ID:         200,
		Status:     "waitacceptance",
		AssignedTo: "alice",
		Accepter:   "bob", // accepter 不是本人，但 assignedTo 是本人
	}

	pa := svc.buildPrimaryActionForDetail(context.Background(), actor, row)
	if pa.Key != string(primaryaction.KeyAcceptDone) {
		t.Fatalf("assignedTo == actor should yield accept, got key=%q", pa.Key)
	}
	if !pa.Enabled {
		t.Fatalf("expected action to be enabled, got reason=%q", pa.Reason)
	}
	if pa.URL != "/demands/200/acceptance" {
		t.Fatalf("expected url /demands/200/acceptance, got %q", pa.URL)
	}
}

// 4. 验收主操作改判：assignedTo != 当前用户 且 accepter == 当前用户，不得派生「验收」，应派生「催办验收」。
func TestBuildPrimaryActionForDetail_WaitAcceptance_AccepterNotAssignedTo(t *testing.T) {
	svc := &DetailService{}
	actor := &model.User{Account: "bob"}
	// 赋予催办权限
	ctx := perm.WithGranted(context.Background(), map[string]bool{perm.PoHomeList.String(): true})

	row := &DemandDetailRow{
		ID:         201,
		Status:     "waitacceptance",
		AssignedTo: "alice", // assignedTo 不是本人
		Accepter:   "bob",   // 虽为 accepter，但不是 assignedTo
	}

	pa := svc.buildPrimaryActionForDetail(ctx, actor, row)
	if pa.Key == string(primaryaction.KeyAcceptDone) {
		t.Fatalf("non-assignedTo user must NOT get accept action, got %+v", pa)
	}
	if pa.Key != string(primaryaction.KeyRemindAccept) {
		t.Fatalf("non-assignedTo user should get remind_accept action, got key=%q", pa.Key)
	}
	if !pa.Enabled {
		t.Fatalf("expected urge action to be enabled with perm, got reason=%q", pa.Reason)
	}
	if pa.URL != "/demands/201/urge" {
		t.Fatalf("expected url /demands/201/urge, got %q", pa.URL)
	}
}

// 5. 验收主操作改判：非 assignedTo 且无催办权限时，主操作为禁用的「催办验收」。
func TestBuildPrimaryActionForDetail_WaitAcceptance_NoUrgePerm(t *testing.T) {
	svc := &DetailService{}
	actor := &model.User{Account: "charlie"}
	ctx := context.Background() // 无任何权限

	row := &DemandDetailRow{
		ID:         202,
		Status:     "waitacceptance",
		AssignedTo: "alice",
		Accepter:   "charlie",
	}

	pa := svc.buildPrimaryActionForDetail(ctx, actor, row)
	if pa.Key != string(primaryaction.KeyRemindAccept) {
		t.Fatalf("want remind_accept, got key=%q", pa.Key)
	}
	if pa.Enabled {
		t.Fatal("urge action should be disabled without perm")
	}
	if pa.Reason != "当前用户没有催办验收权限" {
		t.Fatalf("unexpected reason: %q", pa.Reason)
	}
}

// 6. B1 责任字段与提出部门映射测试。
func TestDemandSummary_B1_ResponsibilityFields(t *testing.T) {
	row := &DemandDetailRow{
		ID:              300,
		Name:            "责任字段测试需求",
		AssignedTo:      "dev_01",
		AssignedToName:  "开发者一",
		RD:              "rd_01",
		RDName:          "研发负责人一",
		LeadDept:        "dept_10",
		LeadDeptName:    "网络金融部",
		TeamGroup:       "tg_05",
		TeamGroupName:   "零售敏捷组",
		ProposeDept:     "dept_03",
		ProposeDeptName: "个人业务部",
		OriginatorDept:  "渠道部",
		Hang:            "0",
		IsChange:        "",
		IsReturned:      "0",
	}

	svc := &DetailService{}
	summary := svc.buildSummary(row)

	if summary.AssignedTo != "开发者一" || summary.AssignedToName != "开发者一" {
		t.Fatalf("assignedTo want 开发者一, got assignedTo=%q, assignedToName=%q", summary.AssignedTo, summary.AssignedToName)
	}
	if summary.CurrentOwner != "开发者一" {
		t.Fatalf("currentOwner want 开发者一, got %q", summary.CurrentOwner)
	}
	if summary.AcceptOwner != "研发负责人一" || summary.RDName != "研发负责人一" {
		t.Fatalf("acceptOwner/rdName want 研发负责人一, got acceptOwner=%q, rdName=%q", summary.AcceptOwner, summary.RDName)
	}
	if summary.LeadDept != "网络金融部" {
		t.Fatalf("leadDept want 网络金融部, got %q", summary.LeadDept)
	}
	if summary.TeamGroup != "零售敏捷组" {
		t.Fatalf("teamGroup want 零售敏捷组, got %q", summary.TeamGroup)
	}
	if summary.ProposeDept != "个人业务部" || summary.ProposerDept != "个人业务部" {
		t.Fatalf("proposeDept want 个人业务部, got proposeDept=%q, proposerDept=%q", summary.ProposeDept, summary.ProposerDept)
	}
	if summary.Flags.Hang || summary.Flags.Changing || summary.Flags.Returning {
		t.Fatalf("expected all flags false, got %+v", summary.Flags)
	}
	if summary.FlagNotice != "" {
		t.Fatalf("expected empty flagNotice, got %q", summary.FlagNotice)
	}
}
