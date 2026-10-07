// =============================================================================
// 文件: internal/module/po/service_handler.go
// 模块: PO 工作台
// 类型: service
// 职责: 验收负责人及当前账号。
// =============================================================================
package po

import (
	"strings"
	"workbench/internal/model"
)

func acceptanceOwner(rd, assignedTo string) string {
	if owner := strings.TrimSpace(rd); owner != "" {
		return owner
	}
	return strings.TrimSpace(assignedTo)
}
func actorAccount(actor *model.User) string {
	if actor == nil {
		return ""
	}
	return strings.TrimSpace(actor.Account)
}
