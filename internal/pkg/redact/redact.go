// =============================================================================
// 文件: internal/pkg/redact/redact.go
// 模块: 基础设施
// 类型: infra
// 职责: API 与操作日志共用的敏感字段脱敏。
// 依赖: encoding/json
// =============================================================================

package redact

import (
	"encoding/json"
	"regexp"
	"strings"
)

var sqlStrings = regexp.MustCompile(`'(?:''|\\.|[^'\\])*'|"(?:""|\\.|[^"\\])*"`)

func Sensitive(key string) bool {
	key = strings.ToLower(strings.NewReplacer("_", "", "-", "", " ", "").Replace(key))
	return strings.Contains(key, "password") || strings.Contains(key, "token") || strings.Contains(key, "secret") || key == "authorization" || key == "cookie"
}

func JSON(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	if len(raw) > 64*1024 {
		return "[响应过长，省略]"
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return "[非 JSON 内容，省略]"
	}
	filter(value)
	return value
}

func filter(value any) {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if Sensitive(key) {
				value[key] = "[FILTERED]"
			} else {
				filter(child)
			}
		}
	case []any:
		for _, child := range value {
			filter(child)
		}
	}
}

// SQL removes string values even when GORM Scan bypasses ParamsFilter.
func SQL(query string) string {
	return sqlStrings.ReplaceAllString(query, "?")
}
