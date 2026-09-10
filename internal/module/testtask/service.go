// =============================================================================
// 文件: internal/module/testtask/service.go
// 模块: 提测办理
// 类型: action
// 职责: 提测上下文与产品执行列表业务装配。
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

	products, err := s.repo.FindDemandInvolvedProducts(ctx, demandID)
	if err != nil {
		return nil, err
	}
	systems := BuildSystemItems(products, row.MainSystemID, row.MainSystemName)

	account, name := "", ""
	if actor != nil {
		account = actor.Account
		name = actor.DisplayName
	}
	return BuildContextResp(*row, displayMap, account, name, systems), nil
}

// ListProductExecutions 当前产品下执行列表（对齐禅道版本创建 stagefilter|leaf|order_asc[+noclosed]）。
func (s *Service) ListProductExecutions(ctx context.Context, actor *model.User, productID uint) ([]ExecutionOption, error) {
	_ = actor // 预留：后续可按可见执行权限过滤
	if productID == 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "产品 ID 无效")
	}
	projectIDs, err := s.repo.FindProductProjectIDs(ctx, productID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.FindExecutionsByProjectIDs(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	crExec, err := s.repo.FindCRExecution(ctx)
	if err != nil {
		return nil, err
	}
	noClosed := crExec == 0
	return BuildExecutionOptions(rows, noClosed), nil
}
