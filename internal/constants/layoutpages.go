// =============================================================================
// 文件: internal/constants/layoutpages.go
// 模块: 常量
// 职责: 登记无顶栏、无侧栏的全屏 SSR 页面（与 render.Page 的 page 参数一致）。
// =============================================================================

package constants

// hideChromePages 无顶栏、无侧栏的页面模板名（值为 render.Page 的 page 参数）。
var hideChromePages = map[string]struct{}{
	// 模板名常量: {},
}

// PageHidesChrome 判断该 SSR 页面是否隐藏顶栏与侧栏。
func PageHidesChrome(page string) bool {
	_, ok := hideChromePages[page]
	return ok
}
