// =============================================================================
// 文件: internal/module/po/servicereview.go
// 模块: PO 工作台
// 类型: action
// 职责: 业需发起评审 / 评审 / 撤回：本地校验资格后，以当前用户身份转发禅道 API。
// 依赖: internal/model
//       internal/pkg/errorx
//       internal/pkg/zentao
// =============================================================================

package po

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"workbench/internal/model"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/zentao"
)

// SubmitDemandReview 发起业需评审（仅创建人 + draft/refuse；代理禅道 submit）。
func (s *Service) SubmitDemandReview(ctx context.Context, actor *model.User, req SubmitDemandReviewReq) (SubmitDemandReviewResp, error) {
	empty := SubmitDemandReviewResp{}
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
	status := strings.TrimSpace(demand.Status)
	if status != "draft" && status != "refuse" {
		return empty, errorx.New(errorx.ErrCodeConflict, "仅草稿或已驳回的需求可发起评审")
	}
	if strings.TrimSpace(demand.CreatedBy) != account {
		return empty, errorx.New(errorx.ErrCodeForbidden, "只有创建人可以发起评审")
	}

	client := s.ztAPI
	if client == nil {
		client = zentao.API()
	}
	if client == nil {
		return empty, errorx.New(errorx.ErrCodeInternal, "禅道 API 未配置")
	}

	if callErr := submitDemandReviewViaZentao(ctx, client, submitDemandReviewViaZentaoReq{
		DemandID: req.ID,
		Reviewer: req.Reviewer,
		Comment:  req.Comment,
	}); callErr != nil {
		if s.logger != nil {
			s.logger.Error("zentao demand submit",
				zap.Error(callErr),
				zap.Int64("id", req.ID),
				zap.String("account", account),
			)
		}
		return empty, errorx.Wrap(errorx.ErrCodeInvalidParam, fmt.Sprintf("提交评审失败：%s", callErr.Error()), callErr)
	}

	return SubmitDemandReviewResp{ID: req.ID}, nil
}

// ReviewDemand 提交业需评审（代理禅道 API，不再本地写 zt_demandreview）。
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

	client := s.ztAPI
	if client == nil {
		client = zentao.API()
	}
	if client == nil {
		return empty, errorx.New(errorx.ErrCodeInternal, "禅道 API 未配置")
	}

	if callErr := reviewDemandViaZentao(ctx, client, reviewDemandViaZentaoReq{
		DemandID: req.ID,
		Result:   req.Result,
		Comment:  req.Comment,
	}); callErr != nil {
		if s.logger != nil {
			s.logger.Error("zentao demand review",
				zap.Error(callErr),
				zap.Int64("id", req.ID),
				zap.String("account", account),
				zap.String("result", req.Result),
			)
		}
		return empty, errorx.Wrap(errorx.ErrCodeInvalidParam, fmt.Sprintf("评审失败：%s", callErr.Error()), callErr)
	}

	return ReviewDemandResp{ID: req.ID}, nil
}

// WithdrawDemandReview 撤回业需评审申请（仅创建人或超级管理员；代理禅道 withdrawReview）。
func (s *Service) WithdrawDemandReview(ctx context.Context, actor *model.User, req WithdrawDemandReviewReq) (WithdrawDemandReviewResp, error) {
	empty := WithdrawDemandReviewResp{}
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
	if strings.TrimSpace(demand.Status) != "wait" {
		return empty, errorx.New(errorx.ErrCodeConflict, "该需求不是待评审状态")
	}
	if !actor.IsSuperAdmin && strings.TrimSpace(demand.CreatedBy) != account {
		return empty, errorx.New(errorx.ErrCodeForbidden, "只有创建人可以撤回评审")
	}

	client := s.ztAPI
	if client == nil {
		client = zentao.API()
	}
	if client == nil {
		return empty, errorx.New(errorx.ErrCodeInternal, "禅道 API 未配置")
	}

	if callErr := withdrawDemandReviewViaZentao(ctx, client, withdrawDemandReviewViaZentaoReq{
		DemandID: req.ID,
		Comment:  req.Comment,
	}); callErr != nil {
		if s.logger != nil {
			s.logger.Error("zentao demand withdrawReview",
				zap.Error(callErr),
				zap.Int64("id", req.ID),
				zap.String("account", account),
			)
		}
		return empty, errorx.Wrap(errorx.ErrCodeInvalidParam, fmt.Sprintf("撤回评审失败：%s", callErr.Error()), callErr)
	}

	return WithdrawDemandReviewResp{ID: req.ID}, nil
}
