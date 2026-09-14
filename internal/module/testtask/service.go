// =============================================================================
// 文件: internal/module/testtask/service.go
// 模块: 提测办理
// 类型: action
// 职责: 提测上下文、产品执行/已有版本列表与创建版本业务装配。
// 依赖: internal/module/user
//       internal/pkg/errorx
//       internal/pkg/zentao
// =============================================================================

package testtask

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"workbench/internal/model"
	"workbench/internal/module/user"
	"workbench/internal/pkg/errorx"
	"workbench/internal/pkg/zentao"
)

// Service 提测办理业务逻辑。
type Service struct {
	repo    *Repo
	userSvc *user.Service
	ztAPI   *zentao.Client
	logger  *zap.Logger
}

// NewService 创建 Service。ztAPI 可为空，写入禅道时回退 zentao.API()。
func NewService(repo *Repo, userSvc *user.Service, ztAPI *zentao.Client, logger *zap.Logger) *Service {
	return &Service{repo: repo, userSvc: userSvc, ztAPI: ztAPI, logger: logger}
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

	users := []UserOption{}
	if s.userSvc != nil {
		pickerUsers, listErr := s.userSvc.ListInsideUsers(ctx, actor)
		if listErr != nil {
			return nil, listErr
		}
		users = make([]UserOption, 0, len(pickerUsers))
		for _, item := range pickerUsers {
			users = append(users, UserOption{Account: item.Account, Realname: item.Realname})
		}
	}

	account, name := "", ""
	if actor != nil {
		account = actor.Account
		name = actor.DisplayName
	}
	return BuildContextResp(*row, displayMap, account, name, systems, users), nil
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

// ListProductBuilds 当前产品下已有版本列表（代理禅道 GET /products/:id/builds）。
func (s *Service) ListProductBuilds(ctx context.Context, actor *model.User, productID uint) ([]BuildOption, error) {
	_ = actor // 预留：后续可按可见版本权限过滤
	if productID == 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "产品 ID 无效")
	}
	client := s.ztAPI
	if client == nil {
		client = zentao.API()
	}
	if client == nil {
		return nil, errorx.New(errorx.ErrCodeInternal, "禅道 API 未配置")
	}
	items, err := listProductBuilds(ctx, client, productID)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("zentao list product builds", zap.Error(err), zap.Uint("productId", productID))
		}
		return nil, errorx.Wrap(errorx.ErrCodeInvalidParam, fmt.Sprintf("获取已有版本失败：%s", err.Error()), err)
	}
	return BuildBuildOptions(items), nil
}

// CreateBuilds 将「创建新版本」项同步到禅道 POST /projects/:id/builds；builder 为当前用户 account。
func (s *Service) CreateBuilds(ctx context.Context, actor *model.User, demandID uint, req CreateBuildsReq) (*CreateBuildsResp, error) {
	_ = demandID // 路由上下文，便于日志与后续扩展校验
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	client := s.ztAPI
	if client == nil {
		client = zentao.API()
	}
	if client == nil {
		return nil, errorx.New(errorx.ErrCodeInternal, "禅道 API 未配置")
	}

	out := &CreateBuildsResp{Builds: make([]CreateBuildResult, 0, len(req.Builds))}
	for _, item := range req.Builds {
		build, err := createProjectBuild(ctx, client, createProjectBuildReq{
			ProjectID:   item.ProjectID,
			ExecutionID: item.ExecutionID,
			ProductID:   item.ProductID,
			Name:        strings.TrimSpace(item.Name),
			Builder:     strings.TrimSpace(actor.Account),
			Date:        strings.TrimSpace(item.Date),
			Desc:        item.Desc,
		})
		if err != nil {
			if s.logger != nil {
				s.logger.Error("zentao create build",
					zap.Error(err),
					zap.Uint("productId", item.ProductID),
					zap.Uint("projectId", item.ProjectID),
					zap.String("name", item.Name),
				)
			}
			return nil, errorx.Wrap(errorx.ErrCodeInvalidParam, fmt.Sprintf("保存版本失败：%s", err.Error()), err)
		}
		out.Builds = append(out.Builds, CreateBuildResult{
			ProductID: item.ProductID,
			BuildID:   build.ID,
			Name:      build.Name,
		})
	}
	return out, nil
}
