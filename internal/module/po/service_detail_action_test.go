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
		{"hang", &DemandDetailRow{Hang: "1", IsChange: "", IsReturned: "0"}, DemandFlags{Hang: true}, "需求已挂起，主操作已暂停，请在禅道处理"},
		{"changing", &DemandDetailRow{Hang: "0", IsChange: "changing", IsReturned: "0"}, DemandFlags{Changing: true}, "需求变更中，主操作已暂停，请在禅道处理"},
		{"returning", &DemandDetailRow{Hang: "0", IsChange: "", IsReturned: "1"}, DemandFlags{Returning: true}, "需求退回中，主操作已暂停，请在禅道处理"},
		{"hang_and_changing", &DemandDetailRow{Hang: "1", IsChange: "changing", IsReturned: "0"}, DemandFlags{Hang: true, Changing: true}, "需求已挂起、变更中，主操作已暂停，请在禅道处理"},
		{"hang_and_returning", &DemandDetailRow{Hang: "1", IsChange: "", IsReturned: "1"}, DemandFlags{Hang: true, Returning: true}, "需求已挂起、退回中，主操作已暂停，请在禅道处理"},
		{"changing_and_returning", &DemandDetailRow{Hang: "0", IsChange: "changing", IsReturned: "1"}, DemandFlags{Changing: true, Returning: true}, "需求变更中、退回中，主操作已暂停，请在禅道处理"},
		{"all_three", &DemandDetailRow{Hang: "1", IsChange: "changing", IsReturned: "1"}, DemandFlags{Hang: true, Changing: true, Returning: true}, "需求已挂起、变更中、退回中，主操作已暂停，请在禅道处理"},
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

// 2. 标志存在时详情主操作置空（PrimaryAction 为 nil）。
func TestBuildPrimaryActionForDetail_FlagsYieldsNone(t *testing.T) {
	svc := &DetailService{}
	actor := &model.User{Account: "alice"}

	cases := []struct {
		name  string
		flags DemandFlags
	}{
		{"hang", DemandFlags{Hang: true}},
		{"changing", DemandFlags{Changing: true}},
		{"returning", DemandFlags{Returning: true}},
		{"combined", DemandFlags{Hang: true, Changing: true}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := &DemandDetailRow{ID: 101, Status: "waitacceptance", AssignedTo: "alice"}
			resp := &DemandDetailResp{Summary: DemandSummary{Flags: tc.flags}}
			svc.applyPrimaryActionForDetail(context.Background(), actor, row, resp, true)
			if resp.PrimaryAction != nil {
				t.Fatalf("flags %+v should yield nil PrimaryAction, got %+v", tc.flags, resp.PrimaryAction)
			}
		})
	}
}

func TestAcceptancePrimaryActionOwnerAndPermission(t *testing.T) {
	for _, tc := range []struct {
		rd, assigned, account string
		grant, handler        bool
		key                   primaryaction.ActionKey
		enabled               bool
	}{
		{"alice", "bob", "alice", true, true, primaryaction.KeyAcceptDone, true},
		{"", "alice", "alice", false, true, primaryaction.KeyAcceptDone, false},
		{"alice", "bob", "bob", true, true, primaryaction.KeyRemindAccept, true},
		{"alice", "bob", "charlie", false, false, primaryaction.KeyRemindAccept, false},
	} {
		ctx := context.Background()
		if tc.grant {
			ctx = perm.WithGranted(ctx, map[string]bool{perm.PoHomeList.String(): true})
		}
		pa := deriveDemandPrimaryAction(ctx, &model.User{Account: tc.account}, primaryActionFactsDemand{ObjectID: 200, Kind: primaryaction.ObjectBusinessDemand, Stage: primaryaction.StageAcceptance, IsAcceptanceOwner: acceptanceOwner(tc.rd, tc.assigned) == tc.account, IsHandler: tc.handler})
		if pa.Key != string(tc.key) || pa.Enabled != tc.enabled {
			t.Fatalf("case %+v: %+v", tc, pa)
		}
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
