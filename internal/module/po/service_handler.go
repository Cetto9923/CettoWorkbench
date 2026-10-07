// =============================================================================
// 文件: internal/module/po/service_handler.go
// 模块: PO 工作台
// 类型: service
// 职责: 验收负责人判定。
// =============================================================================
package po

import "strings"

func acceptanceOwner(rd, assignedTo string) string {
	if owner := strings.TrimSpace(rd); owner != "" {
		return owner
	}
	return strings.TrimSpace(assignedTo)
}
