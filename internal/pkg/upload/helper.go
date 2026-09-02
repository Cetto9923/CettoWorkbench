// =============================================================================
// 文件: internal/pkg/upload/helper.go
// 模块: 基础设施
// 类型: infra
// 职责: 提供上传相关辅助函数。
// 依赖: 无
// =============================================================================

package upload

import (
	"path/filepath"
	"strings"
)

// Ext 安全获取扩展名（小写，防路径穿越）。
func Ext(filename string) string {
	base := filepath.Base(strings.TrimSpace(filename))
	if base == "." || base == "/" || base == `\` {
		return ""
	}
	return strings.ToLower(filepath.Ext(base))
}
