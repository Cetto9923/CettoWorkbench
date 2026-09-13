// =============================================================================
// 文件: internal/module/metrics/form.go
// 模块: 指标管理 (metrics)
// 类型: form
// 职责: 指标页面的只读契约、查询/响应结构与归一化/校验。
// 依赖: 无
// =============================================================================

package metrics

import (
	"fmt"
	"strings"
)

// MetricItem 是指标管理 / 雷达页面统一返回的展示契约。
// Service 通过 catalog.go 的 evaluateStatus 给出 Status，
// 前端在 workspace-table 中按 status 渲染颜色 token。
type MetricItem struct {
	Code             string `json:"code"`
	Name             string `json:"name"`
	Category         string `json:"category"`         // 5 分类中文 label（ValidCategories）
	Unit             string `json:"unit"`             // "%" / "个" / "—" 等展示单位
	DataSource       string `json:"dataSource"`       // 禅道 zt_story / DevOps / 门禁 等人类可读来源
	Period           string `json:"period"`           // 日 / 周 / 月 / 实时
	Value            string `json:"value"`            // 当前值（rate 已拼百分比，count 为数字；无数据为 "—"）
	Target           string `json:"target"`           // 目标文本，如 "≥90%" / "≤5"
	WarningThreshold string `json:"warningThreshold"` // 关注阈值（与 Unit 一致）
	DangerThreshold  string `json:"dangerThreshold"`  // 风险阈值（与 Unit 一致）
	GoodDirection    string `json:"goodDirection"`    // up = 越大越好；down = 越小越好
	Status           string `json:"status"`           // normal / warn / danger / unknown
	Enabled          bool   `json:"enabled"`          // 是否启用（catalog.Enabled）
	OwnerRole        string `json:"ownerRole"`        // PO / SM / PMO / 测试
	Description      string `json:"description"`      // 指标口径（人类可读）
	Formula          string `json:"formula"`          // 派生公式 / SQL 摘要
	LastCalcTime     string `json:"lastCalcTime"`     // 本次计算时间（Service 层填写）
	Source           string `json:"source"`           // 兼容旧 JS 的原始 SQL 来源 label
}

// ManageSummary 是指标管理页面顶部的 4 项摘要。
//
//	TotalItems    = 全量指标条数（在 Service 拼装前固定 12）
//	CategoryCount = 各分类指标条数 map
//	WithSnapshot  = value 不为 "—" 的指标条数（有真实数据）
//	Abnormal      = status 为 warn 或 danger 的指标条数
//
// 由 Service.computeSummary(items) 一次循环计算。
type ManageSummary struct {
	TotalItems    int64            `json:"totalItems"`
	CategoryCount map[string]int64 `json:"categoryCount"`
	WithSnapshot  int64            `json:"withSnapshot"`
	Abnormal      int64            `json:"abnormal"`
}

// ManageListReq 是 /metrics/api 的查询参数。
// 前端走 GET + query string；状态白名单在 Validate() 强制。
type ManageListReq struct {
	Category string `form:"category"`
	Status   string `form:"status"`
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

// Normalize 在 Handler 与 Service 边界各调用一次；把空字符串归位、page 上下限收紧。
func (r *ManageListReq) Normalize() {
	r.Category = strings.TrimSpace(r.Category)
	r.Status = strings.TrimSpace(r.Status)
	r.Keyword = strings.TrimSpace(r.Keyword)
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 || r.PageSize > 100 {
		r.PageSize = 15
	}
}

// Validate 在 Normalize 之后调用；Category 由 Service 复用 ValidCategories 白名单校验，
// Status 显式收紧到 4 个枚举值（含 ""）。
func (r *ManageListReq) Validate() error {
	switch r.Status {
	case "", "normal", "warn", "danger", "unknown":
	default:
		return fmt.Errorf("metrics: invalid status %q", r.Status)
	}
	return nil
}

// ManageListResp 是指标管理页面 List 接口的完整响应。
// Items 是当前页（不超过 PageSize）；Filter 回显请求参数便于前端 reset；
// Total 是 filter 后的总条数，前端分页器据此渲染。
type ManageListResp struct {
	Summary  ManageSummary `json:"summary"`
	Items    []MetricItem  `json:"items"`
	Filter   ManageListReq `json:"filter"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}
