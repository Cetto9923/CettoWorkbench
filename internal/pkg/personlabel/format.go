// Package personlabel formats account + realname display strings.
//
// Shared by po / testtask / user so suffix-dedupe semantics stay identical.
package personlabel

import "strings"

// Format 账号 + realname →「姓名(账号)」；无 realname 回退账号。
// realname 已等于账号、或以「(账号)」结尾、或已包含「(账号)」时不重复拼接。
func Format(account, realname string) string {
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
