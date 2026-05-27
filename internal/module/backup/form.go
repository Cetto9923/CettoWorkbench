// =============================================================================
// 文件: internal/module/backup/form.go
// 模块: 数据库备份
// 类型: action
// 职责: 定义备份列表请求响应结构。
// 依赖: internal/model
//       internal/pkg/pagination
// =============================================================================

package backup

import (
	"workbrench/internal/model"
	"workbrench/internal/pkg/pagination"
)

// ListReq 备份记录列表查询请求。
type ListReq struct {
	Page     int `form:"page"`
	PageSize int `form:"pageSize"`
}

// ListResp 备份记录列表查询响应。
type ListResp struct {
	Items []model.BackupRecord
	Pager *pagination.Pager
}

// CreateReq 手动创建备份请求。
type CreateReq struct{}

// CreateResp 手动创建备份响应。
type CreateResp struct {
	Filename string
}

// DownloadReq 下载备份请求。
type DownloadReq struct {
	ID uint64 `form:"id"`
}

// RepoFindAllReq 仓储层分页查询请求。
type RepoFindAllReq struct {
	Page     int
	PageSize int
}
