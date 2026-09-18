// =============================================================================
// 文件: internal/module/kanban/service_bizdemand_test.go
// 模块: 工作看板
// 类型: readonly
// 职责: 验证需求与研需编号提取、数量填充逻辑正确性
// 依赖: 无
// =============================================================================

package kanban

import (
	"testing"
)

func TestExtractNumericID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"US63418", "63418"},
		{"#US63418", "63418"},
		{"us-1234", "1234"},
		{"REQ-5678", "5678"},
		{"SUB-999", "999"},
		{"U70849", "70849"},
		{"#U70849", "70849"},
		{"RD-8888", "8888"},
		{"#8888", "8888"},
		{"12345", "12345"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		actual := extractNumericID(tt.input)
		if actual != tt.expected {
			t.Errorf("extractNumericID(%q) = %q, want %q", tt.input, actual, tt.expected)
		}
	}
}
