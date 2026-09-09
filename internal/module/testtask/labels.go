// =============================================================================
// 文件: internal/module/testtask/labels.go
// 模块: 提测办理
// 类型: action
// 职责: 禅道 status → 工作台展示文案；人员展示名解析。
// 依赖: 无
// =============================================================================

package testtask

import "strings"

// 业需 zt_demand.status → 中文（与首页 home.js ZENTAO_STATUS_LABELS 对齐）。
var demandStatusLabels = map[string]string{
	"draft":          "暂存",
	"wait":           "待评审",
	"active":         "已评审",
	"clarified":      "已澄清",
	"changed":        "已变更",
	"developing":     "开发中",
	"testing":        "测试中",
	"waitacceptance": "待验收",
	"acceptanced":    "已验收",
	"waitdeliver":    "待交付",
	"delivered":      "已交付",
	"released":       "已发布",
	"closed":         "已关闭",
	"suspended":      "已挂起",
	"refuse":         "已驳回",
}

// DemandStatusLabel 需求原始状态中文；空为 —，未知码回退原文。
func DemandStatusLabel(status string) string {
	st := strings.ToLower(strings.TrimSpace(status))
	if st == "" {
		return "—"
	}
	if label, ok := demandStatusLabels[st]; ok {
		return label
	}
	return strings.TrimSpace(status)
}

// FormatPersonName 账号 + 姓名 →「姓名(账号)」；无姓名回退账号。
func FormatPersonName(account, realname string) string {
	v := strings.TrimSpace(account)
	if v == "" {
		return ""
	}
	n := strings.TrimSpace(realname)
	if n == "" {
		return v
	}
	suffix := "(" + v + ")"
	if n == v || strings.HasSuffix(n, suffix) || strings.Contains(n, suffix) {
		return n
	}
	return n + suffix
}

func dash(v string) string {
	if strings.TrimSpace(v) == "" {
		return "—"
	}
	return strings.TrimSpace(v)
}

// lookupDisplay 从 AccountDisplayMap 取「姓名(账号)」；无映射回退账号。
func lookupDisplay(displayMap map[string]string, account string) string {
	acc := strings.TrimSpace(account)
	if acc == "" {
		return "—"
	}
	if displayMap != nil {
		if d := strings.TrimSpace(displayMap[acc]); d != "" {
			return d
		}
	}
	return acc
}

// BuildContextResp 将查询行装配为前端上下文响应。
// stage 固定为「提测」；handlerAccount/handlerName 为当前登录用户。
func BuildContextResp(row DemandContextRow, displayMap map[string]string, handlerAccount, handlerName string) *ContextResp {
	return &ContextResp{
		DemandID:       row.ID,
		Title:          strings.TrimSpace(row.Name),
		Stage:          "提测",
		RawStatus:      DemandStatusLabel(row.Status),
		MainSystemName: dash(row.MainSystemName),
		EstimateLaunch: dash(row.EstimateLaunch),
		BRAName:        lookupDisplay(displayMap, row.BRA),
		RDName:         lookupDisplay(displayMap, row.RD),
		QDName:         lookupDisplay(displayMap, row.QD),
		HandlerName:    dash(FormatPersonName(handlerAccount, handlerName)),
	}
}
