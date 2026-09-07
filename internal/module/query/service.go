// 需求查询 Service：薄壳，负责把 ListReq 校验后透传给 Repo。
// 不持有业务规则；Repo 是禅道只读视图的真实来源。
package query

import "context"

type Service struct{ repo *Repo }

func NewService(repo *Repo) *Service { return &Service{repo: repo} }

// List 校验请求并返回 Repo 查询结果；任何阶段都把字符串字段 Normalize 后再下推。
func (s *Service) List(ctx context.Context, req ListReq) (ListResp, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return ListResp{}, err
	}
	return s.repo.List(ctx, req)
}
