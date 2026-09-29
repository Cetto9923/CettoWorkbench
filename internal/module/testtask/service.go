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
	if err := s.attachNextBuildSeqs(ctx, systems); err != nil {
		return nil, err
	}
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

// ListProductExecutions 当前产品下执行列表（对齐禅道 product::getExecutionPairsByProduct(productID,”,0,'stagefilter')）。
// 非超管按禅道 view.sprints 过滤（user.VisibleSprintIDs / ZentaoView）。
func (s *Service) ListProductExecutions(ctx context.Context, actor *model.User, productID uint) ([]ExecutionOption, error) {
	if productID == 0 {
		return nil, errorx.New(errorx.ErrCodeInvalidParam, "产品 ID 无效")
	}
	rows, err := s.repo.FindExecutionsByProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	rows, err = s.filterExecutionsByActorView(ctx, actor, rows)
	if err != nil {
		return nil, err
	}
	projectIDs := uniqueProjectIDs(rows)
	stages, err := s.repo.FindStagesByProjectIDs(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	return BuildExecutionOptions(rows, stages), nil
}

// filterExecutionsByActorView 对齐禅道：!$admin 时 andWhere id in view.sprints。
func (s *Service) filterExecutionsByActorView(ctx context.Context, actor *model.User, rows []executionRow) ([]executionRow, error) {
	if len(rows) == 0 {
		return rows, nil
	}
	if s.userSvc == nil {
		return nil, errorx.New(errorx.ErrCodeInternal, "用户服务未配置")
	}
	view, err := s.userSvc.ZentaoView(ctx, actor)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("zentao user view", zap.Error(err))
		}
		return nil, errorx.Wrap(errorx.ErrCodeInternal, "获取可见执行失败", err)
	}
	if view.Admin {
		return rows, nil
	}
	return filterExecutionsByVisibleSprints(rows, view.Sprints), nil
}

// filterExecutionsByVisibleSprints 保留 id ∈ sprintIDs 的执行行。
func filterExecutionsByVisibleSprints(rows []executionRow, sprintIDs []uint) []executionRow {
	if len(rows) == 0 {
		return rows
	}
	if len(sprintIDs) == 0 {
		return []executionRow{}
	}
	allow := make(map[uint]struct{}, len(sprintIDs))
	for _, id := range sprintIDs {
		if id == 0 {
			continue
		}
		allow[id] = struct{}{}
	}
	out := make([]executionRow, 0, len(rows))
	for _, row := range rows {
		if _, ok := allow[row.ID]; ok {
			out = append(out, row)
		}
	}
	return out
}

func uniqueProjectIDs(rows []executionRow) []uint {
	if len(rows) == 0 {
		return nil
	}
	seen := make(map[uint]struct{}, len(rows))
	out := make([]uint, 0, len(rows))
	for _, row := range rows {
		if row.ProjectID == 0 {
			continue
		}
		if _, ok := seen[row.ProjectID]; ok {
			continue
		}
		seen[row.ProjectID] = struct{}{}
		out = append(out, row.ProjectID)
	}
	return out
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
	if actor == nil || strings.TrimSpace(actor.Account) == "" {
		return nil, errorx.New(errorx.ErrCodeForbidden, "请先登录")
	}
	// 对象级写权限闸门：无权时直接 403，不请求禅道。
	if err := s.RequireDemandWriteAccess(ctx, actor, demandID); err != nil {
		return nil, err
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

		name := strings.TrimSpace(item.Name)

		build, err := createProjectBuild(ctx, client, createProjectBuildReq{
			ProjectID:   item.ProjectID,
			ExecutionID: item.ExecutionID,
			ProductID:   item.ProductID,
			Name:        name,
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
					zap.String("name", name),
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

// attachNextBuildSeqs 就地回填每个系统的下一个版本编号（现有版本总数含已删除 + 1）。
func (s *Service) attachNextBuildSeqs(ctx context.Context, systems []SystemItem) error {
	for i := range systems {
		if systems[i].ID == 0 {
			continue
		}
		total, err := s.repo.CountProductBuilds(ctx, systems[i].ID)
		if err != nil {
			if s.logger != nil {
				s.logger.Error("count product builds", zap.Error(err), zap.Uint("productId", systems[i].ID))
			}
			return errorx.Wrap(errorx.ErrCodeInternal, "获取版本编号失败", err)
		}
		systems[i].NextBuildSeq = int(total) + 1
	}
	return nil
}
