// =============================================================================
// 文件: internal/module/schedule/form_filter_compat.go
// 模块: 排期工作台
// 类型: action
// 职责: 快捷筛选与窗口互斥、参数兼容及角标联动辅助。
// 依赖: internal/module/schedule/form.go
// =============================================================================

package schedule

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// NormalizeDemandFilterWithWindows 规范化快捷筛选参数，支持窗口参数互斥规则。
// 当带了窗口筛选时，后端不再叠加「待排期（未绑窗口）」条件，自动视为「全部未关闭」。
func NormalizeDemandFilterWithWindows(filter, windows string) string {
	hasWindows := len(ParseCommaSeparatedUints(windows)) > 0
	trimmed := strings.TrimSpace(filter)
	if hasWindows {
		if trimmed == "" || trimmed == FilterUnscheduled {
			return FilterAllOpen
		}
	}
	return NormalizeDemandFilter(filter)
}

// enrichFilterCountsReqFromContext 从请求 Query 或 Referer 补充缺省的筛选参数（小组/产品/窗口/阶段）。
func enrichFilterCountsReqFromContext(c *gin.Context, req *FilterCountsReq) {
	if req == nil {
		return
	}
	// 1. 若 Query 中带有 stage，兼容映射到 stages
	if strings.TrimSpace(req.Stages) == "" && strings.TrimSpace(req.Stage) != "" {
		req.Stages = strings.TrimSpace(req.Stage)
	}

	// 2. 若关键筛选全为空，尝试从 Referer Header 中反解析当前排期页上下文
	if strings.TrimSpace(req.Groups) == "" && strings.TrimSpace(req.Windows) == "" &&
		strings.TrimSpace(req.Products) == "" && strings.TrimSpace(req.Stages) == "" {
		ref := c.Request.Header.Get("Referer")
		if ref != "" {
			if u, err := url.Parse(ref); err == nil {
				q := u.Query()
				if v := q.Get("groups"); v != "" {
					req.Groups = strings.TrimSpace(v)
				}
				if v := q.Get("windows"); v != "" {
					req.Windows = strings.TrimSpace(v)
				}
				if v := q.Get("products"); v != "" {
					req.Products = strings.TrimSpace(v)
				}
				if v := q.Get("stages"); v != "" {
					req.Stages = strings.TrimSpace(v)
				} else if v := q.Get("stage"); v != "" {
					req.Stages = strings.TrimSpace(v)
				}
			}
		}
	}
}

// NormalizeDemandFilter 规范化快捷筛选参数，默认待排期。
func NormalizeDemandFilter(filter string) string {
	switch strings.TrimSpace(filter) {
	case FilterAllOpen, FilterUnscheduled, FilterPendingReview, FilterManagerReviewing, FilterClosed:
		return strings.TrimSpace(filter)
	case "suspended":
		return FilterAllOpen
	default:
		return FilterUnscheduled
	}
}

// NormalizeFilterCountsTab 规范化角标统计 tab，默认业务需求。
func NormalizeFilterCountsTab(tab string) string {
	switch strings.ToLower(strings.TrimSpace(tab)) {
	case FilterCountsTabStory:
		return FilterCountsTabStory
	default:
		return FilterCountsTabDemand
	}
}

func filterCountReuseMatches(reuseFilter, field string) bool {
	return strings.TrimSpace(reuseFilter) == field
}

func applyFilterCountReuse(counts *FilterCounts, reuseFilter string, reuseTotal int64) {
	if counts == nil {
		return
	}
	switch strings.TrimSpace(reuseFilter) {
	case FilterAllOpen:
		counts.AllOpen = reuseTotal
	case FilterUnscheduled:
		counts.Unscheduled = reuseTotal
	case FilterPendingReview:
		counts.PendingReview = reuseTotal
	case FilterManagerReviewing:
		counts.ManagerReviewing = reuseTotal
	case FilterClosed:
		counts.Closed = reuseTotal
	case FilterCountReuseSuspended:
		counts.Suspended = reuseTotal
	}
}

// advancedFilterParamsFromCountsReq 从角标请求参数中提取高级筛选参数。
func advancedFilterParamsFromCountsReq(req FilterCountsReq) advancedFilterParams {
	stages := ParseCommaSeparatedStages(req.Stages)
	if len(stages) == 0 && strings.TrimSpace(req.Stage) != "" {
		stages = ParseCommaSeparatedStages(req.Stage)
	}
	return advancedFilterParams{
		groupIDs:   ParseCommaSeparatedUints(req.Groups),
		productIDs: ParseCommaSeparatedUints(req.Products),
		windowIDs:  ParseCommaSeparatedUints(req.Windows),
		stages:     stages,
	}
}
