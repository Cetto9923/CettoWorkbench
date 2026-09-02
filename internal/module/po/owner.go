// =============================================================================
// 文件: internal/module/po/owner.go
// 模块: PO 工作台
// 类型: action
// 职责: 下一责任人推导真源（对齐原型 workbench/rules.DeriveCurrentHandler）。
// 依赖: 无
// =============================================================================

package po

import "strings"

// FormatAccountName 账号 + realname 展示名；realname 已带账号后缀时不重复拼接。
func FormatAccountName(account, realname string) string {
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

// DeriveCurrentHandler 按禅道 status 推导当前办理人（账号 + 显示名）。
//
//	draft/wait → assignedTo
//	active → PM → assignedTo → QD（全空显示 —）
//	clarified → assignedTo → QD（全空显示 —）
//	developing/testing/waitacceptance → RD
//	waitdeliver/acceptanced → 待确认（不用 BRA）
//	其它 → 待分配
//
// bra/braNm 仅保留签名兼容，内部不使用。
func DeriveCurrentHandler(status, assignedTo, qd, rd, bra, pm, pmNm,
	assignedToNm, qdNm, rdNm, braNm string) (account, display string) {
	_ = bra
	_ = braNm

	get := accountNamePair

	switch status {
	case "draft", "wait":
		return get(assignedTo, assignedToNm)
	case "active":
		if acc, disp := pmHandler(pm, pmNm); acc != "" {
			return acc, disp
		}
		if acc, disp := get(assignedTo, assignedToNm); acc != "" {
			return acc, disp
		}
		if acc, disp := get(qd, qdNm); acc != "" {
			return acc, disp
		}
		return "", "—"
	case "clarified":
		if acc, disp := get(assignedTo, assignedToNm); acc != "" {
			return acc, disp
		}
		if acc, disp := get(qd, qdNm); acc != "" {
			return acc, disp
		}
		return "", "—"
	case "developing", "testing", "waitacceptance":
		return get(rd, rdNm)
	case "waitdeliver", "acceptanced":
		return "", "待确认"
	}
	return "", "待分配"
}

func accountNamePair(value, name string) (string, string) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", ""
	}
	n := strings.TrimSpace(name)
	if n == "" {
		return v, v
	}
	suffix := "(" + v + ")"
	if n == v || strings.HasSuffix(n, suffix) || strings.Contains(n, suffix) {
		return v, n
	}
	return v, n + suffix
}

func pmHandler(pm, pmNm string) (string, string) {
	v := strings.TrimSpace(pm)
	if v == "" {
		return "", ""
	}
	if strings.Contains(v, ",") {
		if n := strings.TrimSpace(pmNm); n != "" {
			return v, n
		}
		return v, v
	}
	return accountNamePair(v, pmNm)
}
