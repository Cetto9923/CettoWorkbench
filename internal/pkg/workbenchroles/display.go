// =============================================================================
// 文件: internal/pkg/workbenchroles/display.go
// 模块: 工作台角色
// 类型: helper
// 职责: 仅规范后端输出的角色显示名，保留角色编码与存储值。
// =============================================================================

package workbenchroles

// DisplayLabel 兼容组织角色表中的旧显示名，不修改数据库。
func DisplayLabel(code, label string) string {
	if code == RolePO || label == "产品负责人" || label == "PO" || label == "产品负责人 (PO)" {
		return "产品经理"
	}
	return label
}
