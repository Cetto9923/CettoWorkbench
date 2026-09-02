// =============================================================================
// 文件: internal/module/debug/form.go
// 模块: SQL 性能分析
// 类型: readonly
// 职责: 定义 SQL 性能分析查询请求与响应结构。
// 依赖: 无
// =============================================================================

package debug

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
