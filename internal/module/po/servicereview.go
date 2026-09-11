// =============================================================================
// 文件: internal/module/po/servicereview.go
// 模块: PO 工作台
// 类型: action
// 职责: 业需评审业务规则（对照禅道 demand->review，不含 OA 同步与转工单）。
// 依赖: internal/model
//       internal/pkg/errorx
// =============================================================================

package po

import (
	"context"
	"strings"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/zentao"
)

// ReviewDemand 提交业需评审。
//
// 业务逻辑与系统联动说明：
//  1. actor 为当前登录用户，执行对象级授权前置校验（待评审 + 业务评审人未出结果）；
//  2. 实际业务流转（状态扭转、转工单流转、OA 待办消息推送等）统一走禅道原生 API，
//     确保逻辑与禅道原版行为完全一致。
func (s *Service) ReviewDemand(ctx context.Context, actor *model.User, req ReviewDemandReq) (ReviewDemandResp, error) {
	empty := ReviewDemandResp{}
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return empty, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	account := strings.TrimSpace(actor.Account)

	demand, err := s.repo.FindDemandForReview(ctx, req.ID)
	if err != nil {
		return empty, err
	}
	if demand == nil || demand.Deleted != "0" {
		return empty, errorx.New(errorx.ErrCodeNotFound, "需求不存在")
	}
	// 对齐禅道：status=wait 且 zt_demandreview 里有当前账号、result 仍为空。
	if strings.TrimSpace(demand.Status) != "wait" {
		return empty, errorx.New(errorx.ErrCodeConflict, "该需求不是待评审状态")
	}
	reviewResult, found, revErr := s.repo.FindDemandReviewerResult(ctx, req.ID, account)
	if revErr != nil {
		return empty, revErr
	}
	if !found {
		return empty, errorx.New(errorx.ErrCodeForbidden, "只有业务评审人可以评审")
	}
	if strings.TrimSpace(reviewResult) != "" {
		return empty, errorx.New(errorx.ErrCodeConflict, "您已评审过该需求")
	}

	// 统一调用禅道原生接口，让禅道处理转工单、状态流转及 OA 消息同步
	ztClient := zentao.DefaultClient()
	ztErr := ztClient.ReviewDemand(ctx, zentao.DemandReviewParams{
		DemandID:    uint(req.ID),
		Account:     account,
		Result:      req.Result,
		IsNeedFocus: req.IsNeedFocus,
		Comment:     req.Comment,
		Mailto:      req.Mailto,
	})
	if ztErr != nil {
		return empty, errorx.New(errorx.ErrCodeInternal, "禅道评审执行失败: "+ztErr.Error())
	}

	return ReviewDemandResp{ID: req.ID}, nil
}

// WithdrawDemandReview 撤回业需评审申请（仅创建人或超级管理员可操作）。
func (s *Service) WithdrawDemandReview(ctx context.Context, actor *model.User, req WithdrawDemandReviewReq) error {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	account := strings.TrimSpace(actor.Account)

	demand, err := s.repo.FindDemandForReview(ctx, req.ID)
	if err != nil {
		return err
	}
	if demand == nil || demand.Deleted != "0" {
		return errorx.New(errorx.ErrCodeNotFound, "需求不存在")
	}
	if strings.TrimSpace(demand.Status) != "wait" {
		return errorx.New(errorx.ErrCodeConflict, "该需求不是待评审状态")
	}
	if !actor.IsSuperAdmin && strings.TrimSpace(demand.CreatedBy) != account {
		return errorx.New(errorx.ErrCodeForbidden, "只有创建人可以撤回评审")
	}

	ztClient := zentao.DefaultClient()
	ztErr := ztClient.WithdrawDemandReview(ctx, zentao.WithdrawDemandReviewParams{
		DemandID: uint(req.ID),
		Account:  account,
		Comment:  req.Comment,
	})
	if ztErr != nil {
		return errorx.New(errorx.ErrCodeInternal, "禅道撤回评审执行失败: "+ztErr.Error())
	}
	return nil
}

// SubmitDemandReview 提交业需评审（草稿/已驳回状态下，创建人或指派人可操作）。
func (s *Service) SubmitDemandReview(ctx context.Context, actor *model.User, req SubmitDemandReviewReq) error {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	account := strings.TrimSpace(actor.Account)

	demand, err := s.repo.FindDemandForReview(ctx, req.ID)
	if err != nil {
		return err
	}
	if demand == nil || demand.Deleted != "0" {
		return errorx.New(errorx.ErrCodeNotFound, "需求不存在")
	}
	st := strings.TrimSpace(demand.Status)
	if st != "draft" && st != "refuse" {
		return errorx.New(errorx.ErrCodeConflict, "当前状态不允许提交评审")
	}
	if !actor.IsSuperAdmin && strings.TrimSpace(demand.CreatedBy) != account && strings.TrimSpace(demand.AssignedTo) != account {
		return errorx.New(errorx.ErrCodeForbidden, "只有创建人或指派人可以提交评审")
	}

	reviewers := req.Reviewer
	if len(reviewers) == 0 {
		reviewers = splitReviewerAccounts(demand.Reviewer)
	}

	ztClient := zentao.DefaultClient()
	ztErr := ztClient.SubmitDemandReview(ctx, zentao.SubmitDemandReviewParams{
		DemandID: uint(req.ID),
		Account:  account,
		Reviewer: reviewers,
		Comment:  req.Comment,
	})
	if ztErr != nil {
		return errorx.New(errorx.ErrCodeInternal, "禅道提交评审执行失败: "+ztErr.Error())
	}
	return nil
}

// splitReviewerAccounts 把 zt_demand.reviewer 的逗号分隔账号拆成去空串切片。
func splitReviewerAccounts(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		p := strings.TrimSpace(part)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
