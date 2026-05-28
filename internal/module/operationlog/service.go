// =============================================================================
// 文件: internal/module/operationlog/service.go
// 模块: 操作日志
// 类型: readonly
// 职责: 提供操作日志列表查询业务编排。
// 依赖: internal/pkg/pagination
// =============================================================================

package operationlog

import (
	"context"

	"workbench/internal/pkg/pagination"
)

// Service 操作日志业务层。
type Service struct {
	repo *Repo
}

// NewService 创建 Service。
func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

// List 查询操作日志列表。
func (s *Service) List(ctx context.Context, req ListReq) (ListResp, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = pagination.DefaultPageSize
	}
	repoReq := RepoFindAllReq{
		Account:  req.Account,
		Method:   req.Method,
		Path:     req.Path,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	items, total, err := s.repo.FindAll(ctx, repoReq)
	if err != nil {
		return ListResp{}, err
	}

	return ListResp{
		Items: items,
		Pager: pagination.New(total, req.Page, req.PageSize),
	}, nil
}
