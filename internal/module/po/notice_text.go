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
	"strconv"
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

// parseNoticeSubject 解析通知主题中的对象类型与编号（如 "CHARTER #529 云销管理平台" -> "charter", 529, "云销管理平台"）。
func parseNoticeSubject(subject string) (string, int64, string) {
	s := strings.TrimSpace(subject)
	if s == "" {
		return "", 0, ""
	}
	parts := strings.SplitN(s, "#", 2)
	if len(parts) == 2 {
		typePart := strings.ToLower(strings.TrimSpace(parts[0]))
		for {
			if strings.HasPrefix(typePart, "[") {
				if idx := strings.Index(typePart, "]"); idx >= 0 {
					typePart = strings.TrimSpace(typePart[idx+1:])
					continue
				}
			}
			if strings.HasPrefix(typePart, "【") {
				if idx := strings.Index(typePart, "】"); idx >= 0 {
					typePart = strings.TrimSpace(typePart[idx+len("】"):])
					continue
				}
			}
			break
		}
		rest := strings.TrimSpace(parts[1])
		numEnd := 0
		for numEnd < len(rest) && rest[numEnd] >= '0' && rest[numEnd] <= '9' {
			numEnd++
		}
		if numEnd > 0 {
			if id, err := strconv.ParseInt(rest[:numEnd], 10, 64); err == nil && id > 0 {
				cleanTitle := strings.TrimSpace(rest[numEnd:])
				cleanTitle = strings.TrimLeft(cleanTitle, "-:：· ")
				normType := normalizeNoticeObjectType(typePart)
				if normType != "" {
					return normType, id, cleanTitle
				}
			}
		}
	}
	return "", 0, s
}

func normalizeNoticeObjectType(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	switch raw {
	case "charter", "立项", "章程", "项目章程":
		return "charter"
	case "guideline", "建设指引", "指引", "buildguideline":
		return "guideline"
	case "review", "评审", "需求评审":
		return "review"
	case "approval", "审批":
		return "approval"
	case "planchange", "计划变更":
		return "planchange"
	case "story", "研需", "研发需求":
		return "story"
	case "demand", "需求", "业务需求":
		return "demand"
	case "task", "任务":
		return "task"
	case "bug", "缺陷":
		return "bug"
	case "feedback", "反馈":
		return "feedback"
	case "project", "项目":
		return "project"
	case "testtask", "testcase", "测试单", "测试":
		return "testtask"
	case "risk", "风险":
		return "risk"
	case "issue", "问题":
		return "issue"
	case "release", "发布":
		return "release"
	case "kanbancard", "看板", "看板卡片":
		return "kanbancard"
	case "ticket", "工单":
		return "ticket"
	default:
		return ""
	}
}

// cleanNoticeSummary 剔除正文中重复的标题与 ZenTao 邮件模板路径前缀，提取有效通知摘要。
func cleanNoticeSummary(title, rawData string, maxRunes int) string {
	if rawData == "" {
		return ""
	}
	cleaned := cleanNoticeText(rawData, 0)
	if cleaned == "" {
		return ""
	}

	title = strings.TrimSpace(title)
	baseTitle := strings.TrimSpace(strings.Split(title, " - ")[0])
	_, _, cleanTitle := parseNoticeSubject(title)

	// 依次尝试剥离开头与标题、主标题、去对象前缀标题重合的片段
	for _, t := range []string{title, baseTitle, cleanTitle} {
		if t != "" && strings.HasPrefix(cleaned, t) {
			cleaned = strings.TrimSpace(cleaned[len(t):])
		}
	}

	// 剥离 ZenTao 邮件模板中的系统路径/面包屑前缀（如 "CRCB CHARTER #529 ..."）
	if strings.HasPrefix(cleaned, "CRCB") {
		cleaned = strings.TrimSpace(cleaned[4:])
		for _, t := range []string{title, baseTitle, cleanTitle} {
			if t != "" && strings.HasPrefix(cleaned, t) {
				cleaned = strings.TrimSpace(cleaned[len(t):])
			}
		}
		if _, _, rest := parseNoticeSubject(cleaned); rest != "" && rest != cleaned {
			cleaned = rest
			for _, t := range []string{title, baseTitle, cleanTitle} {
				if t != "" && strings.HasPrefix(cleaned, t) {
					cleaned = strings.TrimSpace(cleaned[len(t):])
				}
			}
		}
	}

	// 剥离开头遗留的标点符号与无语义前缀
	cleaned = strings.TrimLeft(cleaned, " \t\r\n-:：·●>，,。；;")
	cleaned = strings.TrimSpace(cleaned)

	// 若去重后内容与标题相同、无信息增量或过短，则置空不硬塞副标题
	if cleaned == "" || cleaned == title || cleaned == baseTitle || cleaned == cleanTitle {
		return ""
	}

	if maxRunes > 0 {
		runes := []rune(cleaned)
		if len(runes) > maxRunes {
			cleaned = string(runes[:maxRunes]) + "…"
		}
	}
	return cleaned
}
