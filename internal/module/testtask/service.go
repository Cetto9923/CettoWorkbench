// =============================================================================
// 文件: internal/module/testtask/service.go
// 模块: 提测办理
// 类型: action
// 职责: 提测上下文业务装配（人员展示名走 user.AccountDisplayMap）。
// 依赖: internal/module/user
//       internal/pkg/errorx
// =============================================================================

package testtask

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"workbench/internal/model"
	"workbench/internal/module/user"
	"workbench/internal/pkg/errorx"
)

// Service 提测办理业务逻辑。
type Service struct {
	repo    *Repo
	userSvc *user.Service
	logger  *zap.Logger
}

// NewService 创建 Service。
func NewService(repo *Repo, userSvc *user.Service, logger *zap.Logger) *Service {
	return &Service{repo: repo, userSvc: userSvc, logger: logger}
}

// GetContext 获取业需提测上下文。提测办理人为当前登录用户。
func (s *Service) GetContext(ctx context.Context, actor *model.User, demandID uint) (*ContextResp, error) {
	if demandID == 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "需求 ID 无效")
	}
	row, err := s.repo.FindDemandContext(ctx, demandID)
	if err != nil {
		if errors.Is(err, errDemandNotFound) {
			return nil, errorx.New(errorx.ErrCodeNotFound, "需求不存在")
		}
		return nil, err
	}

	displayMap := map[string]string{}
	if s.userSvc != nil {
		m, mapErr := s.userSvc.AccountDisplayMap(ctx, actor)
		if mapErr != nil {
			return nil, mapErr
		}
		if m != nil {
			displayMap = m
		}
	}

	account, name := "", ""
	if actor != nil {
		account = actor.Account
		name = actor.DisplayName
	}
	return BuildContextResp(*row, displayMap, account, name), nil
}
