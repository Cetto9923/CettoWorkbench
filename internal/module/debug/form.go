// =============================================================================
// 文件: internal/module/debug/form.go
// 模块: SQL 性能分析
// 类型: readonly
// 职责: 定义 SQL 性能分析 / API 日志查询请求与响应结构。
// 依赖: 无
// =============================================================================

package debug

import "encoding/json"

// RequestItem 请求级 SQL 汇总记录。
type RequestItem struct {
	Time      string  `json:"time"`
	Level     string  `json:"level"`
	RequestID string  `json:"request_id"`
	Method    string  `json:"method"`
	Route     string  `json:"route"`
	Elapsed   string  `json:"elapsed"`
	ElapsedMS float64 `json:"elapsed_ms"`
	SQLCount  int     `json:"sql_count"`
}

// RequestsReq 请求汇总数据查询参数。
type RequestsReq struct {
	StartDate string `form:"startDate"`
	EndDate   string `form:"endDate"`
}

// RequestsResp 请求汇总数据查询响应。
type RequestsResp struct {
	Requests []RequestItem
}

// RepoFindAllReq 仓储层查询请求汇总数据参数。
type RepoFindAllReq struct {
	StartDate string
	EndDate   string
}

// QueryItem 单条 SQL 查询记录。
type QueryItem struct {
	Time      string  `json:"time"`
	RequestID string  `json:"request_id"`
	Method    string  `json:"method"`
	Route     string  `json:"route"`
	Seq       int     `json:"seq"`
	SQL       string  `json:"sql"`
	Elapsed   string  `json:"elapsed"`
	ElapsedMS float64 `json:"elapsed_ms"`
	Rows      int64   `json:"rows"`
	File      string  `json:"file"`
	Error     string  `json:"error,omitempty"`
}

// QueriesReq 按日 SQL 明细查询参数。
type QueriesReq struct {
	Date  string `form:"date"`
	Limit int    `form:"limit"`
	Route string `form:"route"` // 完整 "METHOD path"，空表示全部
}

// QueriesResp 按日 SQL 明细查询响应。
type QueriesResp struct {
	Date    string
	Total   int64
	Queries []QueryItem
	Routes  []string // 当天 top-K 中去重接口，供下拉
}

// RepoFindQueriesReq 仓储层按日 SQL 明细查询参数。
type RepoFindQueriesReq struct {
	Date  string
	Limit int
	Route string
}

// APILogItem 单条禅道 API 请求日志。
type APILogItem struct {
	Time      string          `json:"time"`
	Elapsed   string          `json:"elapsed"`
	ElapsedMS float64         `json:"elapsed_ms"`
	Method    string          `json:"method"`
	Path      string          `json:"path"`
	URL       string          `json:"url"`
	Status    int             `json:"status"`
	Success   bool            `json:"success"`
	Request   json.RawMessage `json:"request,omitempty"`
	Response  json.RawMessage `json:"response,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// APIEntriesReq 按日 API 日志查询参数。
type APIEntriesReq struct {
	Date     string `form:"date"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

// APIEntriesResp 按日 API 日志查询响应。
type APIEntriesResp struct {
	Date     string
	Total    int64
	Page     int
	PageSize int
	Entries  []APILogItem
}

// RepoFindAPIEntriesReq 仓储层按日 API 日志查询参数。
type RepoFindAPIEntriesReq struct {
	Date     string
	Page     int
	PageSize int
}
