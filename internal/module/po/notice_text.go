// =============================================================================
// 文件: internal/module/po/notice_text.go
// 模块: PO 工作台
// 类型: repo
// 职责: 通知中心富文本纯文本清洗、标签剥离、实体解码与空白归一。
// 依赖: html, strings
// =============================================================================

package po

import (
	"html"
	"strings"
)

// cleanNoticeText 执行 7 步受控文本清洗管道：
// 1. 剥离 script/style/comment 节点（含实体编码形态）；
// 2. 受控单次 HTML 实体解码；
// 3. 防御性再次剥离解码后显露的 script/style/comment；
// 4. 剥离真实 HTML 标签（保留中文尖括号如 <测试>）；
// 5. 清理样式声明与 CSS 字面残留；
// 6. 规范化空白字符；
// 7. 基于 rune 的安全截断。
func cleanNoticeText(raw string, maxRunes int) string {
	if raw == "" {
		return ""
	}
	s := stripScriptAndStyle(raw)
	s = stripEntityScriptAndStyle(s)

	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = html.UnescapeString(s)

	s = stripScriptAndStyle(s)
	s = stripHTMLTags(s)
	s = stripCSSResidue(s)
	s = normalizeNoticeWhitespace(s)

	if maxRunes > 0 {
		runes := []rune(s)
		if len(runes) > maxRunes {
			s = string(runes[:maxRunes]) + "…"
		}
	}
	return s
}

func stripScriptAndStyle(s string) string {
	lower := strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	n := len(s)
	for i < n {
		if strings.HasPrefix(lower[i:], "<script") {
			end := strings.Index(lower[i:], "</script>")
			if end != -1 {
				i += end + len("</script>")
				continue
			}
			break
		}
		if strings.HasPrefix(lower[i:], "<style") {
			end := strings.Index(lower[i:], "</style>")
			if end != -1 {
				i += end + len("</style>")
				continue
			}
			break
		}
		if strings.HasPrefix(lower[i:], "<!--") {
			end := strings.Index(lower[i:], "-->")
			if end != -1 {
				i += end + len("-->")
				continue
			}
			break
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func stripEntityScriptAndStyle(s string) string {
	lower := strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	n := len(s)
	for i < n {
		if strings.HasPrefix(lower[i:], "&lt;script") {
			end := strings.Index(lower[i:], "&lt;/script&gt;")
			if end != -1 {
				i += end + len("&lt;/script&gt;")
				continue
			}
			break
		}
		if strings.HasPrefix(lower[i:], "&lt;style") {
			end := strings.Index(lower[i:], "&lt;/style&gt;")
			if end != -1 {
				i += end + len("&lt;/style&gt;")
				continue
			}
			break
		}
		if strings.HasPrefix(lower[i:], "&lt;!--") {
			end := strings.Index(lower[i:], "--&gt;")
			if end != -1 {
				i += end + len("--&gt;")
				continue
			}
			break
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func stripHTMLTags(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	runes := []rune(value)
	n := len(runes)
	i := 0
	for i < n {
		if runes[i] == '<' {
			if i+1 < n && isHTMLTagStart(runes[i+1:]) {
				end := -1
				for j := i + 1; j < n; j++ {
					if runes[j] == '>' {
						end = j
						break
					}
				}
				if end != -1 {
					i = end + 1
					continue
				}
			}
		}
		builder.WriteRune(runes[i])
		i++
	}
	return builder.String()
}

func isHTMLTagStart(remaining []rune) bool {
	if len(remaining) == 0 {
		return false
	}
	c := remaining[0]
	if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '!' {
		return true
	}
	if c == '/' && len(remaining) > 1 {
		next := remaining[1]
		if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') {
			return true
		}
	}
	return false
}

func stripCSSResidue(s string) string {
	var builder strings.Builder
	builder.Grow(len(s))
	runes := []rune(s)
	n := len(runes)
	i := 0
	for i < n {
		if runes[i] == '{' {
			end := -1
			for j := i + 1; j < n; j++ {
				if runes[j] == '}' {
					end = j
					break
				}
				if runes[j] == '{' {
					break
				}
			}
			if end != -1 {
				block := string(runes[i+1 : end])
				if isCSSBlock(block) {
					builderStr := builder.String()
					trimmed := strings.TrimRight(builderStr, " \t\r\n")
					lastWordStart := strings.LastIndexAny(trimmed, " \t\r\n;>，。！？\n")
					if lastWordStart == -1 {
						lastWordStart = 0
					} else {
						lastWordStart++
					}
					candSelector := strings.TrimSpace(trimmed[lastWordStart:])
					if isPotentialCSSSelector(candSelector) {
						builder.Reset()
						builder.WriteString(trimmed[:lastWordStart])
					}
					i = end + 1
					continue
				}
			}
		}
		builder.WriteRune(runes[i])
		i++
	}
	return builder.String()
}

func isCSSBlock(s string) bool {
	if !strings.Contains(s, ":") {
		return false
	}
	lower := strings.ToLower(s)
	cssProps := []string{
		"color", "font", "margin", "padding", "background",
		"border", "display", "width", "height", "text-align",
		"line-height", "overflow", "cursor", "visibility",
	}
	for _, prop := range cssProps {
		if strings.Contains(lower, prop) {
			return true
		}
	}
	return false
}

func isPotentialCSSSelector(s string) bool {
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	knownSelectors := []string{
		"body", "p", "div", "span", "table", "td", "th", "tr",
		"h1", "h2", "h3", "h4", "h5", "h6", "ul", "ol", "li", "a",
	}
	for _, sel := range knownSelectors {
		if lower == sel || strings.HasPrefix(lower, sel+".") || strings.HasPrefix(lower, sel+"#") {
			return true
		}
	}
	return strings.HasPrefix(lower, ".") || strings.HasPrefix(lower, "#")
}

func normalizeNoticeWhitespace(s string) string {
	s = strings.ReplaceAll(s, "\u00a0", " ")
	return strings.Join(strings.Fields(s), " ")
}
