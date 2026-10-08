// =============================================================================
// 文件: internal/pkg/menu/active.go
// 模块: 基础设施
// 类型: infra
// 职责: 按当前请求地址算出应高亮的二级菜单与其所属一级分组。
// =============================================================================
package menu

import "strings"

// matchItem 返回当前地址命中的二级菜单及其命中的最长前缀长度。
// 规则：地址等于 path 或等于某个 activePaths 时命中；path 以 / 结尾的（如 /demands/）
// 按前缀匹配，其余要求前缀后紧跟 "/" 或地址结束，避免 /board 误命中 /boardx。
func matchItem(items []Menu, path string) (Menu, int) {
	var hit Menu
	best := -1
	for _, item := range items {
		for _, prefix := range append([]string{item.Path}, item.ActivePaths...) {
			p := strings.TrimSpace(prefix)
			if p == "" {
				continue
			}
			if n := prefixLen(path, p); n > best {
				hit, best = item, n
			}
		}
	}
	return hit, best
}

// prefixLen 返回 path 以 prefix 开头时的匹配长度；不匹配返回 0。
func prefixLen(path, prefix string) int {
	if path == prefix {
		return len(prefix)
	}
	if !strings.HasPrefix(path, prefix) {
		return 0
	}
	// 以 / 结尾的地址段视为整段前缀（如 /demands/）；否则必须以分隔符衔接。
	if strings.HasSuffix(prefix, "/") || strings.HasPrefix(path[len(prefix):], "/") {
		return len(prefix)
	}
	return 0
}

// ActiveItemKey 返回当前地址命中的二级菜单 Key，用于二级高亮；未命中返回空串。
func ActiveItemKey(menus []Menu, path string) string {
	path = strings.TrimSpace(path)
	for _, group := range menus {
		if hit, n := matchItem(group.Children, path); n > 0 {
			_ = group
			return hit.Key
		}
	}
	return ""
}

// ActiveGroupKey 返回当前地址命中的二级菜单所属的一级分组 Key，用于窄轨高亮；
// 未命中返回空串，调用方据此回退到用户记忆的分组。
func ActiveGroupKey(menus []Menu, path string) string {
	path = strings.TrimSpace(path)
	for _, group := range menus {
		if _, n := matchItem(group.Children, path); n > 0 {
			return group.Key
		}
	}
	return ""
}
