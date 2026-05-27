// =============================================================================
// 文件: internal/module/cronjob/form.go
// 模块: 定时任务
// 类型: action
// 职责: 定义定时任务列表、日志、启停与触发请求响应结构。
// 依赖: internal/model
//       internal/pkg/pagination
// =============================================================================

package cronjob

import (
	"workbrench/internal/model"
	"workbrench/internal/pkg/pagination"
)

// ListReq 定时任务列表查询请求。
type ListReq struct {
	Page     int `form:"page"`
	PageSize int `form:"pageSize"`
}

// ListResp 定时任务列表响应。
type ListResp struct {
	Items []model.CronJob
	Pager *pagination.Pager
}

// LogListReq 任务日志列表查询请求。
type LogListReq struct {
	JobName  string `form:"jobName"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

// LogListResp 任务日志列表响应。
type LogListResp struct {
	Items []model.CronLog
	Pager *pagination.Pager
}

// ToggleReq 启用/禁用请求。
type ToggleReq struct {
	ID        uint64 `form:"id"`
	IsEnabled bool   `form:"isEnabled"`
}

// TriggerReq 手动触发请求。
type TriggerReq struct {
	ID uint64 `form:"id"`
}

// RepoFindAllReq 仓储层任务列表查询请求。
type RepoFindAllReq struct {
	Page     int
	PageSize int
}

// RepoFindLogsReq 仓储层日志列表查询请求。
type RepoFindLogsReq struct {
	JobName  string
	Page     int
	PageSize int
}
