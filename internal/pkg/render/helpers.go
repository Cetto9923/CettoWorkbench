// =============================================================================
// 文件: internal/pkg/render/helpers.go
// 模块: render
// 类型: helper
// 职责: 模板 funcmap 工具（asset URL、dict/add/sub/alertClass/toInt）。
// =============================================================================

package render

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (r *Renderer) asset(path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		return path
	}
	rel := strings.TrimPrefix(p, "/")
	rel = strings.TrimPrefix(rel, "static/")
	rel = filepath.FromSlash(rel)
	clean := filepath.Clean(rel)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return path
	}
	full := filepath.Join(r.staticDir, clean)
	base := filepath.Clean(r.staticDir)
	fullClean := filepath.Clean(full)
	relFromBase, err := filepath.Rel(base, fullClean)
	if err != nil || relFromBase == ".." || strings.HasPrefix(relFromBase, ".."+string(filepath.Separator)) {
		return path
	}
	st, err := os.Stat(full)
	if err != nil {
		return path
	}
	v := st.ModTime().Unix()
	if strings.HasPrefix(p, "/static/") {
		return fmt.Sprintf("%s?v=%d", p, v)
	}
	return fmt.Sprintf("/static/%s?v=%d", filepath.ToSlash(clean), v)
}

func dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}
	result := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		result[key] = values[i+1]
	}
	return result, nil
}

func add(a, b interface{}) int {
	return toInt(a) + toInt(b)
}

func sub(a, b interface{}) int {
	return toInt(a) - toInt(b)
}

func alertClass(level interface{}) string {
	s := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", level)))
	switch s {
	case "success":
		return "success"
	case "error":
		return "danger"
	case "warning":
		return "warning"
	case "info":
		return "info"
	default:
		return "secondary"
	}
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int8:
		return int(n)
	case int16:
		return int(n)
	case int32:
		return int(n)
	case int64:
		return int(n)
	case uint:
		return int(n)
	case uint8:
		return int(n)
	case uint16:
		return int(n)
	case uint32:
		return int(n)
	case uint64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	default:
		return 0
	}
}
