// =============================================================================
// 文件: internal/module/po/servicedone.go
// 模块: PO 工作台
// 类型: action
// 职责: 我的已办服务（V10.1 02 节: 严格 7 维 AND 公式, 一对象多动作 = 独立已办事件）。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"strings"

	"workbench/internal/model"
)

// DoneList 我的已办列表服务。
func (s *Service) DoneList(ctx context.Context, actor *model.User, req DoneListReq) (*DoneListResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return &DoneListResp{Items: []DoneAction{}, Page: req.Page, PageSize: req.PageSize}, nil
	}
	items, total, err := s.repo.FindDoneActions(ctx, RepoFindDoneActionsReq{
		Account: actor.Account, Tab: req.Tab, TimeRange: req.TimeRange,
		CustomFrom: req.CustomFrom, CustomTo: req.CustomTo, ObjectType: req.ObjectType,
		Result: req.Result, Action: req.Action, Keyword: req.Keyword,
		Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	// 时间段概览计数（与待办 focus 卡同构）：单次 SQL 聚合 7 个区间。
	objectType := req.ObjectType
	if objectType == "all" {
		objectType = ""
	}
	summary, err := s.repo.CountDoneActions(ctx, RepoCountDoneActionsReq{
		Account:    actor.Account,
		ObjectType: objectType,
	})
	if err != nil {
		return nil, err
	}

	return &DoneListResp{
		Items:    items,
		Total:    total,
		Summary:  summary,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
