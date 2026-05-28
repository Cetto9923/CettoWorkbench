// =============================================================================
// 文件: internal/module/operationlog/form.go
// 模块: 操作日志
// 类型: readonly
// 职责: 定义操作日志列表查询请求与响应结构。
// 依赖: internal/model
//       internal/pkg/pagination
// =============================================================================

package operationlog

import (
	"workbench/internal/model"
	"workbench/internal/pkg/pagination"
)

// ListReq 操作日志列表查询请求。
type ListReq struct {
	Account  string `form:"account"`
	Method   string `form:"method"`
	Path     string `form:"path"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

// ListResp 操作日志列表查询响应。
type ListResp struct {
	Items []*model.OperationLog
	Pager *pagination.Pager
}

// RepoFindAllReq 操作日志仓储层列表查询请求。
type RepoFindAllReq struct {
	Account  string
	Method   string
	Path     string
	Page     int
	PageSize int
}
