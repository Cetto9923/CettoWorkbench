// =============================================================================
// 文件: internal/pkg/zentao/zentao.go
// 模块: 基础设施
// 类型: infra
// 职责: 根据禅道站点地址与 m、f 等参数拼接页面链接。
// 依赖: internal/config
// =============================================================================

package zentao

import (
	"fmt"
	"net/url"
	"strings"

	"workbench/internal/config"
)

const indexPath = "/index.php"

var zentaoCfg config.ZentaoConfig

// SetConfig 注册禅道配置，应在应用启动时调用（bootstrap 中传入 cfg.Zentao）。
func SetConfig(cfg config.ZentaoConfig) {
	zentaoCfg = cfg
}

// URL 拼接禅道页面链接，站点前缀取自 config.Zentao.URL（zentao.url）。
func URL(m, f string, params ...string) string {
	base := strings.TrimRight(zentaoCfg.URL, "/")
	if base == "" || m == "" || f == "" {
		return ""
	}

	query := fmt.Sprintf("m=%s&f=%s", url.QueryEscape(m), url.QueryEscape(f))

	if len(params) > 0 && params[0] != "" {
		query += "&" + params[0]
	}

	return base + "/index.php?" + query
}
