// Package demandstage maps ZenTao demand stage/status to workbench display keys
// and Chinese labels. PO value-stream (Map) and query list (Label) intentionally
// diverge on developing → "提测" vs "研发中"; callers MUST use the matching API.
package demandstage

import "strings"

// Map returns (code, label) for PO value-stream display.
// developing → ("submittest", "提测"), aligned with home stage cards.
func Map(stage, status string) (code, label string) {
	st := strings.ToLower(strings.TrimSpace(status))
	switch st {
	case "closed":
		return "closed", "已关闭"
	case "released":
		return "greyverify", "生产验证"
	case "waitdeliver", "delivered":
		return "publish", "发布"
	case "acceptanced":
		return "acceptance", "已验收"
	case "waitacceptance":
		return "acceptance", "待验收"
	case "testing":
		return "testing", "测试中"
	case "developing":
		return "submittest", "提测"
	case "clarified":
		return "schedule", "已排期"
	case "active", "clarify":
		return "clarify", "澄清中"
	case "draft", "refuse", "wait":
		return "accept", "已受理"
	}

	sg := strings.ToLower(strings.TrimSpace(stage))
	switch sg {
	case "wait":
		return "accept", "已受理"
	case "inroadmap", "clarify":
		return "clarify", "澄清中"
	case "incharter", "schedule":
		return "schedule", "排期中"
	case "developing":
		return "submittest", "提测"
	case "delivering", "testing":
		return "testing", "测试中"
	case "delivered":
		return "publish", "发布"
	case "closed":
		return "closed", "已关闭"
	default:
		return "unknown", "未知"
	}
}

// Label returns the Chinese stage label used by the query module list views.
// developing → "研发中" (intentional divergence from Map's "提测").
func Label(stage, status string) string {
	st := strings.ToLower(strings.TrimSpace(status))
	switch st {
	case "closed":
		return "已关闭"
	case "released":
		return "生产验证"
	case "waitdeliver", "delivered":
		return "发布"
	case "acceptanced":
		return "已验收"
	case "waitacceptance":
		return "待验收"
	case "testing":
		return "测试中"
	case "developing":
		return "研发中"
	case "clarified":
		return "已排期"
	case "active", "clarify":
		return "澄清中"
	case "draft", "refuse", "wait":
		return "已受理"
	}

	sg := strings.ToLower(strings.TrimSpace(stage))
	switch sg {
	case "wait":
		return "已受理"
	case "inroadmap", "clarify":
		return "澄清中"
	case "incharter", "schedule":
		return "排期中"
	case "developing":
		return "研发中"
	case "delivering", "testing":
		return "测试中"
	case "delivered":
		return "发布"
	case "closed":
		return "已关闭"
	}
	return "未知"
}
