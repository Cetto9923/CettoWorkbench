// =============================================================================
// 文件: internal/module/po/serviceacceptance.go
// 模块: PO 工作台
// 类型: action
// 职责: 业需验收：以当前用户身份转发禅道 POST /demand/:id/acceptance。
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

// AcceptDemand 提交业需验收（代理禅道 API）。
func (s *Service) AcceptDemand(ctx context.Context, actor *model.User, req AcceptanceReq) (AcceptanceResp, error) {
	empty := AcceptanceResp{}
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

	if callErr := acceptDemandViaZentao(ctx, client, acceptDemandViaZentaoReq{
		DemandID:   req.ID,
		Acceptance: req.Acceptance,
		AssignedTo: req.AssignedTo,
		Comment:    req.Comment,
	}); callErr != nil {
		if s.logger != nil {
			s.logger.Error("zentao demand acceptance",
				zap.Error(callErr),
				zap.Int64("id", req.ID),
				zap.String("account", account),
			)
		}
		return empty, errorx.Wrap(errorx.ErrCodeInvalidParam, fmt.Sprintf("验收失败：%s", callErr.Error()), callErr)
	}

	return AcceptanceResp{ID: req.ID}, nil
}
