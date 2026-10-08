// =============================================================================
// 文件: internal/pkg/workbenchroles/display_test.go
// 模块: 工作台角色
// 类型: test
// 职责: 验证旧角色显示名映射不影响其他角色及编码。
// =============================================================================

package workbenchroles

import "testing"

func TestDisplayLabel(t *testing.T) {
	for _, tc := range []struct{ code, label, want string }{
		{"po", "旧名称", "产品经理"}, {"custom", "PO", "产品经理"},
		{"custom", "产品负责人", "产品经理"}, {"custom", "产品负责人 (PO)", "产品经理"},
		{"pmo", "PMO", "PMO"}, {"custom", "自定义角色", "自定义角色"},
	} {
		if got := DisplayLabel(tc.code, tc.label); got != tc.want {
			t.Errorf("DisplayLabel(%q, %q) = %q; want %q", tc.code, tc.label, got, tc.want)
		}
	}
	if got := RoleMap()[RolePO]; got.Key != "po" || got.Label != "产品经理" {
		t.Fatalf("unexpected role: %+v", got)
	}
}

func TestSearchAliases(t *testing.T) {
	for _, keyword := range []string{"产品经理", "产品负责人", "PO", "po", "经理", "负责人", "产品"} {
		code, labels := SearchAliases(keyword)
		if code != RolePO || len(labels) != 4 {
			t.Fatalf("%q: %q %v", keyword, code, labels)
		}
	}
	for _, keyword := range []string{"", "  ", "管理员", "PMO"} {
		code, labels := SearchAliases(keyword)
		if code != "" || len(labels) != 0 {
			t.Fatalf("%q: unexpected aliases %q %v", keyword, code, labels)
		}
	}
}
