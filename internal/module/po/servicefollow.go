// =============================================================================
// 文件: internal/module/po/servicefollow.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的关注服务。V10.1 04 节: 只有 2 个对象视图（业务需求默认 / 项目报告），无"全部"。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"strings"

	"workbench/internal/model"
	"workbench/internal/pkg/zentao"
)

// FollowList 我的关注列表服务，当前仅提供业务需求与周报列表两条独立接口。
func (s *Service) FollowList(ctx context.Context, actor *model.User, req FollowListReq) (*FollowListResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return &FollowListResp{Items: []FollowItem{}, Page: req.Page, PageSize: req.PageSize}, nil
	}
	switch req.Tab {
	case FollowTabDemand:
		items, total, stats, err := s.repo.FindFollowedDemands(ctx, RepoFindFollowedDemandsReq{
			Account: actor.Account, Scope: req.Scope, Lifecycle: req.Lifecycle,
			Keyword: req.Keyword, Page: req.Page, PageSize: req.PageSize,
		})
		if err != nil {
			return nil, err
		}
		if err := s.attachFollowPrimaryActions(ctx, actor, items); err != nil {
			return nil, err
		}
		return &FollowListResp{Items: items, Total: total, Page: req.Page, PageSize: req.PageSize, Stats: stats}, nil
	}
	return &FollowListResp{Items: []FollowItem{}, Page: req.Page, PageSize: req.PageSize}, nil
}

// attachFollowPrimaryActions 为关注列表业需批量挂主操作。
func (s *Service) attachFollowPrimaryActions(ctx context.Context, actor *model.User, items []FollowItem) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(items))
	for _, it := range items {
		if it.ID > 0 {
			ids = append(ids, uint(it.ID))
		}
	}
	actions, err := s.DeriveDemandPrimaryActions(ctx, actor, ids)
	if err != nil {
		return err
	}
	for i := range items {
		if pa, ok := actions[uint(items[i].ID)]; ok {
			paCopy := pa
			items[i].PrimaryAction = &paCopy
		}
	}
	return nil
}

// FollowSetDemand 切换对业务需求的关注。
// 写入走禅道 ajaxFollowObject / ajaxUnfollowObject；取消关注后补写 followed=0，压制历史 mailto 抄送关注。
func (s *Service) FollowSetDemand(ctx context.Context, actor *model.User, req FollowSetReq) error {
	if actor == nil || strings.TrimSpace(actor.Account) == "" || req.Followed == nil || req.ID <= 0 {
		return nil
	}
	zt := zentao.SiteClient()
	var err error
	if *req.Followed {
		err = zt.FollowDemandObject(ctx, actor.Account, req.ID)
	} else {
		err = zt.UnfollowDemandObject(ctx, actor.Account, req.ID)
		// 禅道 unfollowObject 在无 starinfo 行时直接返回 0（mailto 历史关注），
		// 需显式写入 followed=0，列表 WHERE 才能排除抄送关注。
		if err == nil {
			if ensureErr := s.repo.EnsureDemandUnfollowed(ctx, RepoSaveDemandFollowReq{
				Account: actor.Account, DemandID: req.ID, Followed: false,
			}); ensureErr != nil {
				return ensureErr
			}
		}
	}
	return err
}

// FollowRemoveProjectReport 解除当前用户对项目周报的关注。
func (s *Service) FollowRemoveProjectReport(ctx context.Context, actor *model.User, projectID int64) error {
	if actor == nil || strings.TrimSpace(actor.Account) == "" || projectID <= 0 {
		return nil
	}
	return s.repo.RemoveProjectReportFollow(ctx, RepoRemoveProjectReportFollowReq{Account: actor.Account, ProjectID: projectID})
}
