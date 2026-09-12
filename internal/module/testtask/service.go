// =============================================================================
// 文件: internal/module/testtask/service.go
// 模块: 提测办理
// 类型: action
// 职责: 提测上下文、产品执行列表与创建版本业务装配。
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

// NewService 创建 Service。ztAPI 可为空，写入禅道时回退 zentao.DefaultClient()。
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

	stories, storiesErr := s.repo.FindDemandConvertedStories(ctx, demandID)
	if storiesErr != nil {
		return nil, storiesErr
	}
	systems := BuildSystemItemsWithStories(stories, row.MainSystemID, row.MainSystemName)

	users, usersErr := s.repo.ListInsideUsers(ctx)
	if usersErr != nil {
		return nil, usersErr
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
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
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

// ListProductBuilds 返回产品已有版本。
func (s *Service) ListProductBuilds(ctx context.Context, actor *model.User, productID uint) ([]BuildOption, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	if productID == 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "产品 ID 无效")
	}
	client := s.ztAPI
	if client == nil {
		client = zentao.DefaultClient()
	}
	if client == nil {
		return nil, errorx.New(errorx.ErrCodeInternal, "禅道 API 未配置")
	}
	items, err := listProductBuilds(ctx, client, productID)
	if err != nil {
		return nil, err
	}
	out := make([]BuildOption, 0, len(items))
	for _, item := range items {
		if item.ID == 0 {
			continue
		}
		label := strings.TrimSpace(item.Name)
		if label == "" {
			label = fmt.Sprint(item.ID)
		}
		out = append(out, BuildOption{Value: fmt.Sprint(item.ID), Label: label})
	}
	return out, nil
}

// CreateBuilds 将「创建新版本」项同步到禅道 POST /projects/:id/builds；builder 为当前用户 account。
func (s *Service) CreateBuilds(ctx context.Context, actor *model.User, demandID uint, req CreateBuildsReq) (*CreateBuildsResp, error) {
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	if demandID == 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "需求 ID 无效")
	}
	if s.repo == nil {
		return nil, errorx.New(errorx.ErrCodeInternal, "提测上下文不可用")
	}
	if _, err := s.repo.FindDemandContext(ctx, demandID); err != nil {
		if errors.Is(err, errDemandNotFound) {
			return nil, errorx.New(errorx.ErrCodeNotFound, "需求不存在")
		}
		return nil, err
	}
	client := s.ztAPI
	if client == nil {
		client = zentao.DefaultClient()
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
