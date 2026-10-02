// =============================================================================
// 文件: internal/module/po/owner_test.go
// 模块: PO 工作台
// 类型: action
// 职责: DeriveCurrentHandler 当前办理人推导的单元测试（验收 A-2 提测口径）。
// 依赖: 标准库 testing
// =============================================================================

package po

import "testing"

// 验收 A-2：提测(developing)的列表「当前负责人」必须与 my_action 办理人同口径——
// BRA 优先；BRA 未指派时显示负责人，不再走 RD 另显一套。
func TestDeriveCurrentHandlerDevelopingFollowsHandlerRule(t *testing.T) {
	const (
		rdDisplay = "章倩倩(771349)"
		braName   = "章倩倩(003030)"
		ownName   = "姜亚梅(771349)"
	)
	cases := []struct {
		name        string
		bra         string
		assignedTo  string
		wantAccount string
		wantDisplay string
	}{
		{name: "BRA 优先于负责人", bra: "003030", assignedTo: "771349", wantAccount: "003030", wantDisplay: braName},
		{name: "BRA 空时显示负责人", bra: "", assignedTo: "771349", wantAccount: "771349", wantDisplay: ownName},
		{name: "BRA 与负责人皆空", bra: "", assignedTo: "", wantAccount: "", wantDisplay: ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			acc, disp := DeriveCurrentHandler("developing", c.assignedTo, "", "771349", c.bra, "", "",
				ownName, "", rdDisplay, braName)
			if acc != c.wantAccount || disp != c.wantDisplay {
				t.Fatalf("developing handler = (%q, %q), want (%q, %q)", acc, disp, c.wantAccount, c.wantDisplay)
			}
		})
	}
}

// 未回归：testing/waitacceptance 仍以 RD 为当前负责人，本轮只动 developing。
func TestDeriveCurrentHandlerTestingStillUsesRD(t *testing.T) {
	for _, status := range []string{"testing", "waitacceptance"} {
		acc, disp := DeriveCurrentHandler(status, "771349", "", "771349", "003030", "", "",
			"姜亚梅(771349)", "", "章倩倩(771349)", "章倩倩(003030)")
		if acc != "771349" || disp != "章倩倩(771349)" {
			t.Fatalf("%s handler = (%q, %q), want (771349, 章倩倩(771349))", status, acc, disp)
		}
	}
}
