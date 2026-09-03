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
		CustomFrom: req.CustomFrom, CustomTo: req.CustomTo, ObjectType: req.ObjectType, Result: req.Result,
		Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	// 时间段概览计数（与待办 focus 卡同构）：全部时间 / 今天 / 近7天 / 本周 / 近30天 / 本月 / 本季度
	summary := DoneSummary{}
	if req.ObjectType == "" || req.ObjectType == "all" {
		if summary.All, err = s.repo.CountDoneActions(ctx, actor.Account, TimeRangeAll); err != nil {
			return nil, err
		}
		if summary.Today, err = s.repo.CountDoneActions(ctx, actor.Account, TimeRangeToday); err != nil {
			return nil, err
		}
		if summary.Last7d, err = s.repo.CountDoneActions(ctx, actor.Account, TimeRange7d); err != nil {
			return nil, err
		}
		if summary.Week, err = s.repo.CountDoneActions(ctx, actor.Account, TimeRangeWeek); err != nil {
			return nil, err
		}
		if summary.Last30d, err = s.repo.CountDoneActions(ctx, actor.Account, TimeRange30d); err != nil {
			return nil, err
		}
		if summary.Month, err = s.repo.CountDoneActions(ctx, actor.Account, TimeRangeMonth); err != nil {
			return nil, err
		}
		if summary.Quarter, err = s.repo.CountDoneActions(ctx, actor.Account, TimeRangeQuarter); err != nil {
			return nil, err
		}
	} else {
		// 选择具体对象后概览卡只对该对象计数
		for tr, dst := range map[TimeRange]*int64{
			TimeRangeAll: &summary.All, TimeRangeToday: &summary.Today,
			TimeRange7d: &summary.Last7d, TimeRangeWeek: &summary.Week,
			TimeRange30d: &summary.Last30d, TimeRangeMonth: &summary.Month,
			TimeRangeQuarter: &summary.Quarter,
		} {
			if *dst, err = s.countDoneForObject(ctx, actor.Account, req.ObjectType, tr); err != nil {
				return nil, err
			}
		}
	}

	return &DoneListResp{
		Items:    items,
		Total:    total,
		Summary:  summary,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// countDoneForObject 按对象类型 + 时间段统计已办动作数（对象型概览）。
func (s *Service) countDoneForObject(ctx context.Context, account, objectType string, tr TimeRange) (int64, error) {
	return s.repo.CountDoneActionsForObject(ctx, account, objectType, tr)
}
