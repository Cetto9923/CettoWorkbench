// =============================================================================
// 文件: internal/module/sqlperf/service.go
// 模块: SQL 性能分析
// 类型: readonly
// 职责: 编排 SQL 性能分析数据查询。
// 依赖: 无
// =============================================================================

package debug

import (
	"context"
	"strings"
)

// Service SQL 性能分析业务层。
type Service struct {
	repo *Repo
}

// NewService 创建 Service。
func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

// Requests 查询请求汇总数据；全量查询，不接收 actor。
func (s *Service) Requests(ctx context.Context, req RequestsReq) (RequestsResp, error) {
	items, _, err := s.repo.FindAll(ctx, RepoFindAllReq{
		StartDate: strings.TrimSpace(req.StartDate),
		EndDate:   strings.TrimSpace(req.EndDate),
	})
	if err != nil {
		return RequestsResp{}, err
	}

	return RequestsResp{Requests: items}, nil
}
