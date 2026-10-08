// =============================================================================
// 文件: internal/module/po/clarify_text_test.go
// 模块: PO 工作台
// 类型: test
// 职责: 验证澄清说明的可读文本与未编辑原文保留。
// =============================================================================
package po

import "testing"

func TestClarifyDescriptionText(t *testing.T) {
	cases := []struct{ input, want string }{
		{"<p>abc</p><br>def", "abc\ndef"},
		{"<p>第一段</p><p>第二段<br>下一行</p>", "第一段\n第二段\n下一行"},
		{"&amp;&amp;", "&&"},
		{"&lt;p&gt;abc&lt;/p&gt;", "abc"},
		{"<p>&nbsp;&quot;&#39;&lt; &gt;</p>", "\u00a0\"'< >"},
		{"", ""}, {"  原文\n下一行  ", "  原文\n下一行  "},
		{"1 < 2", "1 < 2"},
		{"<p>文本</p><script>alert(1)</script>", "文本"},
	}
	for _, tc := range cases {
		if got := clarifyDescriptionText(tc.input); got != tc.want {
			t.Errorf("%q: got %q want %q", tc.input, got, tc.want)
		}
	}
}

func TestClarifyDescriptionForSubmit(t *testing.T) {
	for _, original := range []string{"<p>abc</p><br>def", "&amp;&amp;", "", "纯文本\n下一行"} {
		visible := clarifyDescriptionText(original)
		if got := clarifyDescriptionForSubmit(original, visible); got != original {
			t.Errorf("其他字段变化时丢失原文: %q", got)
		}
		changed := visible + "修改"
		if got := clarifyDescriptionForSubmit(original, changed); got != changed {
			t.Errorf("修改说明未保存文本: %q", got)
		}
	}
	if got := clarifyDescriptionForSubmit("<p>abc</p>", ""); got != "" {
		t.Errorf("清空说明: %q", got)
	}
}
