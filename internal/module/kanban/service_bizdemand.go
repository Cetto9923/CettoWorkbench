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
	"workbench/internal/pkg/errorx"
)

const bizDemandPageSize = 100

// ListValueStreamBizDemands 按选中负责人拉取价值流「全部」业需与研需。
// account 为空时回落到 actor；非小组成员返回 forbidden。
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
	target, err := resolveDemandAccount(actorAccount, req.Account, collectMemberAccounts(groups))
	if err != nil {
		return ListBizDemandsResp{}, err
	}

	viewAs := viewAsUser(actor, target)
	var all []po.WorkItemDetail
	page := 1
	for {
		dreq := po.DemandsReq{Status: "all", Page: page, PageSize: bizDemandPageSize}
		dreq.Normalize()
		resp, demErr := s.poSvc.Demands(ctx, viewAs, dreq)
		if demErr != nil {
			return ListBizDemandsResp{}, demErr
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

	return ListBizDemandsResp{Items: toBizDemandItems(all)}, nil
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

func collectMemberAccounts(groups []TeamgroupItem) map[string]struct{} {
	out := make(map[string]struct{})
	for _, g := range groups {
		for _, m := range g.Members {
			acc := strings.TrimSpace(m.Account)
			if acc == "" {
				continue
			}
			out[acc] = struct{}{}
		}
	}
	return out
}

func resolveDemandAccount(actorAccount, reqAccount string, allowed map[string]struct{}) (string, error) {
	actorAccount = strings.TrimSpace(actorAccount)
	acc := strings.TrimSpace(reqAccount)
	if acc == "" {
		acc = actorAccount
	}
	if acc == "" {
		return "", errorx.New(errorx.ErrCodeInvalidParam, "账号不能为空")
	}
	if acc == actorAccount {
		return acc, nil
	}
	if _, ok := allowed[acc]; !ok {
		return "", errorx.New(errorx.ErrCodeForbidden, "无权查看该成员需求")
	}
	return acc, nil
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
