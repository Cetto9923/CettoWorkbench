// =============================================================================
// 文件: internal/module/sqlperf/service.go
// 模块: SQL 性能分析
// 类型: readonly
// 职责: 编排 SQL 性能分析与 API 日志数据查询。
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

// Queries 查询指定日期的 SQL 明细（按耗时降序）；不接收 actor。
func (s *Service) Queries(ctx context.Context, req QueriesReq) (QueriesResp, error) {
	date := strings.TrimSpace(req.Date)
	items, total, routes, err := s.repo.FindQueries(ctx, RepoFindQueriesReq{
		Date:  date,
		Limit: req.Limit,
		Route: strings.TrimSpace(req.Route),
	})
	if err != nil {
		return QueriesResp{}, err
	}
	return QueriesResp{
		Date:    date,
		Total:   total,
		Queries: items,
		Routes:  routes,
	}, nil
}

// APIEntries 查询指定日期的禅道 API 日志（按时间倒序分页）；不接收 actor。
func (s *Service) APIEntries(ctx context.Context, req APIEntriesReq) (APIEntriesResp, error) {
	date := strings.TrimSpace(req.Date)
	items, total, page, pageSize, err := s.repo.FindAPIEntries(ctx, RepoFindAPIEntriesReq{
		Date:     date,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return APIEntriesResp{}, err
	}
	return APIEntriesResp{
		Date:     date,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Entries:  items,
	}, nil
}
