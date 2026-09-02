// =============================================================================
// 文件: internal/pkg/zentao/zentao.go
// 模块: 基础设施
// 类型: infra
// 职责: 根据禅道站点地址、requestType 与 m、f 等参数拼接页面链接。
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
	return URLWithBase(strings.TrimRight(zentaoCfg.URL, "/"), m, f, params...)
}

// URLWithBase 使用指定站点前缀拼接禅道页面链接（base 为空时回退全局配置）。
func URLWithBase(base, m, f string, params ...string) string {
	base = strings.TrimRight(base, "/")
	if base == "" {
		base = strings.TrimRight(zentaoCfg.URL, "/")
	}
	if base == "" || m == "" || f == "" {
		return ""
	}

	if isPathInfo(zentaoCfg.RequestType) {
		return pathInfoURL(base, m, f, params...)
	}
	return getURL(base, m, f, params...)
}

func isPathInfo(requestType string) bool {
	return strings.EqualFold(strings.TrimSpace(requestType), "PATH_INFO")
}

func getURL(base, m, f string, params ...string) string {
	query := fmt.Sprintf("m=%s&f=%s", url.QueryEscape(m), url.QueryEscape(f))
	if len(params) > 0 && params[0] != "" {
		query += "&" + params[0]
	}
	return base + indexPath + "?" + query
}

func pathInfoURL(base, m, f string, params ...string) string {
	parts := []string{m, f}
	if len(params) > 0 && params[0] != "" {
		parts = append(parts, pathInfoValues(params[0])...)
	}
	return base + "/" + strings.Join(parts, "-") + ".html"
}

// pathInfoValues 从 GET 风格参数中取出值序列，对齐禅道 createLink 的 PATH_INFO 拼法。
func pathInfoValues(raw string) []string {
	var values []string
	for _, pair := range strings.Split(raw, "&") {
		if pair == "" {
			continue
		}
		_, val, found := strings.Cut(pair, "=")
		if found {
			values = append(values, val)
			continue
		}
		values = append(values, pair)
	}
	return values
}

// DemandViewURL 业需详情页链接。
func DemandViewURL(demandID uint) string {
	if demandID == 0 {
		return ""
	}
	return URL("demand", "view", fmt.Sprintf("demandID=%d", demandID))
}

// DemandViewURLWithBase 使用指定站点前缀拼接业需详情页链接。
func DemandViewURLWithBase(base string, demandID uint) string {
	if demandID == 0 {
		return ""
	}
	return URLWithBase(base, "demand", "view", fmt.Sprintf("demandID=%d", demandID))
}

// StoryViewURL 研发需求详情页链接。
func StoryViewURL(storyID uint) string {
	if storyID == 0 {
		return ""
	}
	return URL("story", "view", fmt.Sprintf("storyID=%d", storyID))
}

// StoryViewURLWithBase 使用指定站点前缀拼接研发需求详情页链接。
func StoryViewURLWithBase(base string, storyID uint) string {
	if storyID == 0 {
		return ""
	}
	return URLWithBase(base, "story", "view", fmt.Sprintf("storyID=%d", storyID))
}

// ProductViewURL 产品概况页链接。
func ProductViewURL(productID uint) string {
	if productID == 0 {
		return ""
	}
	return URL("product", "view", fmt.Sprintf("productID=%d", productID))
}

// ProductViewURLWithBase 使用指定站点前缀拼接产品概况页链接。
func ProductViewURLWithBase(base string, productID uint) string {
	if productID == 0 {
		return ""
	}
	return URLWithBase(base, "product", "view", fmt.Sprintf("productID=%d", productID))
}
