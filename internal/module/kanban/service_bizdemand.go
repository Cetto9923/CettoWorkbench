// =============================================================================
// 文件: internal/module/kanban/service_bizdemand.go
// 模块: 工作看板
// 类型: readonly
// 职责: 复用首页价值流 Demands，过滤为业务需求列表。
// 依赖: internal/model
//       internal/module/po
// =============================================================================

package kanban

import (
	"context"
	"strings"

	"workbench/internal/model"
	"workbench/internal/module/po"
)

const bizDemandPageSize = 100

// ListValueStreamBizDemands 拉取首页价值流「全部」阶段中的业务需求（排除研需）。
func (s *Service) ListValueStreamBizDemands(ctx context.Context, actor *model.User) (ListBizDemandsResp, error) {
	if s.poSvc == nil {
		return ListBizDemandsResp{Items: []BizDemandItem{}}, nil
	}

	var all []po.WorkItemDetail
	page := 1
	for {
		req := po.DemandsReq{Status: "all", Page: page, PageSize: bizDemandPageSize}
		req.Normalize()
		resp, err := s.poSvc.Demands(ctx, actor, req)
		if err != nil {
			return ListBizDemandsResp{}, err
		}
		if resp == nil || len(resp.Items) == 0 {
			break
		}
		all = append(all, resp.Items...)
		if int64(page*bizDemandPageSize) >= resp.Total || len(resp.Items) < bizDemandPageSize {
			break
		}
		page++
	}

	return ListBizDemandsResp{Items: toBizDemandItems(filterBizDemandItems(all))}, nil
}

func filterBizDemandItems(items []po.WorkItemDetail) []po.WorkItemDetail {
	out := make([]po.WorkItemDetail, 0, len(items))
	for _, it := range items {
		if strings.TrimSpace(it.Kind) != "demand" {
			continue
		}
		out = append(out, it)
	}
	return out
}

func toBizDemandItems(items []po.WorkItemDetail) []BizDemandItem {
	out := make([]BizDemandItem, 0, len(items))
	for _, it := range items {
		out = append(out, BizDemandItem{
			ID:           it.ID,
			Pri:          it.Pri,
			Title:        it.Title,
			Owner:        it.Owner,
			ValueStream:  it.ValueStream,
			ZentaoUrl:    it.ZentaoUrl,
			ZentaoStatus: it.ZentaoStatus,
		})
	}
	return out
}
