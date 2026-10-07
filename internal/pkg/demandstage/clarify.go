// =============================================================================
// 文件: internal/pkg/demandstage/clarify.go
// 模块: 需求阶段
// 职责: 对齐禅道 demand::isClickable 的澄清状态条件。
// =============================================================================
package demandstage

func CanClarify(status, hang string, isParent bool) bool {
	if hang == "1" || isParent {
		return false
	}
	switch status {
	case "active", "clarified", "developing", "testing", "waitacceptance", "acceptanced":
		return true
	}
	return false
}
