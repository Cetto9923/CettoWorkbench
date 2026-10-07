// =============================================================================
// 文件: internal/module/po/service_homecounts.go
// 模块: PO 工作台
// 类型: service
// 职责: 首页责任与关联芯片首屏计数，与点击后的列表同源。
// =============================================================================
package po

import "context"

func (s *Service) countHomeOwnership(ctx context.Context, account string) (KPICounts, error) {
	counts := KPICounts{}
	for _, focus := range []struct {
		name  string
		count *int64
	}{
		{"my_managed", &counts.MyManaged}, {"my_related", &counts.MyRelated},
	} {
		count, err := s.repo.CountHomeFocus(ctx, account, DemandsReq{Status: "all", Focus: focus.name})
		if err != nil {
			return counts, err
		}
		*focus.count = count
	}
	return counts, nil
}
