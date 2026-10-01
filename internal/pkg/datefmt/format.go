// Package datefmt renders dates for human display.
//
// Shared by po / query / metrics so zero-date semantics stay identical:
// 零日期 →「未设置」，空 / nil / 无效 →「—」，有效日期 → yyyy-mm-dd。
package datefmt

import (
	"strings"
	"time"
)

// Layout 是全站日期展示统一格式（AGENTS.md 规则 17）。
const Layout = "2006-01-02"

// 零日期与空值的展示文案，全站唯一事实源。
const (
	Unset = "未设置" // 零日期：IsZero / 年份 < 2000 / 0000-00-00
	Empty = "—"   // 空、nil、无法解析
)

// Date 把可选时间渲染成展示文本；nil →「—」，零日期 →「未设置」。
func Date(t *time.Time) string {
	if t == nil {
		return Empty
	}
	if t.IsZero() || t.Year() < 2000 {
		return Unset
	}
	return t.Format(Layout)
}

// Raw 把数据库原始日期串（可能带时间后缀）渲染成展示文本，口径同 Date。
func Raw(raw string) string {
	v := strings.TrimSpace(raw)
	if len(v) >= len(Layout) {
		v = v[:len(Layout)]
	}
	if t, err := time.ParseInLocation(Layout, v, time.Local); err == nil {
		if t.Year() < 2000 {
			return Unset
		}
		return v
	}
	if strings.HasPrefix(v, "0000-00-00") {
		return Unset
	}
	return Empty
}
