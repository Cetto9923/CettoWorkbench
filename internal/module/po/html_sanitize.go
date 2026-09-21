// =============================================================================
// 文件: internal/module/po/html_sanitize.go
// 模块: PO 工作台
// 类型: utility
// 职责: 详情富文本（spec / verifyPlan）服务器侧白名单净化（F02 富文本
//       注入修复的 API 边界防线）。使用 bluemonday 走 allowlist，确保
//       即便渲染层被绕过，HTTP 出口也不带 <script>/on*/javascript: 等。
// 依赖: github.com/microcosm-cc/bluemonday
// =============================================================================

package po

import "github.com/microcosm-cc/bluemonday"

var richTextPolicy = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()

	// 基础块级与文本标签
	p.AllowElements("p", "br", "strong", "em", "u", "s",
		"ol", "ul", "li",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"blockquote", "pre", "code",
		"table", "thead", "tbody", "tfoot", "tr", "td", "th",
		"span", "div")

	// 链接：限定为 http/https/mailto 协议。
	p.AllowAttrs("href").OnElements("a")
	p.AllowURLSchemes("http", "https", "mailto")

	// 链接 target / rel（仅 a 标签）。
	p.AllowAttrs("target", "rel", "title").OnElements("a")

	// 图片：限定为 http/https 协议；data: URL 在图片 src 上被 bluemonday 默认拒收，
	// 这里允许 src 上的 data:image/* 子集需要单独过滤，保持默认安全策略不变。
	p.AllowAttrs("src", "alt", "title", "width", "height").OnElements("img")

	// 表格排版属性
	p.AllowAttrs("colspan", "rowspan", "align", "valign").OnElements("th", "td")
	p.AllowAttrs("align", "valign").OnElements("tr")
	p.AllowAttrs("align", "width", "border", "cellpadding", "cellspacing").OnElements("table")

	return p
}()

// SanitizeRichTextHTML 对详情富文本字段做白名单净化。空输入返回空串。
//
// 保留：p / br / strong / em / u / ol / ul / li / h1-h6 / blockquote / pre / code /
// table 系 / span / div / a (http|https|mailto) / img (http|https|data:image)。
// 拒绝：script / iframe / object / embed / style / link / meta / form / input /
// button / svg / 任何 on* 事件属性 / javascript: / vbscript:。
func SanitizeRichTextHTML(input string) string {
	if input == "" {
		return ""
	}
	return richTextPolicy.Sanitize(input)
}
