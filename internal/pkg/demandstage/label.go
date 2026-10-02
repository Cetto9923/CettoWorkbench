// Package demandstage maps ZenTao demand stage/status to workbench value-stream
// codes and to query-list labels. Map returns the home-page stage code only;
// Label returns the query list's Chinese status label. Callers MUST use the
// matching API: Map for the value stream, Label for the query list.
package demandstage

import "strings"

// statusCodes / stageCodes 把禅道原始值归一到首页价值流编码。
// 编码取自 po.valueStreamStages，developing 归 developing（首页叫「提测」），
// waitacceptance 与 acceptanced 保持两段，released 归 released；中文名在该表里，本包不产出。
var statusCodes = map[string]string{
	"closed": "closed", "released": "released", "waitdeliver": "publish",
	"delivered": "publish", "acceptanced": "acceptanced", "waitacceptance": "waitacceptance",
	"testing": "testing", "developing": "developing", "clarified": "schedule",
	"active": "clarify", "clarify": "clarify",
	"draft": "accept", "refuse": "accept", "wait": "accept",
}

var stageCodes = map[string]string{
	"wait": "accept", "inroadmap": "clarify", "clarify": "clarify",
	"incharter": "schedule", "schedule": "schedule", "developing": "developing",
	"delivering": "testing", "testing": "testing", "delivered": "publish", "closed": "closed",
}

// Map returns the PO value-stream code for a ZenTao stage/status pair.
// status 未命中时回退 stage；两者都未命中返回 "unknown"。
func Map(stage, status string) string {
	if code, ok := statusCodes[strings.ToLower(strings.TrimSpace(status))]; ok {
		return code
	}
	if code, ok := stageCodes[strings.ToLower(strings.TrimSpace(stage))]; ok {
		return code
	}
	return "unknown"
}

// Label returns the Chinese stage label used by the query module list views.
// developing → "研发中" (intentional divergence from Map's "developing").
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
