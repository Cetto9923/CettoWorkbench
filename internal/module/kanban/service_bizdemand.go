// =============================================================================
// 文件: internal/module/kanban/service_bizdemand.go
// 模块: 工作看板
// 类型: readonly
// 职责: 复用首页价值流 Demands，按选中负责人组装需求树（业需 + 独立研需）。
// 依赖: internal/model
//       internal/module/po
//       internal/pkg/errorx
// =============================================================================

package kanban

import (
	"context"
	"strings"

	"workbench/internal/model"
	"workbench/internal/module/po"
)

const bizDemandPageSize = 100

// ListValueStreamBizDemands 按选中负责人拉取价值流「全部」业需与研需。
// account 为空时回落到 actor；account=all 时按当前敏捷小组全员聚合去重。
func (s *Service) ListValueStreamBizDemands(ctx context.Context, actor *model.User, req ListDemandsReq) (ListBizDemandsResp, error) {
	if s.poSvc == nil {
		return ListBizDemandsResp{Items: []BizDemandItem{}}, nil
	}
	actorAccount := ""
	if actor != nil {
		actorAccount = actor.Account
	}

	groups, err := s.ListMyTeamgroups(ctx, actor)
	if err != nil {
		return ListBizDemandsResp{}, err
	}
	targets, err := resolveKanbanAccounts(actorAccount, req.Account, req.TeamgroupID, groups)
	if err != nil {
		return ListBizDemandsResp{}, err
	}
	if len(targets) == 0 {
		return ListBizDemandsResp{Items: []BizDemandItem{}}, nil
	}

	seen := make(map[string]struct{})
	var all []po.WorkItemDetail
	for _, target := range targets {
		items, demErr := s.fetchDemandsForAccount(ctx, viewAsUser(actor, target))
		if demErr != nil {
			return ListBizDemandsResp{}, demErr
		}
		for _, it := range items {
			key := it.Kind + ":" + it.ID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			all = append(all, it)
		}
	}

	return ListBizDemandsResp{Items: toBizDemandItems(all)}, nil
}

func (s *Service) fetchDemandsForAccount(ctx context.Context, viewAs *model.User) ([]po.WorkItemDetail, error) {
	var all []po.WorkItemDetail
	page := 1
	for {
		dreq := po.DemandsReq{Status: "all", Page: page, PageSize: bizDemandPageSize}
		dreq.Normalize()
		resp, demErr := s.poSvc.Demands(ctx, viewAs, dreq)
		if demErr != nil {
			return nil, demErr
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
	return all, nil
}

func viewAsUser(actor *model.User, account string) *model.User {
	if actor == nil {
		return &model.User{Account: account}
	}
	u := *actor
	u.Account = account
	if account != strings.TrimSpace(actor.Account) {
		u.DisplayName = ""
	}
	return &u
}

func toBizDemandItems(items []po.WorkItemDetail) []BizDemandItem {
	out := make([]BizDemandItem, 0, len(items))
	for _, it := range items {
		out = append(out, BizDemandItem{
			Kind:         it.Kind,
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
