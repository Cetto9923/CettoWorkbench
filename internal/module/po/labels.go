// =============================================================================
// 文件: internal/module/po/labels.go
// 模块: PO 工作台
// 类型: action
// 职责: 业需类别/来源/状态中文标签（对齐禅道 lang/zh-cn）。
// 依赖: 无
// =============================================================================

package po

import "strings"

func demandCategoryLabel(raw string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	labels := map[string]string{
		"feature":     "功能",
		"interface":   "接口",
		"performance": "性能",
		"safe":        "安全",
		"experience":  "体验",
		"improve":     "改进",
		"other":       "其他",
		// 兼容历史/扩展取值
		"request":    "业务需求",
		"business":   "业务需求",
		"research":   "调研需求",
		"bug":        "BUG",
		"tecopt":     "技术优化",
		"datacg":     "数据变更",
		"datachange": "数据变更",
		"dataexport": "数据导出",
	}
	if v, ok := labels[key]; ok {
		return v
	}
	if strings.TrimSpace(raw) == "" {
		return "—"
	}
	return raw
}

func demandSourceLabel(raw string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	labels := map[string]string{
		"customer":   "客户",
		"user":       "用户",
		"po":         "产品经理",
		"market":     "市场",
		"service":    "客服",
		"operation":  "运营",
		"support":    "技术支持",
		"competitor": "竞争对手",
		"partner":    "合作伙伴",
		"dev":        "开发人员",
		"tester":     "测试人员",
		"bug":        "Bug",
		"feedback":   "反馈",
		"other":      "其他",
	}
	if v, ok := labels[key]; ok {
		return v
	}
	if strings.TrimSpace(raw) == "" {
		return "—"
	}
	return raw
}

func demandStatusLabel(raw string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	labels := map[string]string{
		"wait":           "待评审",
		"active":         "已评审",
		"clarified":      "已澄清",
		"developing":     "开发中",
		"testing":        "测试中",
		"waitacceptance": "待验收",
		"acceptanced":    "已验收",
		"waitdeliver":    "待交付",
		"released":       "已发布",
		"closed":         "已关闭",
		"refuse":         "已驳回",
		"draft":          "暂存",
		"canceled":       "已取消",
		"cancelled":      "已取消",
		"suspended":      "已挂起",
		"reviewing":      "评审中",
		"changed":        "已变更",
		"postponed":      "已延期",
	}
	if v, ok := labels[key]; ok {
		return v
	}
	if strings.TrimSpace(raw) == "" {
		return "—"
	}
	return raw
}

func demandValueStageLabel(status string) string {
	key := strings.ToLower(strings.TrimSpace(status))
	if key == "wait" || key == "draft" || key == "refuse" {
		return "待受理"
	}
	if label := valueStreamLabelForStatus(key); label != "" {
		return label
	}
	return demandStatusLabel(status)
}

func formatDemandPri(pri string) string {
	p := strings.TrimSpace(pri)
	if p == "" {
		return "P2"
	}
	if strings.HasPrefix(strings.ToUpper(p), "P") {
		return strings.ToUpper(p)
	}
	return "P" + p
}

func dashOr(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "—"
	}
	return v
}
