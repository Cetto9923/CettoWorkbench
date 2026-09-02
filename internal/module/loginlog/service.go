// =============================================================================
// 文件: internal/module/loginlog/service.go
// 模块: 登录日志
// 类型: readonly
// 职责: 提供登录日志列表查询业务编排。
// 依赖: internal/model
//       internal/module/loginlog/repo.go
// =============================================================================

package loginlog

import (
	"context"

	"workbench/internal/model"
)

// Service 登录日志业务层。
type Service struct {
	repo *Repo
}

// NewService 创建 Service。
func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

// ListResp 列表响应。
type ListResp struct {
	Items []model.LoginLog
	Total int64
}

// List 查询登录日志列表。
func (s *Service) List(ctx context.Context, actor *model.User, req ListReq) (ListResp, error) {
	_ = actor
	req.Normalize()
	startAt, err := req.ParsedStartAt()
	if err != nil {
		return ListResp{}, err
	}
	endAt, err := req.ParsedEndAt()
	if err != nil {
		return ListResp{}, err
	}

	items, total, err := s.repo.FindAll(ctx, RepoFindAllReq{
		Account:  req.Account,
		IP:       req.IP,
		StartAt:  startAt,
		EndAt:    endAt,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return ListResp{}, err
	}
	return ListResp{Items: items, Total: total}, nil
}
