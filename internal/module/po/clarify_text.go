// =============================================================================
// 文件: internal/module/po/clarify_text.go
// 模块: PO 工作台
// 类型: utility
// 职责: 将澄清说明转换为可编辑文本，并保留未编辑的原文。
// =============================================================================
package po

import (
	"html"
	"strings"

	htmlparser "golang.org/x/net/html"
)

func clarifyDescriptionText(input string) string {
	source := input
	if !strings.Contains(source, "<") {
		source = html.UnescapeString(source)
	}
	tokenizer := htmlparser.NewTokenizer(strings.NewReader(source))
	var out strings.Builder
	hasTags := false
	skip := false
	for {
		kind := tokenizer.Next()
		if kind == htmlparser.ErrorToken {
			break
		}
		token := tokenizer.Token()
		if kind == htmlparser.TextToken && !skip {
			out.WriteString(token.Data)
			continue
		}
		if !map[htmlparser.TokenType]bool{htmlparser.StartTagToken: true, htmlparser.EndTagToken: true, htmlparser.SelfClosingTagToken: true}[kind] {
			continue
		}
		hasTags = true
		if map[string]bool{"script": true, "style": true}[token.Data] {
			skip = kind == htmlparser.StartTagToken
			continue
		}
		if !skip && strings.Contains("|p|br|div|li|tr|h1|h2|h3|h4|h5|h6|blockquote|pre|", "|"+token.Data+"|") && !strings.HasSuffix(out.String(), "\n") {
			out.WriteByte('\n')
		}
	}
	if !hasTags {
		return html.UnescapeString(input)
	}
	return strings.Trim(out.String(), "\n")
}

func clarifyDescriptionForSubmit(original, submitted string) string {
	if submitted == clarifyDescriptionText(original) {
		return original
	}
	return submitted
}
