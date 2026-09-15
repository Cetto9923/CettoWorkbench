// =============================================================================
// 文件: internal/module/po/servicedeliver.go
// 模块: PO 工作台
// 类型: action
// 职责: 发起交付：以当前用户身份转发禅道 POST /demand/:id/deliver。
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

// DeliverDemand 提交发起交付（代理禅道 API）。
func (s *Service) DeliverDemand(ctx context.Context, actor *model.User, req DeliverDemandReq) (DeliverDemandResp, error) {
	empty := DeliverDemandResp{}
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return empty, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	account := strings.TrimSpace(actor.Account)

	client := s.ztAPI
	if client == nil {
		client = zentao.API()
	}
	if client == nil {
		return empty, errorx.New(errorx.ErrCodeInternal, "禅道 API 未配置")
	}

	if callErr := deliverDemandViaZentao(ctx, client, deliverDemandViaZentaoReq{
		DemandID:         req.ID,
		DeliverDate:      req.DeliverDate,
		IsGrayVerifyPlan: req.IsGrayVerifyPlan,
		VerifyDate:       req.VerifyDate,
		VerifyPlan:       req.VerifyPlan,
		VeriFier:         req.VeriFier,
		IsCarReview:      req.IsCarReview,
	}); callErr != nil {
		if s.logger != nil {
			s.logger.Error("zentao demand deliver",
				zap.Error(callErr),
				zap.Int64("id", req.ID),
				zap.String("account", account),
			)
		}
		return empty, errorx.Wrap(errorx.ErrCodeInvalidParam, fmt.Sprintf("发起交付失败：%s", callErr.Error()), callErr)
	}

	return DeliverDemandResp{ID: req.ID}, nil
}
