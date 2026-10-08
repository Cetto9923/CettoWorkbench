// =============================================================================
// 文件: internal/pkg/workbenchroles/display.go
// 模块: 工作台角色
// 类型: helper
// 职责: 仅规范后端输出的角色显示名，保留角色编码与存储值。
// =============================================================================

package workbenchroles

import "strings"

var productManagerAliases = []string{"产品经理", "产品负责人", "PO", "产品负责人 (PO)"}

// DisplayLabel 兼容组织角色表中的旧显示名，不修改数据库。
func DisplayLabel(code, label string) string {
	if code == RolePO {
		return "产品经理"
	}
	for _, alias := range productManagerAliases {
		if label == alias {
			return "产品经理"
		}
	}
	return label
}

// SearchAliases 返回匹配关键词的角色编码与兼容名称，不改变存储值。
func SearchAliases(keyword string) (string, []string) {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return "", nil
	}
	for _, alias := range productManagerAliases {
		if strings.Contains(strings.ToLower(alias), keyword) {
			return RolePO, append([]string(nil), productManagerAliases...)
		}
	}
	return "", nil
}
