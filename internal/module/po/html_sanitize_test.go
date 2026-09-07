// =============================================================================
// 文件: internal/module/po/html_sanitize_test.go
// 模块: PO 工作台
// 类型: test
// 职责: F02 富文本净化（bluemonday 白名单）单元测试：覆盖 <script>、
//       事件属性、javascript:/data: URL、<svg> 等危险形态；保留合法
//       富文本（段落、列表、链接）。
// =============================================================================

package po

import (
	"strings"
	"testing"
)

func TestSanitizeRichTextHTML_DropsDangerousContent(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		denied []string
	}{
		{
			name:   "script tag dropped",
			input:  `<p>hello</p><script>alert(1)</script>`,
			denied: []string{"<script", "alert(1)"},
		},
		{
			name:   "img onerror stripped",
			input:  `<img src="/x" onerror="alert('xss')" />`,
			denied: []string{"onerror", "alert"},
		},
		{
			name:   "anchor javascript: stripped",
			input:  `<a href="javascript:alert(1)">click</a>`,
			denied: []string{"javascript:", "alert"},
		},
		{
			name:   "svg onload dropped",
			input:  `<svg onload="alert(1)"></svg>`,
			denied: []string{"<svg", "onload", "alert"},
		},
		{
			name:   "iframe dropped",
			input:  `<iframe src="https://evil"></iframe>`,
			denied: []string{"<iframe"},
		},
		{
			name:   "style tag dropped",
			input:  `<style>body{display:none}</style>`,
			denied: []string{"<style", "display:none"},
		},
		{
			name:   "object/embed dropped",
			input:  `<object data="x"></object><embed src="y">`,
			denied: []string{"<object", "<embed"},
		},
		{
			name:   "data url on anchor denied",
			input:  `<a href="data:text/html,<script>alert(1)</script>">x</a>`,
			denied: []string{"data:", "<script", "alert"},
		},
		{
			name:   "form / input dropped",
			input:  `<form action="x"><input name="y" value="z" /></form>`,
			denied: []string{"<form", "<input"},
		},
		{
			name:   "vbscript: stripped",
			input:  `<a href="vbscript:msgbox(1)">x</a>`,
			denied: []string{"vbscript:"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := SanitizeRichTextHTML(tc.input)
			for _, frag := range tc.denied {
				if strings.Contains(strings.ToLower(out), strings.ToLower(frag)) {
					t.Errorf("sanitized output still contains %q\ninput:  %s\noutput: %s", frag, tc.input, out)
				}
			}
		})
	}
}

func TestSanitizeRichTextHTML_PreservesSafeContent(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		mustAll []string
	}{
		{
			name:    "paragraph and bold preserved",
			input:   `<p>Hello <strong>world</strong></p>`,
			mustAll: []string{"<p>", "<strong>", "Hello", "world", "</strong>", "</p>"},
		},
		{
			name:    "ul / li preserved",
			input:   `<ul><li>one</li><li>two</li></ul>`,
			mustAll: []string{"<ul>", "<li>", "one", "two", "</li>", "</ul>"},
		},
		{
			name:    "table preserved",
			input:   `<table><thead><tr><th>A</th></tr></thead><tbody><tr><td>1</td></tr></tbody></table>`,
			mustAll: []string{"<table>", "<thead>", "<th>", "A", "<td>", "1"},
		},
		{
			name:    "https link preserved",
			input:   `<a href="https://example.com/path" target="_blank" rel="noopener">go</a>`,
			mustAll: []string{"<a", "https://example.com/path", "go"},
		},
		{
			name:    "mailto link preserved",
			input:   `<a href="mailto:foo@bar.com">mail</a>`,
			mustAll: []string{"<a", "mailto:foo@bar.com", "mail"},
		},
		{
			name:    "heading preserved",
			input:   `<h1>Title</h1><h3>Sub</h3>`,
			mustAll: []string{"<h1>", "Title", "<h3>", "Sub"},
		},
		{
			name:    "img with http src preserved",
			input:   `<img src="https://example.com/a.png" alt="logo" />`,
			mustAll: []string{"<img", "https://example.com/a.png", "alt=\"logo\""},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := SanitizeRichTextHTML(tc.input)
			for _, frag := range tc.mustAll {
				if !strings.Contains(out, frag) {
					t.Errorf("expected %q in sanitized output\ninput:  %s\noutput: %s", frag, tc.input, out)
				}
			}
		})
	}
}

func TestSanitizeRichTextHTML_EmptyInput(t *testing.T) {
	if got := SanitizeRichTextHTML(""); got != "" {
		t.Fatalf("empty input should return empty output, got %q", got)
	}
}
